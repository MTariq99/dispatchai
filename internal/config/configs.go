package config

import (
	"fmt"
	"os"

	"github.com/mtariq99/dispatchai/models"
	"github.com/spf13/viper"
)

var Cfg models.Config

func Load() (*models.Config, error) {
	wd, _ := os.Getwd()
	fmt.Println("Working directory:", wd)

	env := os.Getenv("ENV")

	var configFile string

	switch env {
	case "prod":
		configFile = ".env.prod"

	case "staging":
		configFile = ".env.staging"

	default:
		configFile = ".env.dev"
	}

	viper.SetConfigFile(configFile)
	viper.SetConfigType("env")

	// Allow environment variables to override config-file values.
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf(
			"No config file found (%s), using environment variables only\n",
			configFile,
		)
	} else {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Defaults.
	if Cfg.Server.Port == "" {
		Cfg.Server.Port = "8080"
	}

	if Cfg.Server.AllowedOrigins == "" {
		Cfg.Server.AllowedOrigins = "http://localhost:3000"
	}

	if Cfg.Database.MaxOpenConns == 0 {
		Cfg.Database.MaxOpenConns = 50
	}

	if Cfg.Database.MaxIdleConns == 0 {
		Cfg.Database.MaxIdleConns = 10
	}

	if Cfg.Database.MigrationPath == "" {
		Cfg.Database.MigrationPath = "migrations"
	}

	if Cfg.LLM.MaxContextTokens == 0 {
		Cfg.LLM.MaxContextTokens = 8000
	}

	// Required configuration.
	if Cfg.Database.URL == "" && Cfg.Database.DSN == "" {
		return nil, fmt.Errorf(
			"DATABASE_URL or DATABASE_DSN is required but not set",
		)
	}

	return &Cfg, nil
}
