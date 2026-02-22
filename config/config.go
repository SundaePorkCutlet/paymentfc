package config

import (
	"log"

	"github.com/spf13/viper"
)

func LoadConfig() Config {
	var cfg Config

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./files/config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// 환경변수 오버라이드 (Docker Compose 등)
	viper.AutomaticEnv()
	viper.BindEnv("xendit.secret_api_key", "XENDIT_SECRET_API_KEY")
	viper.BindEnv("xendit.webhook_token", "XENDIT_WEBHOOK_TOKEN")
	viper.BindEnv("kafka.broker", "KAFKA_BROKER")
	viper.BindEnv("mongo.uri", "MONGO_URI")

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Xendit API 키 로드 확인
	if cfg.Xendit.XenditAPIKey != "" {
		log.Printf("Xendit API key loaded (length: %d)", len(cfg.Xendit.XenditAPIKey))
	} else {
		log.Println("Warning: Xendit API key is empty")
	}

	return cfg
}

func GetJwtSecret() string {
	return viper.GetString("secret.jwt_secret")
}
