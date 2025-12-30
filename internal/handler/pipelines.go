package handler

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/haatos/simple-ci/internal/views"
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
)

const maxRunsPerPage int64 = 10

func SetupPipelineRoutes(
	g *echo.Group,
	pipelineService PipelineServicer,
	apiKeyService APIKeyServicer,
) {
	h := NewPipelineHandler(pipelineService, apiKeyService)
	g.POST(
		"/app/pipelines/:pipeline_id/webhook-trigger/:branch",
		h.PostPipelineRunWebhookTrigger,
	)
	pipelinesGroup := g.Group("/app/pipelines", IsAuthenticated)
	pipelinesGroup.GET("", h.GetPipelinesPage)
	pipelinesGroup.POST("", h.PostPipeline)
	pipelinesGroup.PATCH("", h.PatchPipeline)
	pipelinesGroup.GET("/:pipeline_id", h.GetPipelinePage)
	pipelinesGroup.GET("/:pipeline_id/card-content", h.GetPipelineCardContent)
	pipelinesGroup.DELETE("/:pipeline_id", h.DeletePipeline)
	pipelinesGroup.PATCH("/:pipeline_id/schedule", h.PatchPipelineSchedule)
	pipelinesGroup.GET("/:pipeline_id/latest-runs", h.GetLatestPipelineRuns)
	pipelinesGroup.POST("/:pipeline_id/runs", h.PostPipelineRun)
	pipelinesGroup.GET("/:pipeline_id/runs/:run_id", h.GetPipelineRunPage)
	pipelinesGroup.GET("/:pipeline_id/runs", h.GetPipelineRunsPage)
	pipelinesGroup.GET("/:pipeline_id/runs-list", h.GetPipelineRunsList)
	pipelinesGroup.GET("/:pipeline_id/runs/:run_id/sse", h.GetPipelineRunSSE)
	pipelinesGroup.GET("/:pipeline_id/runs/:run_id/output", h.GetPipelineRunOutput)
	pipelinesGroup.GET("/:pipeline_id/runs/:run_id/status", h.GetPipelineRunStatus)
	pipelinesGroup.GET("/:pipeline_id/runs/:run_id/artifacts", h.GetPipelineRunArtifacts)
	pipelinesGroup.POST("/:pipeline_id/runs/:run_id/cancel", h.PostCancelPipelineRun)
}

type PipelineWriter interface {
	CreatePipeline(
		ctx context.Context,
		agentID int64,
		name, description, repository, scriptPath string,
	) (*store.Pipeline, error)
	UpdatePipeline(
		ctx context.Context,
		pipelineID, agentID int64,
		name, description, repository, scriptPath string,
	) error
	UpdatePipelineSchedule(ctx context.Context, id int64, branch, schedule *string) error
	UpdatePipelineScheduleJobID(ctx context.Context, id int64, jobID *string) error
	DeletePipeline(ctx context.Context, pipelineID int64) error
}

type PipelineReader interface {
	GetPipelineByID(
		ctx context.Context,
		pipelineID int64,
	) (*store.Pipeline, error)
	GetPipelineAndAgents(ctx context.Context, id int64) (*store.Pipeline, []*store.Agent, error)
	GetPipelineRunData(ctx context.Context, id int64) (*store.PipelineRunData, error)
	ListPipelines(ctx context.Context) ([]*store.Pipeline, error)
	ListPipelinesAndAgents(ctx context.Context) ([]*store.Pipeline, []*store.Agent, error)
	ListScheduledPipelines(ctx context.Context) ([]*store.Pipeline, error)
	CollectPipelineRunArtifacts(ctx context.Context, pipelineID, runID int64) (string, error)
}

type PipelineRunWriter interface {
	CreatePipelineRun(ctx context.Context, pipelineID int64, branch string) (*store.Run, error)
	UpdatePipelineRunStartedOn(
		ctx context.Context,
		runID int64,
		workingDirectory string,
		status store.RunStatus,
		startedOn *time.Time,
	) error
	UpdatePipelineRunEndedOn(
		ctx context.Context,
		runID int64,
		status store.RunStatus,
		artifacts *string,
		endedOn *time.Time,
	) error
	AppendPipelineRunOutput(context.Context, int64, string) error
	DeletePipelineRun(ctx context.Context, runID int64) error
}

