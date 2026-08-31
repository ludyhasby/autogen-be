package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type Env struct {
	AppName     string
	PreFork     bool
	LogLevel    string
	WebPort     int
	Location    *time.Location
	FrontEndURL string

	DBMigrate  bool
	DBUser     string
	DBPass     string
	DBHost     string
	DBPort     string
	DBName     string
	DBSslMode  string
	DBIdleConn int
	DBMaxConn  int
	DBMaxLife  time.Duration

	SecretKey string
	AESSecret string
	AESPepper string

	RowMaxLimit           int
	NumberBatch           int
	DeletedDurationInHour int
	MaxConcurrentUploads  int
	UploadTempDir         string

	OTelEndpoint string

	NewsAPI      string
	NewsCronExpr string

	NumberBatchAMRDelete int
	AMRCronExpr          string

	EmailAddress string
	DialHost     string
	DialPassword string
	DialUser     string
	DialPort     int
}

func NewEnv(viper *viper.Viper) *Env {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	timezone := os.Getenv("TIMEZONE")
	if l, err := time.LoadLocation(timezone); err == nil {
		loc = l
	}
	return &Env{
		AppName:     viper.GetString("APP_NAME"),
		PreFork:     viper.GetBool("PRE_FORK"),
		LogLevel:    viper.GetString("LOG_LEVEL"),
		WebPort:     viper.GetInt("WEB_PORT"),
		Location:    loc,
		FrontEndURL: viper.GetString("FRONT_END_URL"),

		DBMigrate:  viper.GetBool("DB_MIGRATE"),
		DBUser:     viper.GetString("DB_USER"),
		DBPass:     viper.GetString("DB_PASS"),
		DBHost:     viper.GetString("DB_HOST"),
		DBPort:     viper.GetString("DB_PORT"),
		DBName:     viper.GetString("DB_NAME"),
		DBSslMode:  viper.GetString("DB_SSL_MODE"),
		DBIdleConn: viper.GetInt("DB_IDLE_CONN"),
		DBMaxConn:  viper.GetInt("DB_MAX_CONN"),
		DBMaxLife:  viper.GetDuration("DB_MAX_LIFE"),

		SecretKey: viper.GetString("SECRET_KEY"),
		AESSecret: viper.GetString("AES_SECRET"),
		AESPepper: viper.GetString("AES_PEPPER"),

		RowMaxLimit:           viper.GetInt("ROW_MAX_LIMIT"),
		NumberBatch:           viper.GetInt("NUMBER_BATCH"),
		DeletedDurationInHour: viper.GetInt("DELETED_DURATION_IN_HOUR"),
		MaxConcurrentUploads:  viper.GetInt("MAX_CONCURRENT_UPLOADS"),
		UploadTempDir:         viper.GetString("UPLOAD_TEMP_DIR"),

		OTelEndpoint: viper.GetString("OTEL_ENDPOINT"),
		NewsAPI:      viper.GetString("NEWS_API"),
		NewsCronExpr: viper.GetString("NEWS_CRON_EXPR"),

		NumberBatchAMRDelete: viper.GetInt("NUMBER_BATCH_AMR_DELETE"),
		AMRCronExpr:          viper.GetString("AMR_CRON_EXPR"),

		EmailAddress: viper.GetString("EMAIL_ADDRESS"),
		DialHost:     viper.GetString("DIAL_HOST"),
		DialPassword: viper.GetString("DIAL_PASSWORD"),
		DialUser:     viper.GetString("DIAL_USER"),
		DialPort:     viper.GetInt("DIAL_PORT"),
	}
}
