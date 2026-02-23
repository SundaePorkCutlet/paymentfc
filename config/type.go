package config

type Config struct {
	App      AppConfig      `yaml:"app" validate:"required"`
	Database DatabaseConfig `yaml:"database" validate:"required"`
	Redis    RedisConfig    `yaml:"redis" validate:"required"`
	Kafka    KafkaConfig    `yaml:"kafka" validate:"required"`
	Xendit   XenditConfig   `yaml:"xendit"`
	Mongo    MongoConfig    `yaml:"mongo"`
	Toggle   ToggleConfig   `yaml:"toggle"`
	GRPC     GRPCConfig     `yaml:"grpc"`
	Vault    VaultConfig    `yaml:"vault"`
}

type VaultConfig struct {
	Host  string `yaml:"host" validate:"required"`
	Token string `yaml:"token" validate:"required"`
	Path  string `yaml:"path" validate:"required"`
}

type SecretVaultConfig struct {
	DatabaseSecret DatabaseSecretConfig `json:"database"`
	RedisSecret    RedisSecretConfig    `json:"redis"`
	JWTSecret      string               `json:"jwt_secret"`
	GRPCCredential string               `json:"grpcCredentials"`
	XenditSecret   XenditSecretConfig   `json:"xendit"`
}

type DatabaseSecretConfig struct {
	Password string `json:"password"`
}

type RedisSecretConfig struct {
	Password string `json:"password"`
}

type XenditSecretConfig struct {
	SecretAPIKey string `json:"secret_api_key"`
	WebhookToken string `json:"webhook_token"`
}

type GRPCConfig struct {
	UserServiceAddr string `yaml:"user_service_addr" mapstructure:"user_service_addr"`
	Credentials     string `yaml:"credentials" mapstructure:"credentials"`
}

// ToggleConfig feature flags (실시간 인보이스 vs 배치)
type ToggleConfig struct {
	DisableCreateInvoiceDirectly bool `yaml:"disable_create_invoice_directly" mapstructure:"disable_create_invoice_directly"` // true: payment_requests 저장만, 인보이스는 배치에서 생성
}

type XenditConfig struct {
	XenditAPIKey       string `yaml:"secret_api_key" mapstructure:"secret_api_key" validate:"required"`
	XenditWebhookToken string `yaml:"webhook_token" mapstructure:"webhook_token" validate:"required"`
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

type MongoConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}
