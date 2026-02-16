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

	// 환경변수로 Xendit 시크릿 오버라이드
	viper.BindEnv("xendit.secret_api_key", "XENDIT_SECRET_API_KEY")
	viper.BindEnv("xendit.webhook_token", "XENDIT_WEBHOOK_TOKEN")

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	return cfg
}

func GetJwtSecret() string {
	return viper.GetString("secret.jwt_secret")
}