type PipelineRunReader interface {
	GetPipelineRunByID(ctx context.Context, runID int64) (*store.Run, error)
	ListPipelineRuns(ctx context.Context, pipelineID int64) ([]store.Run, error)
	ListLatestPipelineRuns(ctx context.Context, id, limit int64) ([]store.Run, error)
	ListPipelineRunsPaginated(ctx context.Context, id, limit, offset int64) ([]store.Run, error)
	GetPipelineRunCount(ctx context.Context, id int64) (int64, error)
}

type RunQueueServicer interface {
	InitializeRunQueues(ctx context.Context) error
	AddRunQueues(ids []int64, queueSize int64)
	AddRunQueue(id, queueSize int64)
	GetPipelineRunQueue(id int64) (*service.RunQueue, bool)
	RemoveRunQueue(id int64)
	EnqueueRun(run *store.Run) error
	ShutdownRunQueue(id int64)
	ShutdownAll()
}

type PipelineServicer interface {
	PipelineWriter
	PipelineReader
	PipelineRunWriter
	PipelineRunReader
	RunQueueServicer
}

type PipelineHandler struct {
	pipelineService PipelineServicer
	apiKeyService   APIKeyServicer
}

func NewPipelineHandler(
	pipelineService PipelineServicer,
	apiKeyService APIKeyServicer,
) *PipelineHandler {
	return &PipelineHandler{
		pipelineService: pipelineService,
		apiKeyService:   apiKeyService,
	}
}

func (h *PipelineHandler) GetPipelinesPage(c echo.Context) error {
	u := getCtxUser(c)
	pipelines, agents, err := h.pipelineService.ListPipelinesAndAgents(c.Request().Context())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return newError(c, err,
			http.StatusInternalServerError, "something went wrong listing pipelines",
		)
	}

	if isHXRequest(c) {
		return render(c, pages.PipelinesMain(pipelines, agents))
	}
	return render(c, pages.PipelinesPage(u, pipelines, agents))
}

func (h *PipelineHandler) PostPipeline(c echo.Context) error {
	pp := new(PipelineParams)
	if err := c.Bind(pp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	p, err := h.pipelineService.CreatePipeline(
		c.Request().Context(),
		pp.PipelineAgentID,
		pp.Name,
		pp.Description,
		pp.Repository,
		pp.ScriptPath,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return newError(c, err,
				http.StatusConflict,
				fmt.Sprintf("An pipeline with the name %s already exists", pp.Name),
			)
		} else {
			return newError(c, err,
				http.StatusInternalServerError, "Unable to create pipeline",
			)
		}
	}

	return render(c, pages.PipelineCard(p))
}

func (h *PipelineHandler) GetPipelinePage(c echo.Context) error {
	u := getCtxUser(c)
	pp := new(PipelineParams)
	if err := c.Bind(pp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	p, agents, err := h.pipelineService.GetPipelineAndAgents(c.Request().Context(), pp.PipelineID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusNotFound, "pipeline not found")
		}
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong getting pipeline data",
		)
	}

	if isHXRequest(c) {
		return render(c, pages.PipelineMain(p, agents))
	}
	return render(c, pages.PipelinePage(u, p, agents))
}

func (h *PipelineHandler) PatchPipeline(c echo.Context) error {
	pp := new(PipelineParams)
	if err := c.Bind(pp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	pp.Name = strings.TrimSpace(pp.Name)
	pp.Description = strings.TrimSpace(pp.Description)
	pp.ScriptPath = strings.TrimSpace(pp.ScriptPath)

	err := h.pipelineService.UpdatePipeline(
		c.Request().Context(),
		pp.PipelineID,
		pp.PipelineAgentID,
		pp.Name,
		pp.Description,
		pp.Repository,
		pp.ScriptPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusNotFound, "pipeline not found")
		}
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong updating the pipeline",
		)
	}

	return renderToast(c, views.SuccessToast("Pipeline updated", 3000))
}

