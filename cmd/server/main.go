package main

import (
	"context"
	"log"
	"net/http"

	assets "github.com/haatos/simple-ci"
	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/handler"
	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/settings"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/types"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "modernc.org/sqlite"
)

func main() {
	settings.ReadDotenv()
	settings.Settings = settings.NewSettings()
	hashKey, blockKey := security.NewKeys()
	rdb := store.InitDatabase(true)
	defer rdb.Close()
	rwdb := store.InitDatabase(false)
	defer rwdb.Close()
	store.RunMigrations(rwdb, "migrations")

	kvStoreScheduler := service.NewScheduler()
	defer kvStoreScheduler.Shutdown()
	pipelineScheduler := service.NewScheduler()
	defer pipelineScheduler.Shutdown()

	cookieSvc := service.NewCookieService(hashKey, blockKey)
	userSvc := service.NewUserService(store.NewUserSQLiteStore(rdb, rwdb))
	credSvc := service.NewCredentialService(
		store.NewCredentialSQLiteStore(rdb, rwdb),
		security.NewAESEncrypter(),
	)
	agentSvc := service.NewAgentService(store.NewAgentSQLiteStore(rdb, rwdb), credSvc)
	apiKeySvc := service.NewAPIKeyService(
		store.NewAPIKeySQLiteStore(rdb, rwdb),
		service.NewUUIDGen(),
	)
	pipelineSvc := service.NewPipelineService(
		store.NewPipelineSQLiteStore(rdb, rwdb),
		store.NewRunSQLiteStore(rdb, rwdb),
		credSvc,
		agentSvc,
		apiKeySvc,
		pipelineScheduler,
	)
	if err := pipelineSvc.InitializeRunQueues(context.Background()); err != nil {
		log.Fatal(err)
	}

	userSvc.InitializeSuperuser(context.Background())

	authH := handler.NewAuthHandler(userSvc, cookieSvc)
	userH := handler.NewUserHandler(userSvc, cookieSvc)
	credH := handler.NewCredentialHandler(credSvc)
	agentH := handler.NewAgentHandler(agentSvc)
	pipelineH := handler.NewPipelineHandler(pipelineSvc)
	apiKeyH := handler.NewAPIKeyHandler(apiKeySvc)

	store.KVStore = store.NewKeyValueStore()
	store.KVStore.ScheduleDailyCleanUp(kvStoreScheduler)
	kvStoreScheduler.Start()

	pipelineH.SchedulePipelines(pipelineScheduler)
	pipelineScheduler.Start()

	e := setupEcho()
	router := e.Group("", authH.SessionMiddleware)

	router.GET("", authH.GetLoginPage, handler.AlreadyLoggedIn)
	router.GET("/auth/logout", authH.GetLogOut)
	router.POST("/auth/login", authH.PostLogin)
	router.GET("/auth/set-password", authH.GetSetPasswordPage)
	router.POST("/auth/set-password", authH.PostSetPassword)

	app := router.Group("/app", handler.IsAuthenticated)
	app.GET("", handler.GetAppPage)

	app.GET("/user", userH.GetProfilePage)
	app.GET("/users", userH.GetUsers, handler.RoleMiddleware(types.Admin))
	app.POST("/users", userH.PostUsers, handler.RoleMiddleware(types.Admin))
	app.DELETE("/users/:user_id", userH.DeleteUser, handler.RoleMiddleware(types.Admin))
	app.PATCH("/users/:user_id/change-password", userH.PatchChangeUserPassword)
	app.PATCH("/users/:user_id/password", userH.PatchUserPassword)
	app.PATCH(
		"/users/:user_id/reset-password",
		userH.PatchResetUserPassword,
		handler.RoleMiddleware(types.Admin),
	)
	app.PATCH("/users/:user_id/role", userH.PatchUserRole, handler.RoleMiddleware(types.Superuser))

	app.GET("/credentials", credH.GetCredentialsPage)
	app.POST("/credentials", credH.PostCredentials, handler.RoleMiddleware(types.Admin))
	app.PATCH("/credentials", credH.PatchCredential, handler.RoleMiddleware(types.Admin))
	app.DELETE(
		"/credentials/:credential_id",
		credH.DeleteCredential,
		handler.RoleMiddleware(types.Admin),
	)
	app.GET("/credentials/:credential_id", credH.GetCredentialPage)

	app.GET("/agents", agentH.GetAgentsPage)
	app.POST("/agents", agentH.PostAgent)
	app.PATCH("/agents", agentH.PatchAgent)
	app.GET("/agents/:agent_id", agentH.GetAgentPage)
	app.DELETE("/agents/:agent_id", agentH.DeleteAgent)
	app.POST("/agents/:agent_id/test-connection", agentH.PostTestAgentConnection)

	app.GET("/pipelines", pipelineH.GetPipelinesPage)
	app.POST("/pipelines", pipelineH.PostPipeline)
	app.PATCH("/pipelines", pipelineH.PatchPipeline)
	app.GET("/pipelines/:pipeline_id", pipelineH.GetPipelinePage)
	app.GET("/pipelines/:pipeline_id/card-content", pipelineH.GetPipelineCardContent)
	app.DELETE("/pipelines/:pipeline_id", pipelineH.DeletePipeline)
	app.PATCH("/pipelines/:pipeline_id/schedule", pipelineH.PatchPipelineSchedule)

	app.GET("/pipelines/:pipeline_id/latest-runs", pipelineH.GetLatestPipelineRuns)
	app.POST("/pipelines/:pipeline_id/runs", pipelineH.PostPipelineRun)
	app.POST(
		"/pipelines/:pipeline_id/runs/webhook-trigger/:branch",
		pipelineH.PostPipelineRunWebhookTrigger,
	)
	app.GET("/pipelines/:pipeline_id/runs/:run_id", pipelineH.GetPipelineRunPage)
	app.GET("/pipelines/:pipeline_id/runs", pipelineH.GetPipelineRunsPage)
	app.GET("/pipelines/:pipeline_id/runs-list", pipelineH.GetPipelineRunsList)
	app.GET("/pipelines/:pipeline_id/runs/:run_id/sse", pipelineH.GetPipelineRunSSE)
	app.GET("/pipelines/:pipeline_id/runs/:run_id/output", pipelineH.GetRunOutput)
	app.GET("/pipelines/:pipeline_id/runs/:run_id/status", pipelineH.GetRunStatus)
	app.GET("/pipelines/:pipeline_id/runs/:run_id/artifacts", pipelineH.GetPipelineRunArtifacts)
	app.POST("/pipelines/:pipeline_id/runs/:run_id/cancel", pipelineH.PostCancelPipelineRun)

	app.GET("/api-keys", apiKeyH.GetAPIKeysPage, handler.RoleMiddleware(types.Admin))
	app.POST("/api-keys", apiKeyH.PostAPIKey, handler.RoleMiddleware(types.Admin))
	app.DELETE("/api-keys/:id", apiKeyH.DeleteAPIKey, handler.RoleMiddleware(types.Admin))

	internal.GracefulShutdown(e, settings.Settings.Port)
}

func setupEcho() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = handler.ErrorHandler
	e.Use(
		middleware.CORSWithConfig(internal.GetCORSConfig()),
		middleware.RateLimiterWithConfig(internal.GetRateLimiterConfig()),
	)

	publicFS := echo.MustSubFS(assets.PublicFS, "public")
	e.StaticFS("/", publicFS)
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/favicon.svg")
	})

	return e
}
