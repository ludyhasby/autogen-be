package config

import (
	"log/slog"
	"logisfy/helper/crypto"
	handler "logisfy/internal/delivery/http"
	internaldeliveryhttproute "logisfy/internal/delivery/http/route"
	"logisfy/internal/repository"
	"logisfy/internal/usecase"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	DB       *gorm.DB
	App      *fiber.App
	Log      *slog.Logger
	Validate *validator.Validate
	Viper    *viper.Viper
	Env      *Env
}

func Bootstrap(config *BootstrapConfig) {
	// setup crypto
	aesGcm := crypto.NewAES256GCM(config.Env.AESSecret, config.Env.AESPepper)

	// Ambil *sql.DB dari GORM untuk keperluan pgx CopyIn
	sqlDB, err := config.DB.DB()
	if err != nil {
		panic("Bootstrap: gagal mendapatkan *sql.DB dari GORM: " + err.Error())
	}

	// setup repository
	userRepo := repository.NewUserRepository(config.Log)
	amrRepo := repository.NewAMRRepository(config.Log)
	amrDetailRepo := repository.NewAMRDetailRepository(config.Log, sqlDB)
	amrConfigRepo := repository.NewAMRConfigRepository(config.Log)
	amrWeightRepo := repository.NewAMRWeightConfigRepository(config.Log)
	amrDetailResultRepo := repository.NewAMRDetailResultRepository(config.Log)
	newsRepo := repository.NewNewsRepository(config.Log)

	// setup useCase
	authUseCase := usecase.NewAuthUseCase(config.DB, config.Log, config.Validate, config.Env.SecretKey, userRepo)
	amrUseCase := usecase.NewAMRUseCase(config.DB, config.Log, config.Validate, aesGcm, config.Env.Location, config.Env.NumberBatch, config.Env.DeletedDurationInHour, amrRepo, amrDetailRepo, amrConfigRepo, amrWeightRepo, amrDetailResultRepo, config.Env.MaxConcurrentUploads, config.Env.UploadTempDir)
	newsUseCase := usecase.NewNewsUseCase(config.DB, config.Log, config.Validate, newsRepo)

	// setup controller
	authHandler := handler.NewAuthHandler(config.Log, authUseCase)
	amrHandler := handler.NewAMRHandler(config.Log, amrUseCase)
	newsHandler := handler.NewNewsHandler(config.Log, newsUseCase)

	routeConfig := internaldeliveryhttproute.RouteConfig{
		App:         config.App,
		SecretKey:   config.Env.SecretKey,
		AuthHandler: authHandler,
		AMRHandler:  amrHandler,
		NewsHandler: newsHandler,
	}

	routeConfig.Setup()
}
