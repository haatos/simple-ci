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
	"golang.org/x/crypto/ssh"
)

type worker struct {
	output      string
	outputCh    chan string
	run         *store.Run
	runStatusCh chan store.Run
}

type runData struct {
	credential *store.Credential
	agent      *store.Agent
	pipeline   *store.Pipeline
	run        *store.Run
	workdir    string
}

func NewRunQueue(pipelineService PipelineServicer, maxRuns int64) *RunQueue {
	return &RunQueue{
		Queue:            make(chan *store.Run, maxRuns),
		Done:             make(chan struct{}),
		pipelineService:  pipelineService,
		OutputSSEClients: NewSSEClientMap[string](),
		StatusSSEClients: NewSSEClientMap[store.Run](),
		CancelRunMap:     NewCancelMap[int64](),
	}
}

type RunQueue struct {
	Queue chan *store.Run
	Done  chan struct{}

	pipelineService  PipelineServicer
	OutputSSEClients *SSEClientMap[string]
	StatusSSEClients *SSEClientMap[store.Run]
	CancelRunMap     *CancelMap[int64]

	mu sync.Mutex
}

func (rq *RunQueue) Run() {
	for {
		select {
		case run := <-rq.Queue:
			w := &worker{
				outputCh:    make(chan string),
				run:         run,
				runStatusCh: make(chan store.Run),
			}

			ctx, cancel := context.WithCancel(context.Background())
			rq.CancelRunMap.AddCancel(run.RunID, cancel)

			go rq.handleOutput(ctx, w)
			go rq.handleStatus(w)

			if err := rq.processRun(ctx, w); err != nil {
				endedOn := time.Now().UTC()
				run.EndedOn = &endedOn
				run.Output = &w.output
				if _, ok := err.(RunCancelError); ok {
					run.Status = store.StatusCancelled
				} else {
					run.Status = store.StatusFailed
				}
				if sqlErr := rq.pipelineService.UpdateRunEndedOn(
					context.Background(),
					run.RunID,
					run.Status,
					run.Output,
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
					w.run = r
					w.runStatusCh <- *r
				}

				failMessage := `
=============================================
FAIL || Pipeline execution failed.
=============================================
`
				w.outputCh <- failMessage
			}

			close(w.outputCh)
			close(w.runStatusCh)
			rq.CancelRunMap.RemoveCancel(run.RunID)
		case <-rq.Done:
			close(rq.Queue)
			return
		}
	}
}

func (rq *RunQueue) Shutdown() {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	select {
	case <-rq.Done:
	default:
		close(rq.Done)
	}
}

func (rq *RunQueue) handleOutput(ctx context.Context, w *worker) {
	for out := range w.outputCh {
		w.output += out
		rq.pipelineService.AppendRunOutput(ctx, w.run.RunID, out)
		rq.OutputSSEClients.SendToClients(out)
	}
}

func (rq *RunQueue) handleStatus(w *worker) {
	for r := range w.runStatusCh {
		rq.StatusSSEClients.SendToClients(r)
	}
}

