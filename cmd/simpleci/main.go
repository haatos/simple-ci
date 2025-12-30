package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/assets"
	"github.com/haatos/simple-ci/internal/handler"
	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/settings"
	"github.com/haatos/simple-ci/internal/store"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "modernc.org/sqlite"
)

func main() {
	internal.InitializeConfiguration()
	settings.ReadDotenv(internal.DotEnvPath)
	settings.Settings = settings.NewSettings()
	hashKey, blockKey := security.NewKeys()
	rdb := store.InitDatabase(true)
	defer rdb.Close()
	rwdb := store.InitDatabase(false)
	defer rwdb.Close()
	store.RunMigrations(rwdb, internal.MigrationsDir)

	scheduler := service.NewScheduler()
	defer scheduler.Shutdown()

	userStore := store.NewUserSQLiteStore(rdb, rwdb)
	credentialStore := store.NewCredentialSQLiteStore(rdb, rwdb)
	agentStore := store.NewAgentSQLiteStore(rdb, rwdb)
	pipelineStore := store.NewPipelineSQLiteStore(rdb, rwdb)
	runStore := store.NewRunSQLiteStore(rdb, rwdb)
	apiKeyStore := store.NewAPIKeySQLiteStore(rdb, rwdb)
	aesEncrypter := security.NewAESEncrypter([]byte(os.Getenv("SIMPLECI_HASH_KEY")))

	cookieSvc := service.NewCookieService(hashKey, blockKey)
	userSvc := service.NewUserService(userStore)
	credentialSvc := service.NewCredentialService(
		credentialStore,
		aesEncrypter,
	)
	agentSvc := service.NewAgentService(agentStore, credentialStore, aesEncrypter)
	_, _ = agentSvc.CreateControllerAgent(context.Background())
	apiKeySvc := service.NewAPIKeyService(
		apiKeyStore,
		service.NewUUIDGen(),
	)
	pipelineSvc := service.NewPipelineService(
		pipelineStore,
		runStore,
		credentialStore,
		agentStore,
		apiKeyStore,
		scheduler,
		aesEncrypter,
	)
	if err := pipelineSvc.InitializeRunQueues(context.Background()); err != nil {
		log.Fatal(err)
	}

	userSvc.InitializeSuperuser(context.Background())

	kvStore := store.NewKeyValueStore()
	kvStore.ScheduleDailyCleanUp(scheduler)
	handler.SchedulePipelines(pipelineSvc, scheduler)
	scheduler.Start()

	e := setupEcho()
	g := e.Group("", handler.SessionMiddleware(userSvc, cookieSvc))
	handler.SetupAuthRoutes(g, userSvc, cookieSvc)
	handler.SetupAppRoutes(g)
	handler.SetupUserRoutes(g, userSvc, cookieSvc)
	handler.SetupCredentialRoutes(g, credentialSvc)
	handler.SetupAgentRoutes(g, agentSvc)
	handler.SetupPipelineRoutes(g, pipelineSvc, apiKeySvc)
	handler.SetupAPIKeyRoutes(g, apiKeySvc)

	internal.GracefulShutdown(e, settings.Settings.Port)
}

func setupEcho() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = handler.ErrorHandler
	e.Use(
		middleware.CORSWithConfig(internal.GetCORSConfig()),
		middleware.RateLimiterWithConfig(internal.GetRateLimiterConfig()),
	)

	publicFS := echo.MustSubFS(assets.PublicFS, internal.PublicDir)
	e.StaticFS("/", publicFS)
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/favicon.svg")
	})

	return e
}
