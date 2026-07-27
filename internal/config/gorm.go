package config

import (
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
	"os"
	"time"
)

func NewDatabase(env *Env, log *slog.Logger) *gorm.DB {

	// generate dsn
	dsn := "host=" + env.DBHost + " user=" + env.DBUser + " password=" + env.DBPass + " dbname=" + env.DBName + " port=" + env.DBPort + " sslmode=" + env.DBSslMode

	// Get Log Level
	levelLog := GetLevelLog(env.LogLevel)
	gormLogger := slogGorm.New(
		slogGorm.WithHandler(log.Handler()),
		slogGorm.WithTraceAll(),
		slogGorm.SetLogLevel(slogGorm.DefaultLogType, levelLog),
	)

	// Open Connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Error("Error connecting to database", err)
		os.Exit(1)
	}

	// Connect DB
	conn, err := db.DB()
	if err != nil {
		log.Error("Error connecting to database", err)
		os.Exit(1)
	}

	conn.SetMaxIdleConns(env.DBIdleConn)
	conn.SetMaxOpenConns(env.DBMaxConn)
	conn.SetConnMaxLifetime(time.Second * env.DBMaxLife)

	return db
}
