package config

import (
	"log"
	"go.uber.org/zap"
)

var Logger *zap.Logger

func InitLogger() {
	var err error
	// می‌توان logger production یا development انتخاب کرد
	Logger, err = zap.NewDevelopment() // برای توسعه
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer Logger.Sync() // flush buffer

	Logger.Info("✅ Zap logger initialized")
}
