package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/types"
	"github.com/haatos/simple-ci/internal/util"
	"golang.org/x/crypto/ssh"
)

func NewRunQueue(pipelineService PipelineServicer, maxRuns int64) *RunQueue {
	return &RunQueue{
		pipelineService:  pipelineService,
		OutputSSEClients: NewSSEClientMap[string](),
		StatusSSEClients: NewSSEClientMap[store.Run](),
		queue:            make(chan *store.Run, maxRuns),
		done:             make(chan struct{}),
		cancelRunMap:     NewCancelMap[int64](),
	}
}

type RunQueue struct {
	pipelineService  PipelineServicer
	OutputSSEClients *SSEClientMap[string]
	StatusSSEClients *SSEClientMap[store.Run]

	queue        chan *store.Run
	done         chan struct{}
	cancelRunMap *CancelMap[int64]

	outputCh chan string
	statusCh chan store.Run
	mu       sync.Mutex
}

func (rq *RunQueue) CancelRun(runID int64) {
	rq.cancelRunMap.Call(runID)
}

func (rq *RunQueue) Enqueue(r *store.Run) error {
	select {
	case rq.queue <- r:
		return nil
	default:
		return NewErrRunQueueFull()
	}
}

func (rq *RunQueue) Run() {
	for {
		select {
		case run := <-rq.queue:
			rq.outputCh = make(chan string)
			rq.statusCh = make(chan store.Run)

			ctx, cancel := context.WithCancel(context.Background())
			rq.cancelRunMap.AddCancel(run.RunID, cancel)

			go rq.handleOutput(ctx, run.RunID)
			go rq.handleStatus()

			if err := rq.processRun(ctx, run); err != nil {
				endedOn := time.Now().UTC()
				run.EndedOn = &endedOn
				if _, ok := err.(RunCancelError); ok {
					run.Status = store.StatusCancelled
				} else {
					run.Status = store.StatusFailed
				}
				if sqlErr := rq.pipelineService.UpdateRunEndedOn(
					context.Background(),
					run.RunID,
					run.Status,
					run.Artifacts,
					run.EndedOn,
				); sqlErr != nil {
					log.Println("err updating run status to failed:", errors.Join(err, sqlErr))
				}
				log.Println("err processing pipeline:", err)
				r, err := rq.pipelineService.GetRunByID(context.Background(), run.RunID)
				if err != nil {
					log.Println("err getting run by id")
				} else {
					run = r
					rq.statusCh <- *r
				}

				failMessage := `
=============================================
FAIL || Pipeline execution failed.
=============================================
`
				rq.outputCh <- failMessage
			}

			close(rq.outputCh)
			close(rq.statusCh)
			rq.cancelRunMap.RemoveCancel(run.RunID)
		case <-rq.done:
			close(rq.queue)
			return
		}
	}
}

func (rq *RunQueue) Shutdown() {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	select {
	case <-rq.done:
	default:
		close(rq.done)
	}
}

func (rq *RunQueue) handleOutput(ctx context.Context, runID int64) {
	for out := range rq.outputCh {
		if err := rq.pipelineService.AppendRunOutput(ctx, runID, out); err != nil {
			log.Printf("err appending run output: %+v\n", err)
		}
		rq.OutputSSEClients.SendToClients(out)
	}
}

func (rq *RunQueue) handleStatus() {
	for r := range rq.statusCh {
		rq.StatusSSEClients.SendToClients(r)
	}
}

