package service

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/testutil"
	"github.com/haatos/simple-ci/internal/util"
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

func (m *MockPipelineStore) ReadPipelineRunData(
	ctx context.Context,
	id int64,
) (*store.PipelineRunData, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.PipelineRunData), nil
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
	artifacts *string,
	endedOn *time.Time,
) error {
	args := m.Called(ctx, id, status, artifacts, endedOn)
	return args.Error(0)
}

func (m *MockRunStore) AppendRunOutput(ctx context.Context, runID int64, out string) error {
	args := m.Called(ctx, runID, out)
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
	args := m.Called(ctx, id, limit)
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
		internal.Config = &internal.Configuration{SessionExpiresHours: 1, QueueSize: 3}
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
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

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
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

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
		mockAgentStore := new(MockAgentStore)
		mockAgentStore.On(
			"ListAgents", context.Background(),
		).Return(expectedAgents, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, mockAgentStore, nil, nil, nil)

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

func TestPipelineService_GetPipelineRunData(t *testing.T) {
	t.Run("success - pipeline run data found", func(t *testing.T) {
		// arrange
		_, _, c := generateCredential()
		a := generateAgent(c.CredentialID)
		p := generatePipeline(a.AgentID)
		expectedPipelineRunData := &store.PipelineRunData{
			PipelineID:        p.PipelineID,
			Repository:        p.Repository,
			ScriptPath:        p.ScriptPath,
			AgentID:           a.AgentID,
			Hostname:          a.Hostname,
			Workspace:         a.Workspace,
			CredentialID:      &c.CredentialID,
			Username:          &c.Username,
			SSHPrivateKeyHash: &c.SSHPrivateKeyHash,
		}
		mockStore := new(MockPipelineStore)
		mockStore.On(
			"ReadPipelineRunData",
			context.Background(), p.PipelineID,
		).Return(expectedPipelineRunData, nil)
		mockEncrypter := new(MockEncrypter)
		mockEncrypter.On("DecryptAES", c.SSHPrivateKeyHash).Return([]byte("test"), nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, mockEncrypter)

		// act
		prd, err := pipelineService.GetPipelineRunData(context.Background(), p.PipelineID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, prd)
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
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

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
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

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
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

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
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

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
		mockAgentStore := new(MockAgentStore)
		mockAgentStore.On(
			"ListAgents", context.Background()).Return(expectedAgents, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, mockAgentStore, nil, nil, nil)

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
		mockAgentStore := new(MockAgentStore)
		expectedPipelines := []*store.Pipeline{}
		expectedAgents := []*store.Agent{}
		mockStore.On(
			"ListPipelines", context.Background(),
		).Return(expectedPipelines, sql.ErrNoRows)
		mockAgentStore.On(
			"ListAgents", context.Background(),
		).Return(expectedAgents, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, mockAgentStore, nil, nil, nil)

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
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

		// act
		err := pipelineService.UpdatePipelineSchedule(
			context.Background(),
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
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

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
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

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
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

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
		artifacts := "artifacts"
		endedOn := time.Now().UTC()
		mockStore := new(MockRunStore)
		mockStore.On(
			"UpdateRunEndedOn",
			context.Background(),
			runID, status, &artifacts, &endedOn,
		).Return(nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

		// act
		err := pipelineService.UpdateRunEndedOn(
			context.Background(),
			runID,
			status,
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
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

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
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

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

func TestPipelineService_ListLatestPipelineRuns(t *testing.T) {
	t.Run("success - latest pipelines are found", func(t *testing.T) {
		// arrange
		var limit int64 = 3
		expectedPipeline := generatePipeline(0)
		expectedRuns := []store.Run{
			*generateRun(0),
			*generateRun(0),
		}
		mockStore := new(MockRunStore)
		mockStore.On(
			"ListLatestPipelineRuns",
			context.Background(), expectedPipeline.PipelineID, limit,
		).Return(expectedRuns, nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

		// act
		runs, err := pipelineService.ListLatestPipelineRuns(
			context.Background(),
			expectedPipeline.PipelineID,
			limit,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, runs)
		assert.Equal(t, len(expectedRuns), len(runs))
	})
}

func TestPipelineService_ListPipelineRunsPaginated(t *testing.T) {
	t.Run("success - pipelines are found", func(t *testing.T) {
		// arrange
		var limit int64 = 3
		var offset int64 = 10
		expectedPipeline := generatePipeline(0)
		expectedRuns := []store.Run{
			*generateRun(0),
			*generateRun(0),
		}
		mockStore := new(MockRunStore)
		mockStore.On(
			"ListPipelineRunsPaginated",
			context.Background(), expectedPipeline.PipelineID, limit, offset,
		).Return(expectedRuns, nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

		// act
		runs, err := pipelineService.ListPipelineRunsPaginated(
			context.Background(),
			expectedPipeline.PipelineID,
			limit,
			offset,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, runs)
		assert.Equal(t, len(expectedRuns), len(runs))
	})
}

func TestPipelineService_GetPipelineRunCount(t *testing.T) {
	t.Run("success - pipeline run count is returned", func(t *testing.T) {
		// arrange
		var expectedCount int64 = 5
		expectedPipeline := generatePipeline(0)
		mockStore := new(MockRunStore)
		mockStore.On(
			"CountPipelineRuns",
			context.Background(), expectedPipeline.PipelineID,
		).Return(expectedCount, nil)
		pipelineService := NewPipelineService(nil, mockStore, nil, nil, nil, nil, nil)

		// act
		count, err := pipelineService.GetPipelineRunCount(
			context.Background(),
			expectedPipeline.PipelineID,
		)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})
}

func TestPipelineService_GetAPIKeyByValue(t *testing.T) {
	t.Run("success - api key is found", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		expectedAPIKey := generateAPIKey()
		mockService := new(testutil.MockAPIKeyService)
		mockService.On(
			"GetAPIKeyByValue",
			ctx,
			expectedAPIKey.Value,
		).Return(expectedAPIKey, nil)
		pipelineService := NewPipelineService(nil, nil, nil, nil, mockService, nil, nil)

		// act
		ak, err := pipelineService.GetAPIKeyByValue(ctx, expectedAPIKey.Value)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, ak)
		assert.Equal(t, expectedAPIKey.ID, ak.ID)
		assert.Equal(t, expectedAPIKey.Value, ak.Value)
	})
}

func TestPipelineService_InitializeRunQueues(t *testing.T) {
	t.Run("success - run queues are initialized", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		expectedPipelines := []*store.Pipeline{generatePipeline(0), generatePipeline(0)}
		mockStore := new(MockPipelineStore)
		mockStore.On(
			"ListPipelines",
			ctx,
		).Return(expectedPipelines, nil)
		pipelineService := NewPipelineService(mockStore, nil, nil, nil, nil, nil, nil)

		// act
		err := pipelineService.InitializeRunQueues(ctx)

		// assert
		assert.NoError(t, err)
	})
}

func TestPipelineService_GetRunQueue(t *testing.T) {
	t.Run("success - run queue is found", func(t *testing.T) {
		// arrange
		p := generatePipeline(0)
		pipelineService := NewPipelineService(nil, nil, nil, nil, nil, nil, nil)
		pipelineService.AddRunQueue(p.PipelineID, internal.Config.QueueSize)

		// act
		rq, ok := pipelineService.GetRunQueue(p.PipelineID)

		// assert
		assert.True(t, ok)
		assert.NotNil(t, rq)
	})
}

func TestPipelineService_RemoveRunQueue(t *testing.T) {
	t.Run("success - run queue is removed", func(t *testing.T) {
		// arrange
		p := generatePipeline(0)
		pipelineService := NewPipelineService(nil, nil, nil, nil, nil, nil, nil)
		pipelineService.AddRunQueue(p.PipelineID, 3)
		_, ok := pipelineService.GetRunQueue(p.PipelineID)
		assert.True(t, ok)

		// act
		pipelineService.RemoveRunQueue(p.PipelineID)
		rq, ok := pipelineService.GetRunQueue(p.PipelineID)

		// assert
		assert.False(t, ok)
		assert.Nil(t, rq)
	})
}

func TestPipelineService_EnqueueRun(t *testing.T) {
	t.Run("success - run is enqueued", func(t *testing.T) {
		// arrange
		p := generatePipeline(0)
		r := generateRun(p.PipelineID)
		pipelineService := NewPipelineService(nil, nil, nil, nil, nil, nil, nil)
		pipelineService.AddRunQueue(p.PipelineID, 2)

		// act
		err := pipelineService.EnqueueRun(r)

		// assert
		assert.NoError(t, err)
	})
	t.Run("failure - run queue is full", func(t *testing.T) {
		// arrange
		p := generatePipeline(0)
		r := generateRun(p.PipelineID)
		pipelineService := NewPipelineService(nil, nil, nil, nil, nil, nil, nil)
		pipelineService.AddRunQueue(p.PipelineID, 1)
		err := pipelineService.EnqueueRun(r)
		assert.NoError(t, err)

		// act
		r = generateRun(p.PipelineID)
		err = pipelineService.EnqueueRun(r)

		// assert
		assert.Error(t, err)
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
