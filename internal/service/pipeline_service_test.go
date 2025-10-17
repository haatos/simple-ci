package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/haatos/simple-ci/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPipelineStore struct {
	mock.Mock
}

func (m *MockPipelineStore) CreatePipeline(
	ctx context.Context,
	agentID int64,
	name,
	description,
	repository,
	scriptPath string,
) (*store.Pipeline, error) {
	args := m.Called(ctx, agentID, name, description, repository, scriptPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Pipeline), args.Error(1)
}

func (m *MockPipelineStore) ReadPipelineByID(
	ctx context.Context,
	id int64,
) (*store.Pipeline, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Pipeline), args.Error(1)
}

func (m *MockPipelineStore) UpdatePipeline(
	ctx context.Context,
	id int64,
	agentID int64,
	name,
	description,
	repository,
	scriptPath string,
) error {
	args := m.Called(ctx, id, agentID, name, description, repository, scriptPath)
	return args.Error(0)
}

func (m *MockPipelineStore) DeletePipeline(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPipelineStore) ListPipelines(
	ctx context.Context,
) ([]*store.Pipeline, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.Pipeline), args.Error(1)
}

func (m *MockPipelineStore) ListScheduledPipelines(ctx context.Context) ([]*store.Pipeline, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.Pipeline), args.Error(1)
}

func (m *MockPipelineStore) UpdatePipelineSchedule(
	ctx context.Context,
	id int64,
	schedule, branch, jobID *string,
) error {
	args := m.Called(ctx, id, schedule, branch, jobID)
	return args.Error(0)
}

func (m *MockPipelineStore) UpdatePipelineScheduleJobID(
	ctx context.Context,
	id int64,
	jobID *string,
) error {
	args := m.Called(ctx, id, jobID)
	return args.Error(0)
}

type MockRunStore struct {
	mock.Mock
}

func (m *MockRunStore) CreateRun(
	ctx context.Context,
	pipelineID int64,
	branch string,
) (*store.Run, error) {
	args := m.Called(ctx, pipelineID, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Run), args.Error(1)
}

func (m *MockRunStore) ReadRunByID(
	ctx context.Context,
	id int64,
) (*store.Run, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Run), args.Error(1)
}

func (m *MockRunStore) UpdateRunStartedOn(
	ctx context.Context,
	id int64,
	dir string,
	status store.RunStatus,
	startedOn *time.Time,
) error {
	args := m.Called(ctx, id, dir, status, startedOn)
	return args.Error(0)
}

func (m *MockRunStore) UpdateRunEndedOn(
	ctx context.Context,
	id int64,
	status store.RunStatus,
	output *string,
	artifacts *string,
	endedOn *time.Time,
) error {
	args := m.Called(ctx, id, status, output, artifacts, endedOn)
	return args.Error(0)
}

func (m *MockRunStore) DeleteRun(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRunStore) ListPipelineRuns(ctx context.Context, id int64) ([]store.Run, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]store.Run), args.Error(1)
}

func (m *MockRunStore) ListLatestPipelineRuns(
	ctx context.Context,
	id, limit int64,
) ([]store.Run, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]store.Run), args.Error(1)
}

func (m *MockRunStore) ListPipelineRunsPaginated(
	ctx context.Context,
	id, limit, offset int64,
) ([]store.Run, error) {
	args := m.Called(ctx, id, limit, offset)
	return args.Get(0).([]store.Run), args.Error(1)
}

func (m *MockRunStore) CountPipelineRuns(ctx context.Context, id int64) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

type MockCredentialService struct {
	mock.Mock
}

func (m *MockCredentialService) CreateCredential(
	ctx context.Context,
	username, description, sshPrivateKeyHash string,
) (*store.Credential, error) {
	args := m.Called(ctx, username, description, sshPrivateKeyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Credential), args.Error(1)
}

func (m *MockCredentialService) GetCredentialByID(
	ctx context.Context,
	credentialID int64,
) (*store.Credential, error) {
	args := m.Called(ctx, credentialID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Credential), args.Error(1)
}

func (m *MockCredentialService) ListCredentials(ctx context.Context) ([]*store.Credential, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.Credential), args.Error(1)
}

func (m *MockCredentialService) UpdateCredential(
	ctx context.Context,
	credentialID int64,
	username, description string,
) error {
	args := m.Called(ctx, credentialID, username, description)
	return args.Error(0)
}

func (m *MockCredentialService) DeleteCredential(ctx context.Context, credentialID int64) error {
	args := m.Called(ctx, credentialID)
	return args.Error(0)
}

