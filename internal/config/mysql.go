package config

import (
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"os"
	"time"
)

// DB متغیر برای دسترسی به دیتابیس
var DB *gorm.DB
var err error

// InitDB اتصال به دیتابیس MySQL را راه‌اندازی می‌کند
func InitDB() {
	// تنظیمات اتصال به دیتابیس
	dsn := os.Getenv("DB_DSN")

	newLogger := gormlogger.New(
		zap.NewStdLog(Logger), // convert zap to stdlog
		gormlogger.Config{
			SlowThreshold: time.Second,
			LogLevel:      gormlogger.Info,
			Colorful:      true,
		},
	)

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 newLogger,
	})
	if err != nil {
		Logger.Fatal("failed to connect database", zap.Error(err))
	}
	Logger.Info("Database connected")

}
