package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stellhub/stellar"
	"github.com/stellhub/stellatlas-service/internal/cmdb"
)

const serviceName = "stellatlas-service"

type Starter struct {
	config  stellar.Config
	service *cmdb.Service
}

func NewStarter() *Starter {
	return &Starter{}
}

func (s *Starter) Name() string {
	return "stellatlas-api"
}

func (s *Starter) Condition(ctx stellar.StarterContext) bool {
	s.config = ctx.Config()
	return true
}

func (s *Starter) Init(_ context.Context, app *stellar.App) error {
	var repository cmdb.Repository
	if db, ok := app.PostgreSQLDB(); ok {
		repository = cmdb.NewPostgreSQLRepository(db)
	}

	var cache cmdb.Cache
	if client, ok := app.RedisClient(); ok {
		cache = cmdb.NewRedisCache(client, cmdb.RedisCacheOptions{
			Prefix: "cmdb",
		})
	}

	s.service = cmdb.NewService(repository, cache)
	registerRoutes(app.HTTP(), s.config, s.service)
	return nil
}

func (s *Starter) Start(context.Context) error {
	return nil
}

func (s *Starter) Stop(context.Context) error {
	return nil
}

func (s *Starter) Health(context.Context) stellar.HealthCheck {
	return stellar.HealthCheck{
		Name:    s.Name(),
		Status:  stellar.HealthStatusUp,
		Message: "StellAtlas API routes registered",
	}
}

func NewHandler() http.Handler {
	app := stellar.New(defaultConfig(), stellar.WithStarter(NewStarter()))
	if err := app.Start(context.Background()); err != nil {
		return startupErrorHandler{err: err}
	}
	return app.Handler()
}

func defaultConfig() stellar.Config {
	return stellar.Config{
		AppName:     serviceName,
		Environment: stellar.EnvDev,
		Zone:        "local",
	}
}

type startupErrorHandler struct {
	err error
}

func (h startupErrorHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, fmt.Sprintf("start StellAtlas handler: %v", h.err), http.StatusInternalServerError)
}
