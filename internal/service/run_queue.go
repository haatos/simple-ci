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

func NewRunQueue(maxRuns int64) *RunQueue {
	return &RunQueue{
		Queue:            make(chan *store.Run, maxRuns),
		Done:             make(chan struct{}),
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

func (rq *RunQueue) Run(pipelineService PipelineServicer) {
	for {
		select {
		case run := <-rq.Queue:
			w := &Worker{
				OutputCh:    make(chan string),
				Run:         run,
				RunStatusCh: make(chan store.Run),
			}

			ctx, cancel := context.WithCancel(context.Background())
			rq.CancelRunMap.AddCancel(run.RunID, cancel)

			go rq.handleStatus(w)

			if err := rq.processRun(ctx, pipelineService, w); err != nil {
				run.EndedOn = util.AsPtr(time.Now().UTC())
				run.Output = &w.Output
				if rcErr, ok := err.(RunCancelError); ok {
					run.Status = store.StatusCancelled
					rq.storeOutput(ctx, run.RunID, rcErr.Message)
				} else {
					run.Status = store.StatusFailed
				}
				if sqlErr := pipelineService.UpdateRunEndedOn(
					context.Background(),
					run.RunID,
					run.Status,
					run.Output,
					run.Artifacts,
					run.EndedOn,
				); sqlErr != nil {
					rq.storeOutput(
						ctx,
						run.RunID,
						fmt.Sprintf("Error updating run status to failed: %+v\n", sqlErr),
					)
				}
				r, err := pipelineService.GetRunByID(context.Background(), run.RunID)
				if err != nil {
					rq.storeOutput(
						ctx,
						run.RunID,
						fmt.Sprintf("Error getting run by ID: %+v\n", err),
					)
				} else {
					w.Run = r
					w.RunStatusCh <- *r
				}
			}
			close(w.OutputCh)
			close(w.RunStatusCh)
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

func (rq *RunQueue) handleStatus(w *Worker) {
	for r := range w.RunStatusCh {
		rq.StatusSSEClients.SendToClients(r)
	}
}

func (rq *RunQueue) processRun(
	ctx context.Context,
	pipelineService PipelineServicer,
	w *Worker,
) error {
	p, a, c, err := pipelineService.GetPipelineAgentAndCredential(
		context.Background(),
		w.Run.RunPipelineID,
	)
	if err != nil {
		log.Printf("err getting pipeline/agent/credential data: %+v\n", err)
		rq.storeOutput(ctx, w.Run.RunID, "Error retrieving pipeline, agent or credential data")
		return err
	}
	workdir := time.Now().UTC().Format(internal.RunDirLayout)

	// update run status to running
	w.Run.Status = store.StatusRunning
	w.Run.StartedOn = util.AsPtr(time.Now().UTC())

	if err := pipelineService.UpdateRunStartedOn(
		context.Background(),
		w.Run.RunID,
		workdir,
		w.Run.Status,
		w.Run.StartedOn,
	); err != nil {
		rq.storeOutput(ctx, w.Run.RunID, "Error updating run started on time and status")
		return err
	}

	r, err := pipelineService.GetRunByID(context.Background(), w.Run.RunID)
	if err != nil {
		rq.storeOutput(ctx, w.Run.RunID, "Error getting run by ID")
		w.OutputCh <- "err getting run by ID"
		return err
	}
	w.Run = r
	w.RunStatusCh <- *r

	hostname := a.Hostname
	split := strings.Split(hostname, ":")
	if len(split) == 1 {
		hostname += ":22"
	}
	sshService := &SSHService{
		username:   c.Username,
		host:       hostname,
		privateKey: c.SSHPrivateKey,
	}

	// create working directory
	// ==============================================================
	cmd := fmt.Sprintf("mkdir -p %s/%s", a.Workspace, workdir)
	out, err := sshService.RunCommand(cmd)
	if err != nil {
		return err
	}
	for output := range out {
		rq.pipelineService.AppendRunOutput(ctx, w.Run.RunID, output)
		rq.OutputSSEClients.SendToClients(output)
	}
	// ==============================================================

	// clone repository
	// ==============================================================
	cmd = fmt.Sprintf("git clone -b %s %s", r.Branch, p.Repository)
	out, err = sshService.RunCommand(cmd)
	if err != nil {
		return err
	}
	for output := range out {
		rq.pipelineService.AppendRunOutput(ctx, w.Run.RunID, output)
		rq.OutputSSEClients.SendToClients(output)
	}
	// ==============================================================

	// read and parse pipeline yaml
	// ==============================================================
	repoDir := p.Repository[strings.LastIndex(p.Repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	cmd = fmt.Sprintf(
		"cd %s && cd %s && cd %s && cat %s",
		a.Workspace,
		workdir,
		repoDir,
		p.ScriptPath,
	)
	out, err = sshService.RunCommand(cmd)
	if err != nil {
		return err
	}
	var yamlScript string
	for output := range out {
		yamlScript = output
	}
	ps := new(types.PipelineScript)
	if err := yaml.Unmarshal([]byte(yamlScript), ps); err != nil {
		rq.storeOutput(ctx, r.RunID, "Error parsing pipeline yaml")
		return err
	}
	// ==============================================================

	// run pipeline script
	// =============================================================================================
	for _, stage := range ps.Stages {
		rq.OutputSSEClients.SendToClients(fmt.Sprintf("Executing pipeline stage '%s'", stage.Stage))
		for _, step := range stage.Steps {
			out, err := sshService.RunCommand(step.Script)
			if err != nil {
				return err
			}
			rq.storeOutput(
				ctx, r.RunID,
				fmt.Sprintf("Executing pipeline step '%s': %s", step.Step, step.Script),
			)
			rq.handleOutput(ctx, out, w.Run.RunID)
		}
	}
	// =============================================================================================

	rq.storeOutput(ctx, r.RunID, "\n=============================================\n")
	rq.storeOutput(ctx, r.RunID, "PASS || Executed pipeline steps successfully.\n")
	rq.storeOutput(ctx, r.RunID, "=============================================\n")

	// update run status and output
	r.Output = &w.Output
	r.Status = store.StatusPassed
	r.EndedOn = util.AsPtr(time.Now().UTC())
	if err := pipelineService.UpdateRunEndedOn(
		context.Background(),
		r.RunID,
		r.Status,
		r.Output,
		r.Artifacts,
		r.EndedOn,
	); err != nil {
		rq.storeOutput(ctx, r.RunID, "Error occurred while updating run ended on time and status")
		return err
	}

	r, err = pipelineService.GetRunByID(context.Background(), w.Run.RunID)
	if err != nil {
		rq.storeOutput(ctx, r.RunID, "Error occurred while getting run by ID")
		return err
	}

	w.Run = r
	w.RunStatusCh <- *r

	return nil
}

func (rq *RunQueue) handleOutput(ctx context.Context, out chan string, runID int64) {
	for output := range out {
		rq.storeOutput(ctx, runID, output)
	}
}

func (rq *RunQueue) storeOutput(ctx context.Context, runID int64, output string) {
	rq.OutputSSEClients.SendToClients(output)
	rq.pipelineService.AppendRunOutput(ctx, runID, output)
}

func ReadPipelineScript(
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

func CloneRepositoryOnAgent(ctx context.Context, client *ssh.Client, runData *RunData) error {
	if _, _, err := runCommand(
		ctx,
		client,
		fmt.Sprintf("mkdir -p %s/%s", runData.Agent.Workspace, runData.Workdir),
		10*time.Second,
	); err != nil {
		return err
	}
	if _, _, err := runCommandInWorkdir(
		ctx,
		client,
		runData.Agent.Workspace,
		runData.Workdir,
		fmt.Sprintf("git clone -b %s %s", runData.Run.Branch, runData.Pipeline.Repository),
		30*time.Second,
	); err != nil {
		return err
	}
	return nil
}

func ExecutePipelineScript(
	ctx context.Context,
	client *ssh.Client,
	w *Worker,
	rd *RunData,
	ps *types.PipelineScript,
) error {
	repoDir := rd.Pipeline.Repository[strings.LastIndex(rd.Pipeline.Repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	for _, stage := range ps.Stages {
		for _, step := range stage.Steps {
			if err := executePipelineStep(
				ctx,
				client,
				w,
				time.Duration(step.TimeoutSeconds)*time.Second,
				rd.Agent.Workspace,
				rd.Workdir,
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
	w *Worker,
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
				w.OutputCh <- text + "\n"
				w.Output += text + "\n"
			}
		})
		wg.Go(func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				text := scanner.Text()
				w.OutputCh <- text + "\n"
				w.Output += text + "\n"
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
		w.OutputCh <- message
		return err
	case <-ctx.Done():
		sess.Signal(ssh.SIGINT)
		message := "step execution cancelled by user"
		w.OutputCh <- message
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
