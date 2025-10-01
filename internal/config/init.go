package config

import (
	"github.com/joho/godotenv"
	"os"
)

func Init() {
	// بارگذاری .env
	if err := godotenv.Load(); err != nil {
		Logger.Info("No .env file found, using system environment variables")
	}

	// مثال خواندن مقادیر
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		Logger.Fatal("DB_DSN is not set")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		Logger.Fatal("REDIS_ADDR is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		Logger.Fatal("JWT_SECRET is not set")
	}
}