func (rq *RunQueue) processRun(
	ctx context.Context,
	w *worker,
) error {
	p, a, c, err := rq.pipelineService.GetPipelineAgentAndCredential(
		context.Background(),
		w.run.RunPipelineID,
	)
	if err != nil {
		w.outputCh <- fmt.Sprintf("err getting pipeline/agent/credential: %+v\n", err)
		return err
	}
	rd := &runData{
		credential: c,
		agent:      a,
		pipeline:   p,
		run:        w.run,
		workdir:    time.Now().UTC().Format(internal.RunDirLayout),
	}

	// update run status to running
	rd.run.Status = store.StatusRunning
	startedOn := time.Now().UTC()
	rd.run.StartedOn = &startedOn

	if err := rq.pipelineService.UpdateRunStartedOn(
		context.Background(),
		rd.run.RunID,
		rd.workdir,
		rd.run.Status,
		rd.run.StartedOn,
	); err != nil {
		w.outputCh <- "err updating run started on"
		return err
	}

	r, err := rq.pipelineService.GetRunByID(context.Background(), w.run.RunID)
	if err != nil {
		w.outputCh <- "err getting run by ID"
		return err
	}
	w.run = r
	w.runStatusCh <- *r

	signer, err := ssh.ParsePrivateKey(c.SSHPrivateKey)
	if err != nil {
		w.outputCh <- "err parsing ssh private key"
		return err
	}
	auth := ssh.PublicKeys(signer)
	cc := &ssh.ClientConfig{
		User:            rd.credential.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// connect to agent through SSH
	hostname := rd.agent.Hostname
	split := strings.Split(hostname, ":")
	if len(split) == 1 {
		hostname += ":22"
	}
	client, err := ssh.Dial("tcp", hostname, cc)
	if err != nil {
		w.outputCh <- "err dialing ssh"
		return err
	}
	defer client.Close()
	w.outputCh <- fmt.Sprintf("SSH connected to %s\n", hostname)

	// new session to clone repository
	if err := cloneRepositoryOnAgent(ctx, client, rd); err != nil {
		w.outputCh <- "err cloning repository on agent"
		return err
	}
	w.outputCh <- fmt.Sprintf("Cloned repository %s\n", rd.pipeline.Repository)

	// base command: cd <workdir> &&
	// new session to read pipeline script
	pipelineYaml, err := readPipelineScript(
		client,
		rd.agent.Workspace,
		rd.workdir,
		rd.pipeline.Repository,
		rd.pipeline.ScriptPath,
	)
	if err != nil {
		w.outputCh <- "err reading pipeline script"
		return err
	}
	ps := new(types.PipelineScript)
	if err := yaml.Unmarshal(pipelineYaml, ps); err != nil {
		w.outputCh <- "err unmarshaling pipeline yaml"
		return err
	}

	w.outputCh <- "Parsed pipeline script. Starting pipeline execution...\n"

	if err := executePipelineScript(ctx, client, w, rd, ps); err != nil {
		w.outputCh <- fmt.Sprintf("err executing pipeline script: %+v\n", err)
		return err
	}

	passMessage := `
=============================================
PASS || Executed pipeline steps successfully.
=============================================
`
	w.outputCh <- passMessage

	// update run status and output
	rd.run.Output = &w.output
	rd.run.Status = store.StatusPassed
	endedOn := time.Now().UTC()
	rd.run.EndedOn = &endedOn
	if err := rq.pipelineService.UpdateRunEndedOn(
		context.Background(),
		rd.run.RunID,
		rd.run.Status,
		rd.run.Output,
		rd.run.Artifacts,
		rd.run.EndedOn,
	); err != nil {
		w.outputCh <- "err updating run ended on"
		return err
	}

	r, err = rq.pipelineService.GetRunByID(context.Background(), w.run.RunID)
	if err != nil {
		w.outputCh <- "err getting run by id"
		return err
	}

	w.run = r
	w.runStatusCh <- *r

	return nil
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

func cloneRepositoryOnAgent(ctx context.Context, client *ssh.Client, runData *runData) error {
	if _, _, err := runCommand(
		ctx,
		client,
		fmt.Sprintf("mkdir -p %s/%s", runData.agent.Workspace, runData.workdir),
		10*time.Second,
	); err != nil {
		return err
	}
	if _, _, err := runCommandInWorkdir(
		ctx,
		client,
		runData.agent.Workspace,
		runData.workdir,
		fmt.Sprintf("git clone -b %s %s", runData.run.Branch, runData.pipeline.Repository),
		30*time.Second,
	); err != nil {
		return err
	}
	return nil
}

func executePipelineScript(
	ctx context.Context,
	client *ssh.Client,
	w *worker,
	rd *runData,
	ps *types.PipelineScript,
) error {
	repoDir := rd.pipeline.Repository[strings.LastIndex(rd.pipeline.Repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	for _, stage := range ps.Stages {
		w.outputCh <- fmt.Sprintf("Executing pipeline stage '%s'\n", stage.Stage)
		for _, step := range stage.Steps {
			w.outputCh <- fmt.Sprintf(" | Executing pipeline step '%s'\n", step.Step)
			if err := executePipelineStep(
				ctx,
				client,
				w,
				time.Duration(step.TimeoutSeconds)*time.Second,
				rd.agent.Workspace,
				rd.workdir,
				repoDir,
				step.Script,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func executePipelineStep(
	ctx context.Context,
	client *ssh.Client,
	w *worker,
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
				w.outputCh <- text + "\n"
				w.output += text + "\n"
			}
		})
		wg.Go(func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				text := scanner.Text()
				w.outputCh <- text + "\n"
				w.output += text + "\n"
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
			"step execution timed out in %d seconds, script: %s",
			int(timeout.Seconds()),
			script,
		)
		message := err.Error()
		w.outputCh <- message
		return err
	case <-ctx.Done():
		sess.Signal(ssh.SIGINT)
		message := "step execution cancelled by user"
		w.outputCh <- message
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
		sess.Signal(ssh.SIGINT)
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
