package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type PipelineWriter interface {
	CreatePipeline(
		context.Context,
		int64,
		string, string, string, string,
	) (*store.Pipeline, error)
	UpdatePipeline(context.Context, int64, int64, string, string, string, string) error
	UpdatePipelineSchedule(context.Context, int64, *string, *string, *string) error
	UpdatePipelineScheduleJobID(context.Context, int64, *string) error
	DeletePipeline(context.Context, int64) error
}

type PipelineReader interface {
	ReadPipelineByID(context.Context, int64) (*store.Pipeline, error)
	ReadPipelineRunData(context.Context, int64) (*store.PipelineRunData, error)
	ListPipelines(context.Context) ([]*store.Pipeline, error)
	ListScheduledPipelines(context.Context) ([]*store.Pipeline, error)
}

type PipelineStore interface {
	PipelineWriter
	PipelineReader
}

type RunWriter interface {
	CreatePipelineRun(context.Context, int64, string) (*store.Run, error)
	UpdatePipelineRunStartedOn(context.Context, int64, string, store.RunStatus, *time.Time) error
	UpdatePipelineRunEndedOn(context.Context, int64, store.RunStatus, *string, *time.Time) error
	AppendPipelineRunOutput(context.Context, int64, string) error
	DeletePipelineRun(context.Context, int64) error
}

type RunReader interface {
	ReadRunByID(context.Context, int64) (*store.Run, error)
	ListPipelineRuns(context.Context, int64) ([]store.Run, error)
	ListLatestPipelineRuns(context.Context, int64, int64) ([]store.Run, error)
	ListPipelineRunsPaginated(context.Context, int64, int64, int64) ([]store.Run, error)
	CountPipelineRuns(context.Context, int64) (int64, error)
}

type RunStore interface {
	RunWriter
	RunReader
}

type PipelineService struct {
	pipelineStore   PipelineStore
	runStore        RunStore
	credentialStore CredentialStore
	agentService    AgentStore
	apiKeyStore     APIKeyStore
	scheduler       gocron.Scheduler
	aesEncrypter    security.Encrypter

	mu     sync.Mutex
	queues map[int64]*RunQueue
}

func NewPipelineService(
	pipelineStore PipelineStore,
	runStore RunStore,
	credentialStore CredentialStore,
	agentStore AgentStore,
	apiKeyStore APIKeyStore,
	scheduler gocron.Scheduler,
	aesEncrypter security.Encrypter,
) *PipelineService {
	return &PipelineService{
		pipelineStore:   pipelineStore,
		runStore:        runStore,
		credentialStore: credentialStore,
		agentService:    agentStore,
		apiKeyStore:     apiKeyStore,
		scheduler:       scheduler,
		aesEncrypter:    aesEncrypter,
		queues:          make(map[int64]*RunQueue),
	}
}

func (s *PipelineService) InitializeRunQueues(ctx context.Context) error {
	pipelines, err := s.ListPipelines(ctx)
	if err != nil {
		return err
	}

	ids := make([]int64, len(pipelines))
	for i, p := range pipelines {
		ids[i] = p.PipelineID
	}

	s.AddRunQueues(ids, internal.Config.QueueSize)
	s.StartRunQueues()
	return nil
}

func (s *PipelineService) CreatePipeline(
	ctx context.Context,
	agentID int64,
	name, description, repository, scriptPath string,
) (*store.Pipeline, error) {
	p, err := s.pipelineStore.CreatePipeline(
		ctx,
		agentID,
		name,
		description,
		repository,
		scriptPath,
	)
	if err != nil {
		return nil, err
	}
	s.AddRunQueue(p.PipelineID, internal.Config.QueueSize)
	if err := s.StartRunQueue(p.PipelineID); err != nil {
		return p, err
	}
	return p, nil
}

func (s *PipelineService) GetPipelineByID(
	ctx context.Context,
	pipelineID int64,
) (*store.Pipeline, error) {
	return s.pipelineStore.ReadPipelineByID(ctx, pipelineID)
}