func (rq *RunQueue) processRun(
	ctx context.Context,
	run *store.Run,
) error {
	prd, err := rq.pipelineService.GetPipelineRunData(ctx, run.RunPipelineID)
	if err != nil {
		rq.outputCh <- fmt.Sprintf("err getting pipeline/agent/credential: %+v\n", err)
		return err
	}
	workdir := time.Now().UTC().Format(internal.RunDirLayout)

	// update run status to running
	run.Status = store.StatusRunning
	startedOn := time.Now().UTC()
	run.StartedOn = &startedOn

	if err := rq.pipelineService.UpdateRunStartedOn(
		context.Background(),
		run.RunID,
		workdir,
		run.Status,
		run.StartedOn,
	); err != nil {
		rq.outputCh <- "err updating run started on"
		return err
	}

	r, err := rq.pipelineService.GetRunByID(context.Background(), run.RunID)
	if err != nil {
		rq.outputCh <- "err getting run by ID"
		return err
	}
	run = r
	rq.statusCh <- *r

	// connect to agent through SSH
	client, err := rq.connectSSH(prd.Username, prd.Hostname, prd.SSHPrivateKey)
	if err != nil {
		rq.outputCh <- "Error connection through SSH."
		return err
	}
	defer client.Close()

	// new session to clone repository
	if err := cloneRepositoryOnAgent(
		ctx, client,
		prd.Repository, prd.Workspace, workdir, r.Branch,
	); err != nil {
		rq.outputCh <- "err cloning repository on agent"
		return err
	}
	rq.outputCh <- fmt.Sprintf("Cloned repository %s\n", prd.Repository)

	// base command: cd <workdir> &&
	// new session to read pipeline script
	pipelineYaml, err := readPipelineScript(
		client,
		prd.Workspace,
		workdir,
		prd.Repository,
		prd.ScriptPath,
	)
	if err != nil {
		rq.outputCh <- "err reading pipeline script"
		return err
	}
	ps := new(types.PipelineScript)
	if err := yaml.Unmarshal(pipelineYaml, ps); err != nil {
		rq.outputCh <- "err unmarshaling pipeline yaml"
		return err
	}

	rq.outputCh <- "Parsed pipeline script. Starting pipeline execution...\n"

	if err := rq.executePipelineScript(ctx, client, prd.Repository, prd.Workspace, workdir, ps); err != nil {
		rq.outputCh <- fmt.Sprintf("err executing pipeline script: %+v\n", err)
		return err
	}

	passMessage := `
=============================================
PASS || Executed pipeline steps successfully.
=============================================
`
	rq.outputCh <- passMessage

	// update run status and output
	run.Status = store.StatusPassed
	run.EndedOn = util.AsPtr(time.Now().UTC())
	if err := rq.pipelineService.UpdateRunEndedOn(
		context.Background(),
		run.RunID,
		run.Status,
		run.Artifacts,
		run.EndedOn,
	); err != nil {
		rq.outputCh <- "err updating run ended on"
		return err
	}

	r, err = rq.pipelineService.GetRunByID(context.Background(), run.RunID)
	if err != nil {
		rq.outputCh <- "err getting run by id"
		return err
	}

	run = r
	rq.statusCh <- *r

	return nil
}

func (rq *RunQueue) connectSSH(username, hostname string, privateKey []byte) (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		rq.outputCh <- "err parsing ssh private key"
		return nil, err
	}
	auth := ssh.PublicKeys(signer)
	cc := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// connect to agent through SSH
	split := strings.Split(hostname, ":")
	if len(split) == 1 {
		hostname += ":22"
	}
	client, err := ssh.Dial("tcp", hostname, cc)
	if err != nil {
		rq.outputCh <- "err dialing ssh"
		return nil, err
	}

	rq.outputCh <- fmt.Sprintf("SSH connected to %s\n", hostname)
	return client, nil
}

