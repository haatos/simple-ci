package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func NewRunQueue(
	pipelineStore store.PipelineStore,
	runStore store.RunStore,
	encrypter security.Encrypter,
	maxRuns int64,
) *RunQueue {
	return &RunQueue{
		pipelineStore:    pipelineStore,
		runStore:         runStore,
		aesEncrypter:     encrypter,
		OutputSSEClients: NewSSEClientMap[string](),
		StatusSSEClients: NewSSEClientMap[store.Run](),
		queue:            make(chan *store.Run, maxRuns),
		done:             make(chan struct{}),
		cancelRunMap:     NewCancelMap[int64](),
	}
}

type RunQueue struct {
	pipelineStore    store.PipelineStore
	runStore         store.RunStore
	aesEncrypter     security.Encrypter
	OutputSSEClients *SSEClientMap[string]
	StatusSSEClients *SSEClientMap[store.Run]

	queue        chan *store.Run
	done         chan struct{}
	cancelRunMap *CancelMap[int64]

	outputCh chan string
	statusCh chan store.Run
	mu       sync.Mutex

	pathSep, mkdirCmd, chainOp string
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
				rq.handleProcessingFailed(err, run)
			}

			<-rq.done

			close(rq.done)
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
		if err := rq.runStore.AppendPipelineRunOutput(ctx, runID, out); err != nil {
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
	prd, err := rq.pipelineStore.ReadPipelineRunData(ctx, run.RunPipelineID)
	if err != nil {
		return fmt.Errorf("err getting pipeline/agent/credential data: %+w", err)
	}
	workdir := time.Now().UTC().Format(internal.RunDirLayout)

	switch prd.OSType {
	case "unix":
		rq.pathSep = "/"
		rq.mkdirCmd = "mkdir -p"
		rq.chainOp = " && "
	case "windows":
		rq.pathSep = "\\"
		rq.mkdirCmd = "mkdir"
		rq.chainOp = " & "
	case "darwin":
		rq.pathSep = "/"
		rq.mkdirCmd = "mkdir -p"
		rq.chainOp = "; "
	default:
		log.Println("invalid OS type: " + prd.OSType)
	}

	// update run status to running
	run.Status = store.StatusRunning
	startedOn := time.Now().UTC()
	run.StartedOn = &startedOn

	if err := rq.runStore.UpdatePipelineRunStartedOn(
		context.Background(),
		run.RunID,
		workdir,
		run.Status,
		run.StartedOn,
	); err != nil {
		return fmt.Errorf("err updating run started on: %+w", err)
	}

	r, err := rq.runStore.ReadRunByID(context.Background(), run.RunID)
	if err != nil {
		return fmt.Errorf("err getting run by id: %+w", err)
	}
	run = r
	rq.statusCh <- *r

	if prd.Hostname != "localhost" {
		// connect to agent through SSH
		client, err := rq.connectSSH(*prd.Username, prd.Hostname, prd.SSHPrivateKey)
		if err != nil {
			rq.outputCh <- "Error connection through SSH."
			return err
		}
		defer client.Close()

		// new session to clone repository
		if err := rq.cloneRepositoryOnAgent(
			ctx, client,
			prd.Repository, prd.Workspace, workdir, r.Branch,
		); err != nil {
			return fmt.Errorf("err cloning repository on agent: %+w", err)
		}

		// base command: cd <workdir> &&
		// new session to read pipeline script
		ps, err := rq.readPipelineScriptOnAgent(
			client,
			prd.Workspace,
			workdir,
			prd.Repository,
			prd.ScriptPath,
		)
		if err != nil {
			return fmt.Errorf("err reading pipeline script: %+w", err)
		}

		if err := rq.executePipelineScriptOnAgent(ctx, client, prd.Repository, prd.Workspace, workdir, ps); err != nil {
			return err
		}

		passMessage := `
=============================================
PASS || Executed pipeline steps successfully.
=============================================
`
		rq.outputCh <- passMessage

		// update run status
		run.Status = store.StatusPassed
		run.EndedOn = util.AsPtr(time.Now().UTC())
		if err := rq.runStore.UpdatePipelineRunEndedOn(
			context.Background(),
			run.RunID,
			run.Status,
			run.Artifacts,
			run.EndedOn,
		); err != nil {
			return fmt.Errorf("err updating run ended on: %+w", err)
		}

		r, err = rq.runStore.ReadRunByID(context.Background(), run.RunID)
		if err != nil {
			return fmt.Errorf("err getting run by id: %+w", err)
		}

		run = r
		rq.statusCh <- *r
		rq.done <- struct{}{}
	} else {
		// run on the controller machine
		if err := rq.cloneRepositoryOnController(
			ctx, prd.Repository, prd.Workspace, workdir, run.Branch,
		); err != nil {
			return fmt.Errorf("err cloning repository on controller: %+v", err)
		}

		ps, err := rq.readPipelineScriptOnController(
			prd.Workspace,
			workdir,
			prd.Repository,
			prd.ScriptPath,
		)
		if err != nil {
			return fmt.Errorf("err reading pipeline script: %+w", err)
		}

		if err := rq.executePipelineScriptOnController(ctx, prd.Repository, prd.Workspace, workdir, ps); err != nil {
			return err
		}

		passMessage := `
=============================================
PASS || Executed pipeline steps successfully.
=============================================
`
		rq.outputCh <- passMessage

		// update run status
		run.Status = store.StatusPassed
		run.EndedOn = util.AsPtr(time.Now().UTC())
		if err := rq.runStore.UpdatePipelineRunEndedOn(
			context.Background(),
			run.RunID,
			run.Status,
			run.Artifacts,
			run.EndedOn,
		); err != nil {
			return fmt.Errorf("err updating run ended on: %+w", err)
		}

		r, err = rq.runStore.ReadRunByID(context.Background(), run.RunID)
		if err != nil {
			return fmt.Errorf("err getting run by id: %+w", err)
		}

		run = r
		rq.statusCh <- *r
		rq.done <- struct{}{}
	}

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

func (rq *RunQueue) readPipelineScriptOnAgent(
	client *ssh.Client,
	workspace, workdir, repository, scriptPath string,
) (*PipelineScript, error) {
	repoDir := repository[strings.LastIndex(repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return nil, err
	}
	parts := []string{workspace, workdir, repoDir, scriptPath}
	remoteFile, err := sftpClient.Open(strings.Join(parts, rq.pathSep))
	if err != nil {
		return nil, err
	}
	remoteFileInfo, err := remoteFile.Stat()
	if err != nil {
		return nil, err
	}
	b := make([]byte, remoteFileInfo.Size())
	if _, err := remoteFile.Read(b); err != nil {
		return nil, err
	}

	ps := new(PipelineScript)
	if err := yaml.Unmarshal(b, ps); err != nil {
		return nil, fmt.Errorf("err unmarshaling pipeline yaml: %+w", err)
	}

	rq.outputCh <- "Parsed pipeline script. Starting pipeline execution...\n"

	return ps, nil
}

func (rq *RunQueue) readPipelineScriptOnController(
	workspace, workdir, repository, scriptPath string,
) (*PipelineScript, error) {
	repoDir := repository[strings.LastIndex(repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	parts := []string{workspace, workdir, repoDir, scriptPath}
	b, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		return nil, err
	}
	ps := new(PipelineScript)
	if err := yaml.Unmarshal(b, ps); err != nil {
		return nil, fmt.Errorf("err unmarshaling pipeline yaml: %+v", err)
	}

	rq.outputCh <- "Parsed pipeline script. Starting pipeline execution...\n"

	return ps, nil
}

func (rq *RunQueue) cloneRepositoryOnAgent(
	ctx context.Context,
	client *ssh.Client,
	repository, workspace, workdir, branch string,
) error {
	workingDir := strings.Join([]string{workspace, workdir}, rq.pathSep)
	cmds := []string{
		fmt.Sprintf("%s %s", rq.mkdirCmd, workingDir),
		fmt.Sprintf("cd %s", workingDir),
		fmt.Sprintf("git clone -b %s %s", branch, repository),
	}
	cmd := strings.Join(cmds, rq.chainOp)
	if _, _, err := runCommand(ctx, client, cmd, 60*time.Second); err != nil {
		return err
	}

	rq.outputCh <- fmt.Sprintf("Cloned repository %s\n", repository)

	return nil
}

func (rq *RunQueue) cloneRepositoryOnController(
	ctx context.Context,
	repository, workspace, workdir, branch string,
) error {
	workingDir := filepath.Join(workspace, workdir)
	if err := os.MkdirAll(workingDir, os.ModePerm); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "clone", "-b", branch, repository)
	cmd.Dir = workingDir
	if err := cmd.Run(); err != nil {
		return err
	}

	rq.outputCh <- fmt.Sprintf("Cloned repository %s\n", repository)

	return nil
}

func (rq *RunQueue) executePipelineScriptOnAgent(
	ctx context.Context,
	client *ssh.Client,
	repository, workspace, workdir string,
	ps *PipelineScript,
) error {
	repoDir := repository[strings.LastIndex(repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	for _, stage := range ps.Stages {
		rq.outputCh <- fmt.Sprintf("    |    Executing pipeline stage '%s'\n", stage.Stage)
		si := 0
		for si < len(stage.Steps) {
			cur := stage.Steps[si]
			if cur.Parallel {
				steps := []Step{cur}
				sii := si + 1
				for sii < len(stage.Steps) && stage.Steps[sii].Parallel {
					parallelStep := stage.Steps[sii]
					steps = append(steps, parallelStep)
					sii++
				}
				var wg sync.WaitGroup
				errs := make([]error, 0)
				mu := sync.Mutex{}
				for _, step := range steps {
					wg.Go(func() {
						rq.outputCh <- fmt.Sprintf("    |    |    Executing step '%s'\n", step.Step)
						if err := rq.executePipelineStepOnAgent(
							ctx,
							client,
							time.Duration(step.TimeoutSeconds)*time.Second,
							workspace,
							workdir,
							repoDir,
							step.Script,
						); err != nil {
							mu.Lock()
							defer mu.Unlock()
							errs = append(errs, err)
							return
						}
						rq.outputCh <- fmt.Sprintf("    |    |    STEP '%s' PASSED\n", step.Step)
					})
				}
				wg.Wait()
				if len(errs) > 0 {
					return errors.Join(errs...)
				}
				si = sii
			} else {
				si++
				rq.outputCh <- fmt.Sprintf("    |    |    Executing step '%s'\n", cur.Step)
				if err := rq.executePipelineStepOnAgent(
					ctx,
					client,
					time.Duration(cur.TimeoutSeconds)*time.Second,
					workspace,
					workdir,
					repoDir,
					cur.Script,
				); err != nil {
					return err
				}
				rq.outputCh <- fmt.Sprintf("    |    |    STEP '%s' PASSED\n", cur.Step)
			}
		}
		rq.outputCh <- fmt.Sprintf("    |    STAGE '%s' PASSED\n", stage.Stage)
	}
	return nil
}

func (rq *RunQueue) executePipelineScriptOnController(
	ctx context.Context,
	repository, workspace, workdir string,
	ps *PipelineScript,
) error {
	repoDir := repository[strings.LastIndex(repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	for _, stage := range ps.Stages {
		rq.outputCh <- fmt.Sprintf("    |    Executing pipeline stage '%s'\n", stage.Stage)
		si := 0
		for si < len(stage.Steps) {
			cur := stage.Steps[si]
			if cur.Parallel {
				steps := []Step{cur}
				sii := si + 1
				for sii < len(stage.Steps) && stage.Steps[sii].Parallel {
					parallelStep := stage.Steps[sii]
					steps = append(steps, parallelStep)
					sii++
				}
				var wg sync.WaitGroup
				errs := make([]error, 0)
				mu := sync.Mutex{}
				for _, step := range steps {
					wg.Go(func() {
						rq.outputCh <- fmt.Sprintf("    |    |    Executing step '%s'\n", step.Step)
						if err := rq.executePipelineStepOnController(
							ctx,
							time.Duration(step.TimeoutSeconds)*time.Second,
							workspace,
							workdir,
							repoDir,
							step.Script,
						); err != nil {
							mu.Lock()
							defer mu.Unlock()
							errs = append(errs, err)
							return
						}
						rq.outputCh <- fmt.Sprintf("    |    |    STEP '%s' PASSED\n", step.Step)
					})
				}
				wg.Wait()
				if len(errs) > 0 {
					return errors.Join(errs...)
				}
				si = sii
			} else {
				si++
				rq.outputCh <- fmt.Sprintf("    |    |    Executing step '%s'\n", cur.Step)
				if err := rq.executePipelineStepOnController(
					ctx,
					time.Duration(cur.TimeoutSeconds)*time.Second,
					workspace,
					workdir,
					repoDir,
					cur.Script,
				); err != nil {
					return err
				}
				rq.outputCh <- fmt.Sprintf("    |    |    STEP '%s' PASSED\n", cur.Step)
			}
		}
		rq.outputCh <- fmt.Sprintf("    |    STAGE '%s' PASSED\n", stage.Stage)
	}
	return nil
}

func (rq *RunQueue) executePipelineStepOnAgent(
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

	baseCmds := []string{
		"cd " + workspace,
		"cd " + workdir,
		"cd " + repoDir,
	}

	doneCh := make(chan error, 1)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		defer cancel()
		// start the step command
		cmds := append(baseCmds, script)
		cmd := strings.Join(cmds, rq.chainOp)
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

func (rq *RunQueue) executePipelineStepOnController(
	ctx context.Context,
	timeout time.Duration,
	workspace, workdir, repoDir, script string,
) error {
	doneCh := make(chan error, 1)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		defer cancel()
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "powershell", "-Command", script)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", script)
		}
		cmd.Dir = filepath.Join(workspace, workdir, repoDir)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			doneCh <- errors.Join(fmt.Errorf("err getting stdout pipe for command %s", cmd), err)
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			doneCh <- errors.Join(fmt.Errorf("err getting stderr pipe for command %s", cmd), err)
			return
		}

		if err := cmd.Start(); err != nil {
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
		if err := cmd.Wait(); err != nil {
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
		message := "step execution cancelled by user"
		rq.outputCh <- message
		return RunCancelError{Message: message}
	case err := <-doneCh:
		return err
	}
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

func (rq *RunQueue) handleProcessingFailed(err error, run *store.Run) {
	endedOn := time.Now().UTC()
	run.EndedOn = &endedOn
	if _, ok := err.(RunCancelError); ok {
		run.Status = store.StatusCancelled
	} else {
		run.Status = store.StatusFailed
	}
	if sqlErr := rq.runStore.UpdatePipelineRunEndedOn(
		context.Background(),
		run.RunID,
		run.Status,
		run.Artifacts,
		run.EndedOn,
	); sqlErr != nil {
		log.Println("err updating run status to failed:", errors.Join(err, sqlErr))
	}
	log.Println("err processing pipeline:", err)
	r, sqlErr := rq.runStore.ReadRunByID(context.Background(), run.RunID)
	if sqlErr != nil {
		log.Println("err getting run by id", errors.Join(err, sqlErr))
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
	rq.done <- struct{}{}
}
