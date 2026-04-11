package config

import (
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
	viper.SetDefault("jwt.secret", "ai-gateway-secret-key-2024")
	viper.SetDefault("admin.password", "dbm52100")
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

	return AppConfig, nil
}
