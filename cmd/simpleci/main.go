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
	settings.ReadDotenv("./.env")
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
	credentialSvc := service.NewCredentialService(
		store.NewCredentialSQLiteStore(rdb, rwdb),
		security.NewAESEncrypter([]byte(os.Getenv("SIMPLECI_HASH_KEY"))),
	)
	agentSvc := service.NewAgentService(store.NewAgentSQLiteStore(rdb, rwdb), credentialSvc)
	apiKeySvc := service.NewAPIKeyService(
		store.NewAPIKeySQLiteStore(rdb, rwdb),
		service.NewUUIDGen(),
	)
	pipelineSvc := service.NewPipelineService(
		store.NewPipelineSQLiteStore(rdb, rwdb),
		store.NewRunSQLiteStore(rdb, rwdb),
		credentialSvc,
		agentSvc,
		apiKeySvc,
		pipelineScheduler,
	)
	if err := pipelineSvc.InitializeRunQueues(context.Background()); err != nil {
		log.Fatal(err)
	}

	userSvc.InitializeSuperuser(context.Background())

	store.KVStore = store.NewKeyValueStore()
	store.KVStore.ScheduleDailyCleanUp(kvStoreScheduler)
	kvStoreScheduler.Start()

	handler.SchedulePipelines(pipelineSvc, pipelineScheduler)
	pipelineScheduler.Start()

	e := setupEcho()
	g := e.Group("", handler.SessionMiddleware(userSvc, cookieSvc))
	handler.SetupAuthRoutes(g, userSvc, cookieSvc)
	handler.SetupAppRoutes(g)
	handler.SetupUserRoutes(g, userSvc, cookieSvc)
	handler.SetupCredentialRoutes(g, credentialSvc)
	handler.SetupAgentRoutes(g, agentSvc)
	handler.SetupPipelineRoutes(g, pipelineSvc)
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

	publicFS := echo.MustSubFS(assets.PublicFS, "public")
	e.StaticFS("/", publicFS)
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/favicon.svg")
	})

	return e
}