func (h *PipelineHandler) PatchPipelineSchedule(c echo.Context) error {
	pp := new(PipelineParams)
	if err := c.Bind(pp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	if err := h.pipelineService.UpdatePipelineSchedule(
		c.Request().Context(), pp.PipelineID, pp.Schedule, pp.ScheduleBranch,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusBadRequest, "invalid pipeline id")
		}
		return newError(
			c, err, http.StatusInternalServerError, "unable to update pipeline schedule",
		)
	}

	return renderToast(c, views.SuccessToast("pipeline schedule updated", 3000))
}

func (h *PipelineHandler) DeletePipeline(c echo.Context) error {
	pp := new(PipelineParams)
	if err := c.Bind(pp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	if pp.PipelineID == 0 {
		return newError(c, errors.New("pipeline id was zero"),
			http.StatusBadRequest, "invalid pipeline id",
		)
	}

	p, err := h.pipelineService.GetPipelineByID(c.Request().Context(), pp.PipelineID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusNotFound, "pipeline not found")
		}
		return newError(c, err, http.StatusInternalServerError, "unable to delete pipeline")
	}

	if err := h.pipelineService.DeletePipeline(
		c.Request().Context(), p.PipelineID,
	); err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to delete pipeline")
	}

	return hxRedirect(c, "/app/pipelines")
}

func (h *PipelineHandler) PostPipelineRun(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	p, err := h.pipelineService.GetPipelineByID(c.Request().Context(), rp.PipelineID)
	if err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to read pipeline data")
	}

	r, err := h.pipelineService.CreatePipelineRun(c.Request().Context(), p.PipelineID, rp.Branch)
	if err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to create pipeline run")
	}

	if err := h.pipelineService.EnqueueRun(r); err != nil {
		return newError(c, err, http.StatusInternalServerError, "pipeline run queue is full")
	}

	return hxRedirect(c, fmt.Sprintf("/app/pipelines/%d/runs/%d", p.PipelineID, r.RunID))
}

func (h *PipelineHandler) PostPipelineRunWebhookTrigger(c echo.Context) error {
	apiKeyValue := c.Request().Header.Get(internal.WebhookTriggerKeyHeader)
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, "invalid pipeline data",
		)
	}
	if rp.Branch == "" {
		rp.Branch = "main"
	}

	_, err := h.apiKeyService.GetAPIKeyByValue(c.Request().Context(), apiKeyValue)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, "invalid api key",
		)
	}

	p, err := h.pipelineService.GetPipelineByID(c.Request().Context(), rp.PipelineID)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusNotFound, "pipeline not found",
		)
	}

	r, err := h.pipelineService.CreatePipelineRun(
		c.Request().Context(), p.PipelineID, rp.Branch,
	)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError, "unable to create run",
		)
	}

	if err := h.pipelineService.EnqueueRun(r); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError, "pipeline run queue is full",
		).WithInternal(err)
	}

	return c.NoContent(http.StatusCreated)
}

func (h *PipelineHandler) GetPipelineRunPage(c echo.Context) error {
	u := getCtxUser(c)
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusInternalServerError, "unablea to get pipeline run page")
	}

	r, err := h.pipelineService.GetPipelineRunByID(c.Request().Context(), rp.RunID)
	if err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to read run data")
	}

	if isHXRequest(c) {
		return render(c, pages.PipelineRunPageMain(r))
	}
	return render(c, pages.PipelineRunPage(u, r))
}

func (h *PipelineHandler) GetPipelineRuns(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	runs, err := h.pipelineService.ListPipelineRuns(c.Request().Context(), rp.PipelineID)
	if err != nil {
		return newError(c, err, http.StatusBadRequest, "unable to list pipeline runs")
	}

	if len(runs) > 3 {
		runs = runs[0:3]
	}

	return render(c, pages.PipelineRuns(runs))
}

