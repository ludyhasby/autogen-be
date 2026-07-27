package main

import (
	"context"
	"fmt"
	helpermigration "logisfy/helper/migration"
	internalconfig "logisfy/internal/config"
	"os"
	"os/signal"
	"syscall"
)

//	@title			Autogen API
//	@version		1.0
//	@description	This is Api for Autogen.
//	@termsOfService	http://swagger.io/terms/

//	@securityDefinitions.http	Bearer
//	@in							header
//	@name						Authorization

// @schemes	https
func main() {
	// === ROOT CONTEXT SHUTDOWN ===
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	viperConfig := internalconfig.NewViper()
	envConfig := internalconfig.NewEnv(viperConfig)
	logConfig := internalconfig.NewSlog(envConfig)
	dbConfig := internalconfig.NewDatabase(envConfig, logConfig)

	validateConfig := internalconfig.NewValidator(envConfig)

	// Init Tracer
	tracerCleanup, err := internalconfig.NewTracer(envConfig)
	if err != nil {
		logConfig.Error("Error init tracer", err)
		os.Exit(1)
	}
	defer func() {
		if err := tracerCleanup(ctx); err != nil {
			logConfig.Error("Error shutting down tracer", err)
		}
	}()

	// Init Fiber
	fiberConfig := internalconfig.NewFiber(envConfig)

	// Migrate DB
	if envConfig.DBMigrate {
		helpermigration.AutoMigrate(dbConfig)
	}

	// Init Bootstrap Server
	internalconfig.Bootstrap(&internalconfig.BootstrapConfig{
		DB:       dbConfig,
		App:      fiberConfig,
		Log:      logConfig,
		Validate: validateConfig,
		Viper:    viperConfig,
		Env:      envConfig,
	})

	// Running App
	go func() {
		addr := fmt.Sprintf(":%d", envConfig.WebPort)
		if err = fiberConfig.Listen(addr); err != nil {
			logConfig.Error("Error starting server", err)
			os.Exit(1)
		}
	}()

	// === WAIT SIGNAL & GRACEFUL SHUTDOWN ===
	<-ctx.Done()
	logConfig.Info("Shutting down server...")
	if err := fiberConfig.Shutdown(); err != nil {
		logConfig.Error("Error shutting down server", err)
	}
}