func (m *MockCredentialService) DecryptAES(hash string) ([]byte, error) {
	args := m.Called(hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), nil
}

func TestPipelineService_CreatePipeline(t *testing.T) {
	t.Run("success - pipeline created", func(t *testing.T) {
		// arrange
		expectedPipeline := generatePipeline(0)
		mockStore := new(MockPipelineStore)
		mockStore.On(
			"CreatePipeline",
			context.Background(),
			expectedPipeline.PipelineAgentID,
			expectedPipeline.Name,
			expectedPipeline.Description,
			expectedPipeline.Repository,
			expectedPipeline.ScriptPath,
		).Return(expectedPipeline, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil)

		// act
		pipeline, err := pipelineService.CreatePipeline(
			context.Background(),
			expectedPipeline.PipelineAgentID,
			expectedPipeline.Name,
			expectedPipeline.Description,
			expectedPipeline.Repository,
			expectedPipeline.ScriptPath,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, pipeline)
		assert.Equal(t, expectedPipeline.PipelineAgentID, pipeline.PipelineAgentID)
		assert.Equal(t, expectedPipeline.Name, pipeline.Name)
		assert.Equal(t, expectedPipeline.Repository, pipeline.Repository)
		assert.Equal(t, expectedPipeline.ScriptPath, pipeline.ScriptPath)
		assert.Equal(t, expectedPipeline.Description, pipeline.Description)
	})
}

func TestPipelineService_GetPipelineByID(t *testing.T) {
	t.Run("success - pipeline is found", func(t *testing.T) {
		// arrange
		expectedPipeline := generatePipeline(0)
		mockStore := new(MockPipelineStore)
		mockStore.On(
			"ReadPipelineByID",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil)

		// act
		pipeline, err := pipelineService.GetPipelineByID(
			context.Background(),
			expectedPipeline.PipelineID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, pipeline)
	})
}

func TestPipelineService_GetPipelineAndAgents(t *testing.T) {
	t.Run("success - pipeline is found", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		expectedPipeline := generatePipeline(agent.AgentID)
		mockStore := new(MockPipelineStore)
		expectedAgents := []*store.Agent{agent}
		mockStore.On(
			"ReadPipelineByID",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, nil)
		mockService := new(testutil.MockAgentService)
		mockService.On(
			"ListAgents", context.Background(),
		).Return(expectedAgents, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, mockService, nil, nil)

		// act
		pipeline, agents, err := pipelineService.GetPipelineAndAgents(
			context.Background(),
			expectedPipeline.PipelineID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, pipeline)
		assert.Equal(t, len(expectedAgents), len(agents))
	})
}

