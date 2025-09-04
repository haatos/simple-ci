package handler

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
	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/types"
	"golang.org/x/crypto/ssh"
)

type Worker struct {
	Output   string
	OutputCh chan string
	Run      *store.Run
	RunCh    chan store.Run
}

type RunData struct {
	Credential *store.Credential
	Agent      *store.Agent
	Pipeline   *store.Pipeline
	Run        *store.Run
	Workdir    string
}

func handleOutput(outputClients *SSEClientMap[string], w *Worker) {
	for out := range w.OutputCh {
		w.Output += out
		outputClients.SendToClients(w.Run.RunID, out)
	}
}

func handleStatus(statusClients *SSEClientMap[store.Run], w *Worker) {
	for r := range w.RunCh {
		statusClients.SendToClients(w.Run.RunID, r)
	}
}

func RunWorkers(
	pipelineService service.PipelineServicer,
	runChan chan *store.Run,
	outputClients *SSEClientMap[string],
	statusClients *SSEClientMap[store.Run],
	cancelRunMap *CancelMap[int64],
) {
	for run := range runChan {
		w := &Worker{OutputCh: make(chan string), Run: run, RunCh: make(chan store.Run)}
		outputClients.AddMap(run.RunID)
		statusClients.AddMap(run.RunID)

		go handleOutput(outputClients, w)
		go handleStatus(statusClients, w)
		ctx, cancel := context.WithCancel(context.Background())
		cancelRunMap.AddCancel(run.RunID, cancel)

		if err := processRun(ctx, pipelineService, w); err != nil {
			endedOn := time.Now().UTC()
			run.EndedOn = &endedOn
			run.Output = &w.Output
			if rcErr, ok := err.(RunCancelError); ok {
				run.Status = store.StatusCancelled
				w.OutputCh <- rcErr.Message
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
				log.Println("err updating run status to failed:", errors.Join(err, sqlErr))
			}
			log.Println("err processing pipeline:", err)
			r, err := pipelineService.GetRunByID(context.Background(), run.RunID)
			if err != nil {
				log.Println("err getting run by id")
			} else {
				w.Run = r
				w.RunCh <- *r
			}
		}
		cancelRunMap.RemoveCancel(run.RunID)
	}
}

func processRun(
	ctx context.Context,
	pipelineService service.PipelineServicer,
	w *Worker,
) error {
	p, a, c, err := pipelineService.GetPipelineAgentAndCredential(
		context.Background(),
		w.Run.RunPipelineID,
	)
	if err != nil {
		w.OutputCh <- fmt.Sprintf("err getting pipeline/agent/credential: %+v\n", err)
		return err
	}
	rd := &RunData{
		Credential: c,
		Agent:      a,
		Pipeline:   p,
		Run:        w.Run,
		Workdir:    time.Now().UTC().Format(internal.RunDirLayout),
	}

	// update run status to running
	rd.Run.Status = store.StatusRunning
	startedOn := time.Now().UTC()
	rd.Run.StartedOn = &startedOn

	if err := pipelineService.UpdateRunStartedOn(
		context.Background(),
		rd.Run.RunID,
		rd.Workdir,
		rd.Run.Status,
		rd.Run.StartedOn,
	); err != nil {
		w.OutputCh <- "err updating run started on"
		return err
	}

	r, err := pipelineService.GetRunByID(context.Background(), w.Run.RunID)
	if err != nil {
		w.OutputCh <- "err getting run by ID"
		return err
	}
	w.Run = r
	w.RunCh <- *r

	signer, err := ssh.ParsePrivateKey(c.SSHPrivateKey)
	if err != nil {
		w.OutputCh <- "err parsing ssh private key"
		return err
	}
	auth := ssh.PublicKeys(signer)
	cc := &ssh.ClientConfig{
		User:            rd.Credential.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// connect to agent through SSH
	hostname := rd.Agent.Hostname
	split := strings.Split(hostname, ":")
	if len(split) == 1 {
		hostname += ":22"
	}
	client, err := ssh.Dial("tcp", hostname, cc)
	if err != nil {
		w.OutputCh <- "err dialing ssh"
		return err
	}
	defer client.Close()
	w.OutputCh <- fmt.Sprintf("SSH connected to %s\n", hostname)

	// new session to clone repository
	if err := cloneRepositoryOnAgent(ctx, client, rd); err != nil {
		w.OutputCh <- "err cloning repository on agent"
		return err
	}
	w.OutputCh <- fmt.Sprintf("Cloned repository %s\n", rd.Pipeline.Repository)

	// base command: cd <workdir> &&
	// new session to read pipeline script
	pipelineYaml, err := readPipelineScript(
		client,
		rd.Agent.Workspace,
		rd.Workdir,
		rd.Pipeline.Repository,
		rd.Pipeline.ScriptPath,
	)
	if err != nil {
		w.OutputCh <- "err reading pipeline script"
		return err
	}
	ps := new(types.PipelineScript)
	if err := yaml.Unmarshal(pipelineYaml, ps); err != nil {
		w.OutputCh <- "err unmarshaling pipeline yaml"
		return err
	}

	w.OutputCh <- "Parsed pipeline script. Starting pipeline execution...\n"

	if err := executePipelineScript(ctx, client, w, rd, ps); err != nil {
		w.OutputCh <- fmt.Sprintf("err executing pipeline script: %+v\n", err)
		return err
	}

	w.OutputCh <- "Executed pipeline steps successfully.\n"

	// update run status and output
	rd.Run.Output = &w.Output
	rd.Run.Status = store.StatusPassed
	endedOn := time.Now().UTC()
	rd.Run.EndedOn = &endedOn
	if err := pipelineService.UpdateRunEndedOn(
		context.Background(),
		rd.Run.RunID,
		rd.Run.Status,
		rd.Run.Output,
		rd.Run.Artifacts,
		rd.Run.EndedOn,
	); err != nil {
		w.OutputCh <- "err updating run ended on"
		return err
	}

	r, err = pipelineService.GetRunByID(context.Background(), w.Run.RunID)
	if err != nil {
		w.OutputCh <- "err getting run by id"
		return err
	}
	w.Run = r
	w.RunCh <- *r

	// manage artifacts??

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

func cloneRepositoryOnAgent(ctx context.Context, client *ssh.Client, runData *RunData) error {
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

func executePipelineScript(
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