func readPipelineScript(
	client *ssh.Client,
	workspace, workdir, repository, scriptPath string,
) ([]byte, error) {
	repoDir := repository[strings.LastIndex(repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	output, err := sess.Output(
		fmt.Sprintf("cd %s && cd %s && cd %s && cat %s", workspace, workdir, repoDir, scriptPath),
	)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func cloneRepositoryOnAgent(
	ctx context.Context,
	client *ssh.Client,
	repository, workspace, workdir, branch string,
) error {
	if _, _, err := runCommand(
		ctx,
		client,
		fmt.Sprintf("mkdir -p %s/%s", workspace, workdir),
		5*time.Second,
	); err != nil {
		return err
	}
	if _, _, err := runCommandInWorkdir(
		ctx,
		client,
		workspace,
		workdir,
		fmt.Sprintf("git clone -b %s %s", branch, repository),
		60*time.Second,
	); err != nil {
		return err
	}
	return nil
}

func (rq *RunQueue) executePipelineScript(
	ctx context.Context,
	client *ssh.Client,
	repository, workspace, workdir string,
	ps *types.PipelineScript,
) error {
	repoDir := repository[strings.LastIndex(repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	for _, stage := range ps.Stages {
		rq.outputCh <- fmt.Sprintf("Executing pipeline stage '%s'\n", stage.Stage)
		for _, step := range stage.Steps {
			rq.outputCh <- fmt.Sprintf("  |  Executing pipeline step '%s'\n", step.Step)
			if err := rq.executePipelineStep(
				ctx,
				client,
				time.Duration(step.TimeoutSeconds)*time.Second,
				workspace,
				workdir,
				repoDir,
				step.Script,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rq *RunQueue) executePipelineStep(
	ctx context.Context,
	client *ssh.Client,
	timeout time.Duration,
	workspace, workdir, repoDir, script string,
) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		return err
	}

	doneCh := make(chan error, 1)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		defer cancel()
		// start the step command
		cmd := fmt.Sprintf("cd %s && cd %s && cd %s && %s", workspace, workdir, repoDir, script)
		if err := sess.Start(cmd); err != nil {
			doneCh <- errors.Join(fmt.Errorf("err starting command %s", cmd), err)
			return
		}

		// scan output produced by the command an pass it to output channel and append to total output
		var wg sync.WaitGroup
		wg.Go(func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				text := scanner.Text()
				rq.outputCh <- text + "\n"
			}
		})
		wg.Go(func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				text := scanner.Text()
				rq.outputCh <- text + "\n"
			}
		})

		// wait for command to finish
		if err := sess.Wait(); err != nil {
			doneCh <- errors.Join(fmt.Errorf("err waiting for command to finish %s", cmd), err)
			return
		}

		wg.Wait()

		doneCh <- nil
	}()

	select {
	case <-timeoutCtx.Done():
		err := fmt.Errorf(
			"step execution timed out in %d seconds, script: '%s'",
			int(timeout.Seconds()),
			script,
		)
		message := err.Error()
		rq.outputCh <- message
		return err
	case <-ctx.Done():
		sess.Signal(ssh.SIGINT)
		message := "step execution cancelled by user"
		rq.outputCh <- message
		return RunCancelError{Message: message}
	case err := <-doneCh:
		return err
	}
}

func runCommandInWorkdir(
	ctx context.Context,
	client *ssh.Client,
	workspace, workdir, command string,
	timeout time.Duration,
) (string, string, error) {
	cmd := fmt.Sprintf(
		"cd %s && cd %s && %s",
		workspace,
		workdir,
		command,
	)
	return runCommand(ctx, client, cmd, timeout)
}

func runCommand(
	ctx context.Context,
	client *ssh.Client,
	command string,
	timeout time.Duration,
) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	sess, err := client.NewSession()
	if err != nil {
		log.Println("err creating new session: ", err)
	}
	defer sess.Close()
	sess.Stdout = stdout
	sess.Stderr = stderr

	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	doneCh := make(chan error, 1)

	go func() {
		doneCh <- sess.Run(command)
	}()

	select {
	case <-ctxTimeout.Done():
		return "", "", fmt.Errorf(
			"command '%s' timeout after %d seconds",
			command,
			int(timeout.Seconds()),
		)
	case <-ctx.Done():
		if err := sess.Signal(ssh.SIGINT); err != nil {
			return "", "", RunCancelError{Message: "err sending SIGINT to agent machine"}
		}
		message := fmt.Sprintf("command '%s' was cancelled by user", command)
		if err != nil {
			message += fmt.Sprintf(": err waiting for SSH terminal to close:: %+v\n", err)
		}
		return "", "", RunCancelError{Message: message}
	case err := <-doneCh:
		if err != nil {
			return "", "", err
		} else {
			return stdout.String(), stderr.String(), nil
		}
	}
}