func TestPipelineService_GetPipelineAgentAndCredential(t *testing.T) {
	t.Run("success - pipeline, agent, credential is found", func(t *testing.T) {
		// arrange
		_, _, expectedCredential := generateCredential()
		expectedAgent := generateAgent(expectedCredential.CredentialID)
		expectedPipeline := generatePipeline(expectedAgent.AgentID)
		mockStore := new(MockPipelineStore)
		mockAgentService := new(testutil.MockAgentService)
		mockStore.On(
			"ReadPipelineByID",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, nil)
		mockAgentService.On(
			"GetAgentByID",
			context.Background(),
			expectedAgent.AgentID,
		).Return(expectedAgent, nil)
		mockCredentialService := new(MockCredentialService)
		mockCredentialService.On(
			"GetCredentialByID",
			context.Background(),
			expectedCredential.CredentialID,
		).Return(expectedCredential, nil)
		mockCredentialService.On(
			"DecryptAES",
			expectedCredential.SSHPrivateKeyHash,
		).Return([]byte(""), nil)
		pipelineService := NewPipelineService(
			mockStore,
			nil,
			mockCredentialService,
			mockAgentService,
			nil,
			nil,
		)

		// act
		p, a, c, err := pipelineService.GetPipelineAgentAndCredential(
			context.Background(),
			expectedPipeline.PipelineID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.NotNil(t, a)
		assert.NotNil(t, c)
	})
}

func TestPipelineService_UpdatePipeline(t *testing.T) {
	t.Run("success - pipeline updated", func(t *testing.T) {
		// arrange
		pipeline := generatePipeline(0)
		mockStore := new(MockPipelineStore)
		mockStore.On(
			"UpdatePipeline", context.Background(),
			pipeline.PipelineID,
			pipeline.PipelineAgentID,
			pipeline.Name,
			pipeline.Description,
			pipeline.Repository,
			pipeline.ScriptPath,
		).Return(nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil)

		// act
		err := pipelineService.UpdatePipeline(
			context.Background(),
			pipeline.PipelineID,
			pipeline.PipelineAgentID,
			pipeline.Name,
			pipeline.Description,
			pipeline.Repository,
			pipeline.ScriptPath,
		)

		// assert
		assert.NoError(t, err)
	})
}

func TestPipelineService_DeletePipeline(t *testing.T) {
	t.Run("success - pipeline found", func(t *testing.T) {
		// arrange
		mockStore := new(MockPipelineStore)
		var pipelineID int64 = 1
		mockStore.On("DeletePipeline", context.Background(), pipelineID).Return(nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil)

		// act
		err := pipelineService.DeletePipeline(context.Background(), pipelineID)

		// assert
		assert.NoError(t, err)
	})
}

func TestPipelineService_ListPipelines(t *testing.T) {
	t.Run("success - pipelines found", func(t *testing.T) {
		// arrange
		expectedPipeline := generatePipeline(0)
		expectedPipelines := []*store.Pipeline{expectedPipeline}
		mockStore := new(MockPipelineStore)
		mockStore.On(
			"ListPipelines", context.Background(),
		).Return(expectedPipelines, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil)

		// act
		pipelines, err := pipelineService.ListPipelines(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, len(expectedPipelines), len(pipelines))
	})
	t.Run("success - list empty", func(t *testing.T) {
		// arrange
		expectedPipelines := []*store.Pipeline{}
		mockStore := new(MockPipelineStore)
		mockStore.On("ListPipelines", context.Background()).
			Return(expectedPipelines, sql.ErrNoRows)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil)

		// act
		pipelines, err := pipelineService.ListPipelines(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, 0, len(pipelines))
	})
}

func TestPipelineService_ListPipelinesAndAgents(t *testing.T) {
	t.Run("success - pipelines found", func(t *testing.T) {
		// arrange
		expectedAgent := generateAgent(0)
		expectedAgents := []*store.Agent{expectedAgent}
		expectedPipeline := generatePipeline(expectedAgent.AgentID)
		expectedPipelines := []*store.Pipeline{expectedPipeline}
		mockStore := new(MockPipelineStore)
		mockStore.On(
			"ListPipelines", context.Background(),
		).Return(expectedPipelines, nil)
		mockService := new(testutil.MockAgentService)
		mockService.On(
			"ListAgents", context.Background()).Return(expectedAgents, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, mockService, nil, nil)

		// act
		pipelines, agents, err := pipelineService.ListPipelinesAndAgents(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, len(expectedPipelines), len(pipelines))
		assert.Equal(t, len(expectedAgents), len(agents))
	})
	t.Run("success - list empty", func(t *testing.T) {
		// arrange
		mockStore := new(MockPipelineStore)
		mockService := new(testutil.MockAgentService)
		expectedPipelines := []*store.Pipeline{}
		expectedAgents := []*store.Agent{}
		mockStore.On(
			"ListPipelines", context.Background(),
		).Return(expectedPipelines, sql.ErrNoRows)
		mockService.On(
			"ListAgents", context.Background(),
		).Return(expectedAgents, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, mockService, nil, nil)

		// act
		pipelines, agents, err := pipelineService.ListPipelinesAndAgents(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, 0, len(pipelines))
		assert.Equal(t, 0, len(agents))
	})
}

func TestPipelineService_UpdatePipelineSchedule(t *testing.T) {
	t.Run("success - pipeline schedule updates", func(t *testing.T) {
		// arrange
		expectedPipeline := generatePipeline(0)
		mockStore := new(MockPipelineStore)
		schedule := "* * * * *"
		branch := "main"
		var jobID *string

		mockStore.On(
			"ReadPipelineByID",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, nil)
		mockStore.On(
			"UpdatePipelineSchedule",
			context.Background(),
			expectedPipeline.PipelineID,
			&schedule,
			&branch,
			jobID,
		).Return(nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil)

		// act
		err := pipelineService.UpdatePipelineSchedule(
			context.Background(),
			nil,
			expectedPipeline.PipelineID,
			&schedule,
			&branch,
		)

		// assert
		assert.NoError(t, err)
	})
}

