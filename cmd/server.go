package cmd

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/config"
	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/agent"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/handlers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/repositories"
	"github.com/CyrilSbrodov/GopherAPIStore/pkg/client/postgresql"
)

type App struct {
	server http.Server
}

func NewApp() *App {
	return &App{}
}

func (a *App) Start() {
	//определение роутера
	router := chi.NewRouter()
	logger := loggers.NewLogger()
	cfg := config.ServerConfigInit()
	ticker := time.NewTicker(5 * time.Second)

	client, err := postgresql.NewClient(context.Background(), 5, &cfg, logger)
	checkError(err, logger)
	//определение БД
	store, err := repositories.NewPGSStore(client, &cfg, logger)
	checkError(err, logger)
	//определение агента
	accrualAgent := agent.NewAgent(store, *logger, cfg)
	//запуск агента в отдельной горутине с тикером
	go accrualAgent.Start(*ticker)
	sessionStore := sessions.NewCookieStore([]byte(cfg.SessionKey))
	//определение хендлера
	handler := handlers.NewHandler(store, logger, sessionStore)
	//регистрация хендлера
	handler.Register(router)
	a.server.Addr = cfg.Addr
	a.server.Handler = router

	logger.LogInfo("server is listen:", cfg.Addr, "start server")
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.LogInfo("server not started:", cfg.Addr, "")
	}
	logger.LogInfo("server is listen:", cfg.Addr, "start server")
}

func checkError(err error, logger *loggers.Logger) {
	if err != nil {
		logger.LogErr(err, "")
		os.Exit(1)
	}
}
