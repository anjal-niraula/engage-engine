package config

import (
	"fmt"
	"time"
)

type ObservabilityConfig struct {
	ServiceName string            `koanf:"service_name" validate:"required"`
	Environment string            `koanf:"environment"  validate:"oneof=development staging production local"`
	Logging     LoggingConfig     `koanf:"logging"`
	NewRelic    NewRelicConfig    `koanf:"new_relic"`
	HealthCheck HealthCheckConfig `koanf:"health_checks"`
}

type LoggingConfig struct {
	Level              string        `koanf:"level"                 validate:"oneof=debug info warn error"`
	Format             string        `koanf:"format"                validate:"oneof=json text console"`
	SlowQueryThreshold time.Duration `koanf:"slow_query_threshold" validate:"gt=0"`
}

type NewRelicConfig struct {
	LicenseKey                string `koanf:"license_key"`
	AppLogForwardingEnabled   bool   `koanf:"app_log_forwarding_enabled"`
	DistributedTracingEnabled bool   `koanf:"distributed_tracing_enabled"`
	DebugLogging              bool   `koanf:"debug_logging"`
}

type HealthCheckConfig struct {
	Enabled  bool          `koanf:"enabled"`
	Interval time.Duration `koanf:"interval" validate:"gt=0"`
	Timeout  time.Duration `koanf:"timeout"  validate:"gt=0"`
	Checks   []string      `koanf:"checks"`
}

func DefaultObservabilityConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		ServiceName: "engage-engine",
		Environment: "local",
		Logging: LoggingConfig{
			Level:              "debug",
			Format:             "console",
			SlowQueryThreshold: 100 * time.Millisecond,
		},
		NewRelic: NewRelicConfig{
			LicenseKey:                "",
			AppLogForwardingEnabled:   false,
			DistributedTracingEnabled: false,
			DebugLogging:              false,
		},
		HealthCheck: HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Checks:   []string{"database", "redis"},
		},
	}
}

func (o ObservabilityConfig) Validate() error {
	if o.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if o.Environment != "development" && o.Environment != "staging" && o.Environment != "production" && o.Environment != "local" {
		return fmt.Errorf("environment must be one of: development, staging, production, local")
	}
	if o.Logging.Level != "debug" && o.Logging.Level != "info" && o.Logging.Level != "warn" && o.Logging.Level != "error" {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}
	if o.Logging.Format != "json" && o.Logging.Format != "text" && o.Logging.Format != "console" {
		return fmt.Errorf("logging.format must be one of: json, text, console")
	}
	if o.Logging.SlowQueryThreshold <= 0 {
		return fmt.Errorf("logging.slow_query_threshold must be greater than 0")
	}
	if o.HealthCheck.Enabled {
		if o.HealthCheck.Interval <= 0 {
			return fmt.Errorf("health_check.interval must be greater than 0")
		}
		if o.HealthCheck.Timeout <= 0 {
			return fmt.Errorf("health_check.timeout must be greater than 0")
		}
	}
	return nil
}

func (c *ObservabilityConfig) GetLogLevel() string {
	switch c.Environment {
	case "production":
		if c.Logging.Level == "" {
			return "info"
		}
	case "development", "local":
		if c.Logging.Level == "" {
			return "debug"
		}
	}
	return c.Logging.Level
}

func (c *ObservabilityConfig) IsProduction() bool {
	return c.Environment == "production"
}
