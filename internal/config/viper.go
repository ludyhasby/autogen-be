package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log/slog"
	"os"
)

func NewViper() *viper.Viper {

	// load file .env
	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file", err)
		os.Exit(1)
	}

	// Init viper
	config := viper.New()
	config.AutomaticEnv()

	return config
}