func (s *PipelineService) GetPipelineAndAgents(
	ctx context.Context,
	id int64,
) (*store.Pipeline, []*store.Agent, error) {
	p, err := s.pipelineStore.ReadPipelineByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	agents, err := s.agentService.ListAgents(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}
	return p, agents, nil
}

func (s *PipelineService) GetPipelineRunData(
	ctx context.Context,
	id int64,
) (*store.PipelineRunData, error) {
	prd, err := s.pipelineStore.ReadPipelineRunData(ctx, id)
	if err != nil {
		return nil, err
	}

	if prd.SSHPrivateKeyHash != nil {
		privateKey, err := s.aesEncrypter.DecryptAES(*prd.SSHPrivateKeyHash)
		if err != nil {
			return nil, err
		}
		prd.SSHPrivateKey = privateKey
	}

	return prd, nil
}

func (s *PipelineService) ListPipelines(
	ctx context.Context,
) ([]*store.Pipeline, error) {
	pipelines, err := s.pipelineStore.ListPipelines(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return pipelines, nil
}

func (s *PipelineService) ListPipelinesAndAgents(
	ctx context.Context,
) ([]*store.Pipeline, []*store.Agent, error) {
	pipelines, err := s.pipelineStore.ListPipelines(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}
	agents, err := s.agentService.ListAgents(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}
	return pipelines, agents, nil
}

func (s *PipelineService) ListScheduledPipelines(
	ctx context.Context,
) ([]*store.Pipeline, error) {
	pipelines, err := s.pipelineStore.ListScheduledPipelines(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return pipelines, nil
}

func (s *PipelineService) UpdatePipeline(
	ctx context.Context,
	pipelineID, agentID int64,
	name, description, repository, scriptPath string,
) error {
	return s.pipelineStore.UpdatePipeline(
		ctx,
		pipelineID,
		agentID,
		name,
		description,
		repository,
		scriptPath,
	)
}

func (s *PipelineService) UpdatePipelineSchedule(
	ctx context.Context,
	id int64,
	schedule, branch *string,
) error {
	p, err := s.pipelineStore.ReadPipelineByID(ctx, id)
	if err != nil {
		return err
	}

	if schedule == nil && p.Schedule != nil && s.scheduler != nil {
		if err := s.scheduler.RemoveJob(uuid.MustParse(*p.ScheduleJobID)); err != nil {
			log.Println("unable to remove existing job: ", err)
		}
	} else {
		jobID, err := s.SchedulePipelineRun(p.PipelineID, *schedule, *branch)
		if err != nil {
			return err
		}
		err = s.pipelineStore.UpdatePipelineSchedule(
			ctx,
			p.PipelineID,
			schedule,
			branch,
			jobID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *PipelineService) UpdatePipelineScheduleJobID(
	ctx context.Context,
	pipelineID int64,
	jobID *string,
) error {
	return s.pipelineStore.UpdatePipelineScheduleJobID(ctx, pipelineID, jobID)
}

func (s *PipelineService) AppendPipelineRunOutput(
	ctx context.Context,
	runID int64,
	out string,
) error {
	return s.runStore.AppendPipelineRunOutput(ctx, runID, out)
}

func (s *PipelineService) DeletePipeline(
	ctx context.Context, pipelineID int64,
) error {
	err := s.pipelineStore.DeletePipeline(ctx, pipelineID)
	if err != nil {
		return err
	}
	s.RemoveRunQueue(pipelineID)
	return nil
}

func (s *PipelineService) CollectPipelineRunArtifacts(
	ctx context.Context,
	pipelineID, runID int64,
) (string, error) {
	if exists, _ := util.PathExists("artifacts"); !exists {
		os.Mkdir("artifacts", os.ModePerm)
	}

	p, err := s.GetPipelineByID(ctx, pipelineID)
	if err != nil {
		return "", err
	}
	a, err := s.agentService.ReadAgentByID(ctx, p.PipelineAgentID)
	if err != nil {
		return "", err
	}
	c, err := s.credentialStore.ReadCredentialByID(ctx, *a.AgentCredentialID)
	if err != nil {
		return "", err
	}
	r, err := s.GetPipelineRunByID(ctx, runID)
	if err != nil {
		return "", err
	}

	artifactsDir := path.Join("artifacts", fmt.Sprintf("%d", r.RunID))
	if exists, _ := util.PathExists(artifactsDir); exists {
		return artifactsDir, nil
	}

	if err := os.Mkdir(artifactsDir, os.ModePerm); err != nil {
		return "", err
	}

	repoDir := p.Repository[strings.LastIndex(p.Repository, "/")+1:]
	repoDir = strings.TrimSuffix(repoDir, ".git")
	baseDir := path.Join(a.Workspace, *r.WorkingDirectory, repoDir)

	privateKey, err := s.aesEncrypter.DecryptAES(c.SSHPrivateKeyHash)
	if err != nil {
		return "", err
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	auth := ssh.PublicKeys(signer)
	cc := &ssh.ClientConfig{
		User:            c.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// connect to agent through SSH
	hostname := a.Hostname
	if !strings.HasSuffix(hostname, ":22") {
		hostname += ":22"
	}
	client, err := ssh.Dial("tcp", hostname, cc)
	if err != nil {
		return "", err
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	output, err := sess.Output(
		fmt.Sprintf("cd %s && cat %s", baseDir, p.ScriptPath),
	)
	if err != nil {
		return "", err
	}

	ps := new(PipelineScript)
	if err := yaml.Unmarshal(output, ps); err != nil {
		return "", err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", err
	}
	defer sftpClient.Close()

	for i, stage := range ps.Stages {
		if stage.Artifacts != "" {
			stageName := util.RemoveNonAlphabetChars(stage.Stage)
			if err := recursiveDownload(
				sftpClient,
				filepath.Join(baseDir, stage.Artifacts),
				filepath.Join(artifactsDir, fmt.Sprintf("%d_%s", i+1, stageName), stage.Artifacts),
			); err != nil {
				return "", err
			}
		}
	}

	return artifactsDir, nil
}

func recursiveDownload(sftpClient *sftp.Client, remotePath, localPath string) error {
	files, err := sftpClient.ReadDir(remotePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(localPath, os.ModePerm); err != nil {
		return err
	}

	for _, f := range files {
		remoteFilePath := filepath.Join(remotePath, f.Name())
		localFilePath := filepath.Join(localPath, f.Name())

		if f.IsDir() {
			if err := recursiveDownload(
				sftpClient, remoteFilePath, localFilePath,
			); err != nil {
				return err
			}
		} else {
			if err := downloadFile(
				sftpClient, remoteFilePath, localFilePath,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func downloadFile(sftpClient *sftp.Client, remotePath, localPath string) error {
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	if _, err := io.Copy(localFile, remoteFile); err != nil {
		return err
	}

	return nil
}

func (s *PipelineService) CreatePipelineRun(
	ctx context.Context,
	pipelineID int64,
	branch string,
) (*store.Run, error) {
	r, err := s.runStore.CreatePipelineRun(ctx, pipelineID, branch)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *PipelineService) GetPipelineRunByID(
	ctx context.Context, runID int64,
) (*store.Run, error) {
	r, err := s.runStore.ReadRunByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *PipelineService) UpdatePipelineRunStartedOn(
	ctx context.Context,
	runID int64,
	workingDirectory string,
	status store.RunStatus,
	startedOn *time.Time,
) error {
	return s.runStore.UpdatePipelineRunStartedOn(
		ctx, runID, workingDirectory, status, startedOn,
	)
}

func (s *PipelineService) UpdatePipelineRunEndedOn(
	ctx context.Context,
	runID int64,
	status store.RunStatus,
	artifacts *string,
	endedOn *time.Time,
) error {
	return s.runStore.UpdatePipelineRunEndedOn(
		ctx, runID, status, artifacts, endedOn,
	)
}

func (s *PipelineService) DeletePipelineRun(
	ctx context.Context, runID int64,
) error {
	return s.runStore.DeletePipelineRun(ctx, runID)
}

func (s *PipelineService) ListPipelineRuns(
	ctx context.Context,
	pipelineID int64,
) ([]store.Run, error) {
	runs, err := s.runStore.ListPipelineRuns(ctx, pipelineID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return runs, nil
}

func (s *PipelineService) ListLatestPipelineRuns(
	ctx context.Context,
	pipelineID, limit int64,
) ([]store.Run, error) {
	return s.runStore.ListLatestPipelineRuns(ctx, pipelineID, limit)
}

func (s *PipelineService) ListPipelineRunsPaginated(
	ctx context.Context,
	pipelineID, limit, offset int64,
) ([]store.Run, error) {
	return s.runStore.ListPipelineRunsPaginated(
		ctx, pipelineID, limit, offset,
	)
}

func (s *PipelineService) GetPipelineRunCount(
	ctx context.Context, id int64,
) (int64, error) {
	return s.runStore.CountPipelineRuns(ctx, id)
}

func (s *PipelineService) GetAPIKeyByValue(
	ctx context.Context,
	value string,
) (*store.APIKey, error) {
	return s.apiKeyStore.ReadAPIKeyByValue(ctx, value)
}

func (s *PipelineService) GetAPIKeyByID(
	ctx context.Context,
	id int64,
) (*store.APIKey, error) {
	return s.apiKeyStore.ReadAPIKeyByID(ctx, id)
}

func (s *PipelineService) ListAPIKeys(
	ctx context.Context,
) ([]*store.APIKey, error) {
	return s.apiKeyStore.ListAPIKeys(ctx)
}

func (s *PipelineService) SchedulePipelineRun(
	pipelineID int64,
	schedule, branch string,
) (*string, error) {
	if s.scheduler == nil {
		return nil, nil
	}
	job, err := s.scheduler.NewJob(
		gocron.CronJob(schedule, false),
		gocron.NewTask(func() {
			if r, err := s.CreatePipelineRun(
				context.Background(),
				pipelineID,
				branch,
			); err == nil {
				if err := s.EnqueueRun(r); err != nil {
					log.Println("queue is full")
					return
				}
			}
		}))
	if err != nil {
		return nil, fmt.Errorf("error scheduling pipeline job: %+w", err)
	}
	return util.AsPtr(job.ID().String()), nil
}

func (s *PipelineService) AddRunQueues(ids []int64, maxRuns int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		s.queues[id] = NewRunQueue(s.pipelineStore, s.runStore, s.aesEncrypter, maxRuns)
	}
}

func (s *PipelineService) StartRunQueues() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.queues {
		go s.queues[i].Run()
	}
}

func (s *PipelineService) AddRunQueue(id int64, maxRuns int64) {
	// Adds and starts a new RunQueue
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queues[id] = NewRunQueue(s.pipelineStore, s.runStore, s.aesEncrypter, maxRuns)
}

func (s *PipelineService) StartRunQueue(id int64) error {
	rq, ok := s.GetPipelineRunQueue(id)
	if !ok {
		return fmt.Errorf("run queue for pipeline %d does not exist", id)
	}
	go rq.Run()
	return nil
}

func (s *PipelineService) GetPipelineRunQueue(id int64) (*RunQueue, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rq, ok := s.queues[id]
	return rq, ok
}

func (s *PipelineService) RemoveRunQueue(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.queues, id)
}

func (s *PipelineService) EnqueueRun(r *store.Run) error {
	rq, ok := s.GetPipelineRunQueue(r.RunPipelineID)
	if !ok {
		return fmt.Errorf("run queue for pipeline %d does not exist", r.RunPipelineID)
	}

	return rq.Enqueue(r)

}

func (s *PipelineService) ShutdownRunQueue(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rq, ok := s.GetPipelineRunQueue(id)
	if !ok {
		return
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		rq.Shutdown()
	})
	wg.Wait()
}

func (s *PipelineService) ShutdownAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var wg sync.WaitGroup
	for _, rq := range s.queues {
		wg.Go(func() {
			rq.Shutdown()
		})
	}
	wg.Wait()
}
