package conf

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

type Server struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type Database struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
}

type Config struct {
	Server   Server   `mapstructure:"server"`
	Log      Log      `mapstructure:"log"`
	Database Database `mapstructure:"database"`
}

var AppConfig Config

func Load(path string) error {
	config := viper.New()
	if path != "" {
		config.SetConfigFile(path)
	} else {
		config.SetConfigName("config")
		config.SetConfigType("json")
		config.AddConfigPath("data")
	}

	config.AutomaticEnv()
	config.SetEnvPrefix(APP_NAME)
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(config)

	if err := config.ReadInConfig(); err == nil {
		log.Infof("Using config file: %s", config.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Infof("Config file not found, creating default config")
			if err := os.MkdirAll("data", 0755); err != nil {
				return fmt.Errorf("create data directory: %w", err)
			}
			if err := config.SafeWriteConfigAs("data/config.json"); err != nil {
				return fmt.Errorf("create default config: %w", err)
			}
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	var loaded Config
	if err := config.Unmarshal(&loaded); err != nil {
		return fmt.Errorf("unable to decode config into struct: %w", err)
	}
	AppConfig = loaded
	return nil
}

func setDefaults(viper *viper.Viper) {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.path", "data/data.db")
	viper.SetDefault("log.level", "info")
}
