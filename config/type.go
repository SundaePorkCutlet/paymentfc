package config

type Config struct {
	App      AppConfig      `yaml:"app" validate:"required"`
	Database DatabaseConfig `yaml:"database" validate:"required"`
	Redis    RedisConfig    `yaml:"redis" validate:"required"`
	Kafka    KafkaConfig    `yaml:"kafka" validate:"required"`
	Xendit   XenditConfig   `yaml:"xendit" validate:"required"`
}

type XenditConfig struct {
	XenditAPIKey       string `yaml:"secret_api_key" validate:"required"`
	XenditWebhookToken string `yaml:"webhook_token" validate:"required"`
}

type AppConfig struct {
	Port string `yaml:"port" validate:"required"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host" validate:"required"`
	Port     string `yaml:"port" validate:"required"`
	User     string `yaml:"user" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	Name     string `yaml:"name" validate:"required"`
}

type RedisConfig struct {
	Host     string `yaml:"host" validate:"required"`
	Port     string `yaml:"port" validate:"required"`
	Password string `yaml:"password" validate:"required"`
}

type KafkaConfig struct {
	Broker  string              `yaml:"broker" validate:"required"`
	Topics  []map[string]string `yaml:"topics" validate:"required"`
	GroupID string              `yaml:"group_id" validate:"required"`
}
