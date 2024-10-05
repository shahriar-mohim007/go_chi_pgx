package main

import (
	"chi-pgx/pkg/api/handler"
	"chi-pgx/pkg/config"
	"chi-pgx/pkg/infrastructure/database"
	"chi-pgx/pkg/infrastructure/router"
	"chi-pgx/pkg/utils/jsonlog"
	"chi-pgx/pkg/utils/usecase"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {

	cfg := config.LoadConfig()
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	taskRepo, err := database.NewPgxTaskRepository(cfg)
	if err != nil {
		logger.PrintFatal(err, map[string]string{"context": "initialize task repository"})
	}
	taskUsecase := usecase.NewTaskUsecase(taskRepo)
	taskHandler := handler.NewTaskHandler(taskUsecase)
	r := router.NewRouter(taskHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HostPort),
		Handler:      r,
		ErrorLog:     log.New(logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit
		logger.PrintInfo("Shutting down server. Received signal: "+sig.String(), nil)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
			return
		}

		var wg sync.WaitGroup
		wg.Wait()

		shutdownError <- nil
	}()

	err = srv.ListenAndServe()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.PrintFatal(err, map[string]string{"context": "server startup"})
	}

	if err = <-shutdownError; err != nil {
		logger.PrintError(err, map[string]string{"context": "server shutdown"})
	} else {
		logger.PrintInfo("Server shutdown gracefully", map[string]string{"context": "shutdown"})
	}

}