func TestPipelineService_CreateRun(t *testing.T) {
	t.Run("success - run created", func(t *testing.T) {
		// arrange
		expectedRun := generateRun(0)
		mockStore := new(MockRunStore)
		branch := "main"
		mockStore.On(
			"CreateRun",
			context.Background(),
			expectedRun.RunPipelineID, branch,
		).Return(expectedRun, nil)
		pipelineService := NewPipelineService(
			nil, mockStore, nil, nil, nil, nil,
		)

		// act
		r, err := pipelineService.CreateRun(
			context.Background(), expectedRun.RunPipelineID, branch,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, expectedRun.RunID, r.RunID)
	})
}

func TestPipelineService_GetRunByID(t *testing.T) {
	t.Run("success - run is found", func(t *testing.T) {
		// arrange
		expectedRun := generateRun(0)
		mockStore := new(MockRunStore)
		mockStore.On(
			"ReadRunByID",
			context.Background(),
			expectedRun.RunID,
		).Return(expectedRun, nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil)

		// act
		r, err := pipelineService.GetRunByID(context.Background(), expectedRun.RunID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, expectedRun.RunID, r.RunID)
	})
}

func TestPipelineService_UpdateRunStartedOn(t *testing.T) {
	t.Run("success - run found", func(t *testing.T) {
		// arrange
		run := generateRun(0)
		mockStore := new(MockRunStore)
		mockStore.On(
			"UpdateRunStartedOn",
			context.Background(),
			run.RunID,
			*run.WorkingDirectory,
			run.Status,
			run.StartedOn,
		).Return(nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil)

		// act
		err := pipelineService.UpdateRunStartedOn(
			context.Background(),
			run.RunID,
			*run.WorkingDirectory,
			run.Status,
			run.StartedOn,
		)

		// assert
		assert.NoError(t, err)
	})
}

func TestPipelineService_UpdateRunEndedOn(t *testing.T) {
	t.Run("success - run found", func(t *testing.T) {
		// arrange
		var runID int64 = 1
		status := store.StatusPassed
		output := "output"
		artifacts := "artifacts"
		endedOn := time.Now().UTC()
		mockStore := new(MockRunStore)
		mockStore.On(
			"UpdateRunEndedOn",
			context.Background(),
			runID, status, &output, &artifacts, &endedOn,
		).Return(nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil)

		// act
		err := pipelineService.UpdateRunEndedOn(
			context.Background(),
			runID,
			status,
			&output,
			&artifacts,
			&endedOn,
		)

		// assert
		assert.NoError(t, err)
	})
}

func TestPipelineService_DeleteRun(t *testing.T) {
	t.Run("success - run is found", func(t *testing.T) {
		// arrange
		var runID int64 = 1
		mockStore := new(MockRunStore)
		mockStore.On(
			"DeleteRun",
			context.Background(),
			runID,
		).Return(nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil)

		// act
		err := pipelineService.DeleteRun(context.Background(), runID)

		// assert
		assert.NoError(t, err)
	})
}

func TestPipelineService_ListPipelineRuns(t *testing.T) {
	t.Run("success - pipeline runs found", func(t *testing.T) {
		// arrange
		expectedRun := generateRun(0)
		expectedRuns := []store.Run{*expectedRun}
		mockStore := new(MockRunStore)
		mockStore.On(
			"ListPipelineRuns",
			context.Background(),
			expectedRun.RunPipelineID,
		).Return(expectedRuns, nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil)

		// act
		pipelines, err := pipelineService.ListPipelineRuns(
			context.Background(),
			expectedRun.RunPipelineID,
		)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, len(expectedRuns), len(pipelines))
	})
}

func generatePipeline(agentID int64) *store.Pipeline {
	if agentID == 0 {
		agentID = rand.Int63()
	}
	p := &store.Pipeline{
		PipelineID:      rand.Int63(),
		PipelineAgentID: agentID,
		Name:            fmt.Sprintf("pipeline%d", time.Now().UnixNano()),
		Description:     fmt.Sprintf("description%d", time.Now().UnixNano()),
		Repository:      "git@github.com:haatos/simple-ci.git",
		ScriptPath:      "pipelines/testing-pipeline.yml",
	}
	return p
}

func generateRun(pipelineID int64) *store.Run {
	if pipelineID == 0 {
		pipelineID = rand.Int63()
	}
	r := &store.Run{
		RunID:            rand.Int63(),
		RunPipelineID:    pipelineID,
		Branch:           "main",
		WorkingDirectory: util.AsPtr("/tmp"),
		Status:           store.StatusQueued,
		CreatedOn:        time.Now().UTC(),
	}
	return r
}
