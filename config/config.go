package config

import (
	_ "embed"
	"log"
	"os"

	"github.com/joho/godotenv"
)

//go:embed .env
var envContent string

type Config struct {
	NotionDBID   string
	NotionSecret string
	OpenAIAPIKey string
}

var AppConfig Config

func Load() {
	envMap, err := godotenv.Unmarshal(envContent)
	if err != nil {
		log.Println("Error loading embedded .env:", err)
		return
	}

	for k, v := range envMap {
		if err := os.Setenv(k, v); err != nil {
			log.Printf("Could not set %s: %v", k, err)
		}
	}

	AppConfig = Config{
		NotionDBID:   os.Getenv("NOTION_DB_ID"),
		NotionSecret: os.Getenv("NOTION_SECRET"),
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
	}
}
