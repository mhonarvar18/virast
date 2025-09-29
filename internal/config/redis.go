package config

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// RedisClient متغیر برای دسترسی به Redis
var RedisClient *redis.Client
var ctx = context.Background()

// InitRedis اتصال به Redis را راه‌اندازی می‌کند
func InitRedis() {
	redisDBStr := os.Getenv("REDIS_DB")
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		redisDB = 0 // مقدار پیش‌فرض دیتابیس Redis
	}
	// تنظیمات اتصال به Redis
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"), // آدرس Redis
		Password: os.Getenv("REDIS_PASSWORD"),               // رمز عبور
		DB:       redisDB,                // شماره دیتابیس
	})

	// بررسی اتصال به Redis
	s, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Error connecting to Redis:", err)
	}
	log.Println("Redis ping response:", s)
	log.Println("Connected to Redis")
}
