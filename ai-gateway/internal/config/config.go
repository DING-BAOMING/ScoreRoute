package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort    string
	DatabasePath  string
	JwtSecret     string
	AdminPassword string
	LogPath       string
}

var AppConfig *Config

func Load() (*Config, error) {
	viper.SetDefault("server.port", "3000")
	viper.SetDefault("database.path", "./data/gateway.db")
	viper.SetDefault("log.path", "./logs")

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use defaults
	}

	AppConfig = &Config{
		ServerPort:    viper.GetString("server.port"),
		DatabasePath:  viper.GetString("database.path"),
		JwtSecret:     viper.GetString("jwt.secret"),
		AdminPassword: viper.GetString("admin.password"),
		LogPath:       viper.GetString("log.path"),
	}

	if AppConfig.JwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if AppConfig.AdminPassword == "" {
		return nil, fmt.Errorf("ADMIN_PASSWORD environment variable is required")
	}

	return AppConfig, nil
}