func (h *PipelineHandler) GetLatestPipelineRuns(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline data")
	}

	runs, err := h.pipelineService.ListLatestPipelineRuns(
		c.Request().Context(), rp.PipelineID, 3,
	)
	if err != nil {
		return newError(c, err, http.StatusBadRequest, "unable to list pipeline runs")
	}

	return render(c, pages.PipelineRuns(runs))
}

func (h *PipelineHandler) GetPipelineRunArtifacts(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline or run ID")
	}

	artifactsDir, err := h.pipelineService.CollectPipelineRunArtifacts(
		c.Request().Context(),
		rp.PipelineID,
		rp.RunID,
	)
	if err != nil {
		return newError(
			c, err,
			http.StatusInternalServerError, "unable to collect pipeline artifacts",
		)
	}

	archive := path.Join("artifacts", fmt.Sprintf("%d.zip", rp.RunID))
	if exists, _ := util.PathExists(archive); !exists {
		archive, err = util.ArchiveDirectory(artifactsDir)
		if err != nil {
			return newError(
				c, err,
				http.StatusInternalServerError, "unable to archive collected output",
			)
		}
	}

	return c.File(archive)
}

func (h *PipelineHandler) GetPipelineRunSSE(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline or run ID")
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	rq, ok := h.pipelineService.GetPipelineRunQueue(rp.PipelineID)
	if !ok {
		return nil
	}

	id := uuid.NewString()

	rq.StatusSSEClients.AddClient(id)
	defer rq.StatusSSEClients.RemoveClient(id)

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case out := <-rq.StatusSSEClients.GetClient(rp.RunID, id):
			b := new(bytes.Buffer)
			if err := pages.PipelineRunRow(out).Render(c.Request().Context(), b); err != nil {
				log.Println("err rendering pipeline run row:", err)
			} else {
				event := &Event{Data: b.Bytes()}
				if err := event.MarshalTo(w); err != nil {
					log.Println("err marshaling event data:", err)
				}
				w.Flush()
			}
		default:
			time.Sleep(3 * time.Second)
		}
	}
}

func (h *PipelineHandler) GetPipelineCardContent(c echo.Context) error {
	pp := new(PipelineParams)
	if err := c.Bind(pp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline id")
	}

	p, err := h.pipelineService.GetPipelineByID(c.Request().Context(), pp.PipelineID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusNotFound, "pipeline not found")
		}
		return newError(c, err, http.StatusInternalServerError, "unable to read pipeline by id")
	}

	return render(c, pages.PipelineCardContent(p))
}

func (h *PipelineHandler) GetPipelineRunsPage(c echo.Context) error {
	u := getCtxUser(c)
	lrp := new(ListRunsParams)
	if err := c.Bind(lrp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid request data")
	}

	count, err := h.pipelineService.GetPipelineRunCount(c.Request().Context(), lrp.PipelineID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return newError(c, err, http.StatusInternalServerError, "unable to count pipeline runs")
	}

	maxPages := count / maxRunsPerPage
	if maxPages >= 1 {
		maxPages++
	}

	runs, err := h.pipelineService.ListPipelineRunsPaginated(
		c.Request().Context(),
		lrp.PipelineID,
		maxRunsPerPage,
		(lrp.Page-1)*maxRunsPerPage,
	)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusInternalServerError, "error listing pipeline runs")
		}
	}

	if isHXRequest(c) {
		return render(c, pages.RunsMain(runs, lrp.PipelineID, max(lrp.Page, 1), maxPages))
	}
	return render(c, pages.RunsPage(u, runs, lrp.PipelineID, max(lrp.Page, 1), maxPages))
}

