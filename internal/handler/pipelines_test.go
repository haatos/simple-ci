package handler

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPipelineService struct {
	mock.Mock
}

func (m *MockPipelineService) CreatePipeline(
	ctx context.Context,
	agentID int64,
	name, description, repository, scriptPath string,
) (*store.Pipeline, error) {
	args := m.Called(ctx, name, description, repository, scriptPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Pipeline), args.Error(1)
}

func (m *MockPipelineService) GetPipelineByID(
	ctx context.Context,
	id int64,
) (*store.Pipeline, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Pipeline), args.Error(1)
}

func (m *MockPipelineService) GetPipelineAndAgents(
	ctx context.Context,
	id int64,
) (*store.Pipeline, []*store.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil || args.Get(1) == nil {
		return nil, nil, args.Error(2)
	}
	p := args.Get(0).(*store.Pipeline)
	agents := args.Get(1).([]*store.Agent)
	return p, agents, nil
}

func (m *MockPipelineService) GetPipelineAgentAndCredential(
	ctx context.Context,
	id int64,
) (*store.Pipeline, *store.Agent, *store.Credential, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil || args.Get(1) == nil || args.Get(2) == nil {
		return nil, nil, nil, args.Error(3)
	}
	p := args.Get(0).(*store.Pipeline)
	a := args.Get(1).(*store.Agent)
	c := args.Get(2).(*store.Credential)
	return p, a, c, nil
}

func (m *MockPipelineService) ListPipelines(
	ctx context.Context,
) ([]*store.Pipeline, error) {
	args := m.Called(ctx)
	var pipelines []*store.Pipeline
	if args.Get(0) != nil {
		pipelines = args.Get(0).([]*store.Pipeline)
	}
	err := args.Error(1)
	return pipelines, err
}

func (m *MockPipelineService) ListPipelinesAndAgents(
	ctx context.Context,
) ([]*store.Pipeline, []*store.Agent, error) {
	args := m.Called(ctx)
	var pipelines []*store.Pipeline
	var agents []*store.Agent
	if args.Get(0) != nil {
		pipelines = args.Get(0).([]*store.Pipeline)
	}
	if args.Get(1) != nil {
		agents = args.Get(1).([]*store.Agent)
	}
	err := args.Error(2)
	return pipelines, agents, err
}

func (m *MockPipelineService) UpdatePipeline(
	ctx context.Context,
	id int64,
	agentID int64,
	name, description, repository, scriptPath string,
) error {
	args := m.Called(ctx, id, agentID, name, description, repository, scriptPath)
	return args.Error(0)
}

func (m *MockPipelineService) DeletePipeline(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPipelineService) TestPipelineConnection(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPipelineService) CollectPipelineRunArtifacts(
	ctx context.Context,
	pipelineID, runID int64,
) (string, error) {
	args := m.Called(ctx, pipelineID, runID)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockPipelineService) ListScheduledPipelines(
	ctx context.Context,
) ([]*store.Pipeline, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.Pipeline), args.Error(1)
}

func (m *MockPipelineService) UpdatePipelineSchedule(
	ctx context.Context, runCh chan *store.Run, id int64, schedule, branch *string,
) error {
	args := m.Called(ctx, runCh, id, schedule, branch)
	return args.Error(0)
}

func (m *MockPipelineService) UpdatePipelineScheduleJobID(
	ctx context.Context, id int64, jobID *string,
) error {
	args := m.Called(ctx, id, jobID)
	return args.Error(0)
}

func (m *MockPipelineService) CreateRun(
	ctx context.Context,
	pipelineID int64,
	branch string,
) (*store.Run, error) {
	args := m.Called(ctx, pipelineID, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Run), nil
}
func (m *MockPipelineService) GetRunByID(ctx context.Context, runID int64) (*store.Run, error) {
	args := m.Called(ctx, runID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Run), nil
}
func (m *MockPipelineService) UpdateRunStartedOn(
	ctx context.Context,
	runID int64,
	workingDirectory string,
	status store.RunStatus,
	startedOn *time.Time,
) error {
	args := m.Called(ctx, runID, workingDirectory, status, startedOn)
	return args.Error(0)
}
func (m *MockPipelineService) UpdateRunEndedOn(
	ctx context.Context,
	runID int64,
	status store.RunStatus,
	output *string,
	artifacts *string,
	endedOn *time.Time,
) error {
	args := m.Called(ctx, runID, status, output, artifacts, endedOn)
	return args.Error(0)
}
func (m *MockPipelineService) DeleteRun(ctx context.Context, runID int64) error {
	args := m.Called(ctx, runID)
	return args.Error(0)
}

func (m *MockPipelineService) ListPipelineRuns(
	ctx context.Context,
	pipelineID int64,
) ([]store.Run, error) {
	args := m.Called(ctx, pipelineID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]store.Run), nil
}

