package configs

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotName  string
	BotToken string
	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env is not found")
	}

	cfg := &Config{
		BotName:  os.Getenv("BOT_NAME"),
		BotToken: os.Getenv("BOT_TOKEN"),
		Driver:   os.Getenv("DB_DRIVER"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}

	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" || cfg.DBName == "" {
		return nil, errors.New("some of DB_* values is empty")
	}

	if cfg.Driver == "" {
		cfg.Driver = "postgres"
	}

	return cfg, nil
}
