package config

import (
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB متغیر برای دسترسی به دیتابیس
var DB *gorm.DB
var err error

// InitDB اتصال به دیتابیس MySQL را راه‌اندازی می‌کند
func InitDB() {
	// تنظیمات اتصال به دیتابیس
	dsn := os.Getenv("DB_DSN")
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	log.Println("Database connected")

}