func (m *MockPipelineService) ListLatestPipelineRuns(
	ctx context.Context,
	id int64,
	limit int64,
) ([]store.Run, error) {
	args := m.Called(ctx, id, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]store.Run), nil
}

func (m *MockPipelineService) ListPipelineRunsPaginated(
	ctx context.Context,
	id int64,
	limit int64,
	offset int64,
) ([]store.Run, error) {
	args := m.Called(ctx, id, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]store.Run), nil
}
func (m *MockPipelineService) GetPipelineRunCount(ctx context.Context, id int64) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func TestPipelinesHandler_GetPipelinesPage(t *testing.T) {
	t.Run("success - pipelines found on page", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		pipeline := generatePipeline(agent.AgentID)
		expectedAgents := []*store.Agent{agent}
		expectedPipelines := []*store.Pipeline{pipeline}
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On("ListPipelinesAndAgents", context.Background()).
			Return(expectedPipelines, expectedAgents, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/pipelines", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.GetPipelinesPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `<h2 class="card-title"`)
		assert.Contains(t, body, pipeline.Name)
		assert.Contains(t, body, pipeline.Description)
	})
	t.Run("success - pipelines found on main element", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		pipeline := generatePipeline(agent.AgentID)
		expectedAgents := []*store.Agent{agent}
		expectedPipelines := []*store.Pipeline{pipeline}
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On("ListPipelinesAndAgents", context.Background()).
			Return(expectedPipelines, expectedAgents, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/pipelines", nil)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.GetPipelinesPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `<h2 class="card-title"`)
		assert.Contains(t, body, pipeline.Name)
		assert.Contains(t, body, pipeline.Description)
	})
}

func TestPipelinesHandler_GetPipelinePage(t *testing.T) {
	t.Run("success - pipeline form found on page", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		expectedAgents := []*store.Agent{agent}
		expectedPipeline := generatePipeline(agent.AgentID)
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On(
			"GetPipelineAndAgents",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, expectedAgents, nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/app/pipelines/%d", expectedPipeline.PipelineID),
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("pipeline_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedPipeline.PipelineID))
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.GetPipelinePage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="name"`)
		assert.Contains(t, body, `name="description"`)
		assert.Contains(t, body, `name="repository"`)
		assert.Contains(t, body, `name="script_path"`)
	})
	t.Run("success - pipeline form found on main element", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		expectedAgents := []*store.Agent{agent}
		expectedPipeline := generatePipeline(agent.AgentID)
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On(
			"GetPipelineAndAgents",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, expectedAgents, nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/app/pipelines/%d", expectedPipeline.PipelineID),
			nil,
		)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("pipeline_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedPipeline.PipelineID))
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.GetPipelinePage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="name"`)
		assert.Contains(t, body, `name="description"`)
		assert.Contains(t, body, `name="repository"`)
		assert.Contains(t, body, `name="script_path"`)
	})
}

func TestPipelinesHandler_PatchPipeline(t *testing.T) {
	t.Run("success - pipeline updated", func(t *testing.T) {
		// arrange
		pipeline := generatePipeline(0)
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On(
			"UpdatePipeline",
			context.Background(),
			pipeline.PipelineID, pipeline.PipelineAgentID,
			pipeline.Name, pipeline.Description, pipeline.Repository, pipeline.ScriptPath,
		).Return(nil)

		formData := url.Values{}
		formData.Set("pipeline_agent_id", fmt.Sprintf("%d", pipeline.PipelineAgentID))
		formData.Set("name", pipeline.Name)
		formData.Set("description", pipeline.Description)
		formData.Set("repository", pipeline.Repository)
		formData.Set("script_path", pipeline.ScriptPath)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/app/pipelines/%d", pipeline.PipelineID),
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("pipeline_id")
		c.SetParamValues(fmt.Sprintf("%d", pipeline.PipelineID))
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.PatchPipeline(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, `name="success-toast"`)
		assert.Contains(t, body, "Pipeline updated")
	})
	t.Run("failure - pipeline not found", func(t *testing.T) {
		// arrange
		p := generatePipeline(0)
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On(
			"UpdatePipeline",
			context.Background(),
			p.PipelineID, p.PipelineAgentID,
			p.Name, p.Description, p.Repository, p.ScriptPath,
		).Return(sql.ErrNoRows)

		formData := url.Values{}
		formData.Set("pipeline_agent_id", fmt.Sprintf("%d", p.PipelineAgentID))
		formData.Set("name", p.Name)
		formData.Set("description", p.Description)
		formData.Set("repository", p.Repository)
		formData.Set("script_path", p.ScriptPath)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/app/pipelines/%d", p.PipelineID),
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("pipeline_id")
		c.SetParamValues(fmt.Sprintf("%d", p.PipelineID))
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.PatchPipeline(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, htmxErr.Code)
	})
}

