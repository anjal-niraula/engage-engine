package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Primary       Primary              `koanf:"primary" validate:"required"`
	Server        ServerConfig         `koanf:"server" validate:"required"`
	Database      DatabaseConfig       `koanf:"database" validate:"required"`
	Mongo         MongoConfig          `koanf:"mongo"`
	Redis         RedisConfig          `koanf:"redis" validate:"required"`
	Kafka         KafkaConfig          `koanf:"kafka"`
	Observability *ObservabilityConfig `koanf:"observability"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

type DatabaseConfig struct {
	Host            string `koanf:"host" validate:"required"`
	Port            int    `koanf:"port" validate:"required"`
	User            string `koanf:"user" validate:"required"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name" validate:"required"`
	SSLMode         string `koanf:"ssl_mode" validate:"required"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int    `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"required"`
}

// DSN builds a Postgres connection URL with credentials safely escaped.
func (d DatabaseConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     d.Host + ":" + strconv.Itoa(d.Port),
		Path:     d.Name,
		RawQuery: url.Values{"sslmode": {d.SSLMode}}.Encode(),
	}
	return u.String()
}

type MongoConfig struct {
	URI      string `koanf:"uri"`
	Database string `koanf:"database"`
}

type RedisConfig struct {
	Address string `koanf:"address" validate:"required"`
}

type KafkaConfig struct {
	Brokers []string `koanf:"brokers"`
	GroupID string   `koanf:"group_id"`
	Topic   string   `koanf:"topic"`
}

func LoadConfig() (*Config, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")

	// Determine prefix: default to ENGAGE_, fallback to BOILERPLATE_
	prefix := "ENGAGE_"
	if os.Getenv("BOILERPLATE_PRIMARY.ENV") != "" {
		prefix = "BOILERPLATE_"
	}

	err := k.Load(env.Provider(prefix, ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, prefix))
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load initial env variables")
		return nil, fmt.Errorf("load env: %w", err)
	}

	mainConfig := &Config{}

	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not unmarshal main config")
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Supply defaults if optional configs are empty
	if mainConfig.Server.Port == "" {
		mainConfig.Server.Port = "8080"
		mainConfig.Server.ReadTimeout = 30
		mainConfig.Server.WriteTimeout = 30
		mainConfig.Server.IdleTimeout = 60
		mainConfig.Server.CORSAllowedOrigins = []string{"*"}
	}
	if mainConfig.Primary.Env == "" {
		mainConfig.Primary.Env = "local"
	}
	if mainConfig.Database.Host == "" {
		mainConfig.Database.Host = "localhost"
		mainConfig.Database.Port = 5432
		mainConfig.Database.User = "postgres"
		mainConfig.Database.Password = "postgres"
		mainConfig.Database.Name = "engage_engine"
		mainConfig.Database.SSLMode = "disable"
		mainConfig.Database.MaxOpenConns = 25
		mainConfig.Database.MaxIdleConns = 25
		mainConfig.Database.ConnMaxLifetime = 300
		mainConfig.Database.ConnMaxIdleTime = 300
	}
	if mainConfig.Redis.Address == "" {
		mainConfig.Redis.Address = "localhost:6379"
	}

	validate := validator.New()

	err = validate.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("config validation failed")
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Set default observability config if not provided
	if mainConfig.Observability == nil {
		mainConfig.Observability = DefaultObservabilityConfig()
	}

	// Override service name and environment from primary config
	mainConfig.Observability.ServiceName = "engage-engine"
	mainConfig.Observability.Environment = mainConfig.Primary.Env

	// Validate observability config
	if err := mainConfig.Observability.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid observability config")
		return nil, fmt.Errorf("invalid observability config: %w", err)
	}

	return mainConfig, nil
}