func (h *PipelineHandler) GetPipelineRunsList(c echo.Context) error {
	lrp := new(ListRunsParams)
	if err := c.Bind(lrp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid request data")
	}

	count, err := h.pipelineService.GetPipelineRunCount(c.Request().Context(), lrp.PipelineID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return newError(c, err, http.StatusInternalServerError, "unable to count pipeline runs")
	}

	maxPages := count / maxRunsPerPage
	if maxPages >= 1 {
		maxPages++
	}

	runs, err := h.pipelineService.ListPipelineRunsPaginated(
		c.Request().Context(),
		lrp.PipelineID,
		maxRunsPerPage,
		(lrp.Page-1)*maxRunsPerPage,
	)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusInternalServerError, "error listing pipeline runs")
		}
	}

	return render(c, pages.RunsPagination(runs, lrp.PipelineID, lrp.Page, maxPages))
}

func (h *PipelineHandler) GetPipelineRunOutput(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid request ID")
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	rq, ok := h.pipelineService.GetPipelineRunQueue(rp.PipelineID)
	if !ok {
		return nil
	}

	id := uuid.NewString()

	rq.OutputSSEClients.AddClient(id)
	defer rq.OutputSSEClients.RemoveClient(id)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request().Context().Done():
			// client disconnected
			return nil
		case out := <-rq.OutputSSEClients.GetClient(rp.RunID, id):
			// worker's output channel has data
			event := &Event{Data: []byte(out)}
			if err := event.MarshalTo(w); err != nil {
				log.Println("err marshaling event data:", err)
			}
			w.Flush()
		default:
			// no new data, just wait
			time.Sleep(1 * time.Second)
		}
	}
}

func (h *PipelineHandler) GetPipelineRunStatus(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid request ID")
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	rq, ok := h.pipelineService.GetPipelineRunQueue(rp.PipelineID)
	if !ok {
		return nil
	}

	id := uuid.NewString()
	rq.StatusSSEClients.AddClient(id)

	defer func() {
		rq.StatusSSEClients.RemoveClient(id)
	}()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case out := <-rq.StatusSSEClients.GetClient(rp.RunID, id):
			b := new(bytes.Buffer)
			if err := pages.PipelineRunPageStatusDiv(&out).Render(c.Request().Context(), b); err != nil {
				log.Println("err rendering run status div:", err)
			} else {
				event := &Event{Data: b.Bytes()}
				if err := event.MarshalTo(w); err != nil {
					log.Println("err marshaling event data:", err)
				}
				w.Flush()
			}
		default:
			time.Sleep(3 * time.Second)
		}
	}
}

func (h *PipelineHandler) PostCancelPipelineRun(c echo.Context) error {
	rp := new(RunParams)
	if err := c.Bind(rp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid pipeline or run ID")
	}

	rq, ok := h.pipelineService.GetPipelineRunQueue(rp.PipelineID)
	if !ok {
		return renderToast(c, views.FailureToast("pipline run queue not found", 3000))
	}

	rq.CancelRun(rp.RunID)

	return renderToast(c, views.SuccessToast("cancelling run...", 3000))
}

func SchedulePipelines(
	pipelineService PipelineServicer,
	pipelineScheduler gocron.Scheduler,
) {
	scheduledPipelines, err := pipelineService.ListScheduledPipelines(context.Background())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal(err)
	}
	for _, p := range scheduledPipelines {
		job, err := pipelineScheduler.NewJob(
			gocron.CronJob(*p.Schedule, false),
			gocron.NewTask(func() {
				r, err := pipelineService.CreatePipelineRun(
					context.Background(),
					p.PipelineID,
					*p.ScheduleBranch,
				)
				if err != nil {
					log.Println("err running scheduled job: ", err)
				}
				if err := pipelineService.EnqueueRun(r); err != nil {
					log.Println("pipeline run queue is full")
				}
			}),
		)
		if err != nil {
			log.Println("err re-scheduling pipeline:", err)
		}
		jobID := job.ID().String()
		if err := pipelineService.UpdatePipelineScheduleJobID(
			context.Background(), p.PipelineID, &jobID,
		); err != nil {
			log.Println("err updating re-scheduled pipeline job id:", err)
		}
	}
}