func TestPipelinesHandler_DeletePipeline(t *testing.T) {
	t.Run("success - pipeline deleted", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		expectedPipeline := generatePipeline(agent.AgentID)
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On(
			"GetPipelineByID",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, nil)
		mockPipelineService.On(
			"DeletePipeline",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/app/pipelines/%d", expectedPipeline.PipelineID),
			nil,
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("pipeline_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedPipeline.PipelineID))
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.DeletePipeline(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "/app/pipelines", c.Response().Header().Get("hx-redirect"))
	})
	t.Run("failure - pipeline not found", func(t *testing.T) {
		// arrange
		pipeline := generatePipeline(0)
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On("GetPipelineByID", context.Background(), pipeline.PipelineID).
			Return(nil, sql.ErrNoRows)
		mockPipelineService.On("DeletePipeline", context.Background(), pipeline.PipelineID).
			Return(sql.ErrNoRows)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/app/pipelines/%d", pipeline.PipelineID),
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("pipeline_id")
		c.SetParamValues(fmt.Sprintf("%d", pipeline.PipelineID))
		h := NewPipelineHandler(mockPipelineService)

		// act
		err := h.DeletePipeline(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, htmxErr.Code)
	})
}

func TestPipelinesHandler_PostPipelineRun(t *testing.T) {
	t.Run("success - pipeline run is created", func(t *testing.T) {
		// arrange
		expectedPipeline := generatePipeline(0)
		expectedRun := generateRun(expectedPipeline.PipelineID)
		mockPipelineService := new(MockPipelineService)
		mockPipelineService.On(
			"GetPipelineByID",
			context.Background(),
			expectedPipeline.PipelineID,
		).Return(expectedPipeline, nil)
		mockPipelineService.On(
			"CreateRun",
			context.Background(),
			expectedPipeline.PipelineID,
			expectedRun.Branch,
		).Return(expectedRun, nil)

		formData := url.Values{}
		formData.Set("branch", expectedRun.Branch)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/app/pipelines/%d/runs", expectedPipeline.PipelineID),
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("pipeline_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedPipeline.PipelineID))
		h := NewPipelineHandler(mockPipelineService)
		var queuedRun *store.Run
		var wg sync.WaitGroup
		wg.Go(func() {
			for r := range h.RunCh {
				queuedRun = r
				wg.Done()
			}
		})

		// act
		err := h.PostPipelineRun(c)
		wg.Wait()

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expectedRun.RunID, queuedRun.RunID)
		hxRedirect := rec.Header().Get("hx-redirect")
		assert.Equal(
			t,
			hxRedirect,
			fmt.Sprintf(
				"/app/pipelines/%d/runs/%d",
				expectedPipeline.PipelineID,
				expectedRun.RunID,
			),
		)
	})
}

func TestPipelinesHandler_GetPipelineRunPage(t *testing.T) {
	t.Run("success - pipeline run page html is returned", func(t *testing.T) {
		// arrange
		expectedPipeline := generatePipeline(0)
		expectedRun := generateRun(expectedPipeline.PipelineID)
		mockService := new(MockPipelineService)
		mockService.On(
			"GetRunByID",
			context.Background(),
			expectedRun.RunID,
		).Return(expectedRun, nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf(
				"/app/pipelines/%d/runs/%d",
				expectedPipeline.PipelineID,
				expectedRun.RunID,
			),
			nil,
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("pipeline_id", "run_id")
		c.SetParamValues(
			fmt.Sprintf("%d", expectedPipeline.PipelineID),
			fmt.Sprintf("%d", expectedRun.RunID),
		)
		h := NewPipelineHandler(mockService)

		// act
		err := h.GetPipelineRunPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `id="output-textbox"`)
	})
	t.Run("success - pipeline run page main html is returned", func(t *testing.T) {
		// arrange
		expectedPipeline := generatePipeline(0)
		expectedRun := generateRun(expectedPipeline.PipelineID)
		mockService := new(MockPipelineService)
		mockService.On(
			"GetRunByID",
			context.Background(),
			expectedRun.RunID,
		).Return(expectedRun, nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf(
				"/app/pipelines/%d/runs/%d",
				expectedPipeline.PipelineID,
				expectedRun.RunID,
			),
			nil,
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.SetParamNames("pipeline_id", "run_id")
		c.SetParamValues(
			fmt.Sprintf("%d", expectedPipeline.PipelineID),
			fmt.Sprintf("%d", expectedRun.RunID),
		)
		h := NewPipelineHandler(mockService)

		// act
		err := h.GetPipelineRunPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `id="output-textbox"`)
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
