package main

import (
	"context"
	"fmt"
	internalconfig "logisfy/internal/config"
	"logisfy/internal/repository"
	"logisfy/internal/worker"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Time
	loc, _ := time.LoadLocation("Asia/Jakarta")
	time.Local = loc

	// config
	viperConfig := internalconfig.NewViper()
	envConfig := internalconfig.NewEnv(viperConfig)
	logConfig := internalconfig.NewSlog(envConfig)

	dbConfig := internalconfig.NewDatabase(envConfig, logConfig)

	// Repository
	newsRepo := repository.NewNewsRepository(logConfig)

	// Worker UseCase
	newsFetchWorker := worker.NewNewsWorker(dbConfig, logConfig, newsRepo, envConfig.NewsAPI, envConfig.NewsCronExpr)

	// Worker
	go newsFetchWorker.FetchNewNews(ctx)

	// channel
	terminateSignals := make(chan os.Signal, 1)
	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	stop := false
	for !stop {
		select {
		case s := <-terminateSignals:
			logConfig.Info("got one of stop signals, shutting down worker gracefully, SIGNAL NAME : %s", s.String())
			cancel()
			stop = true
		}
	}

	fmt.Println("Worker stopped gracefully")
}
