package config

import (
	_ "embed"
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
	"github.com/joho/godotenv"
	"log"
	"os"
)

var AppConfig *settingsservice.ApplicationSettings

func Init(settings *settingsservice.ApplicationSettings) {
	AppConfig = settings
}

//go:embed .env
var embeddedEnv string

func LoadEnv() {
	envMap, err := godotenv.Unmarshal(embeddedEnv)
	if err != nil {
		log.Println("⚠️ Failed to load embedded .env:", err)
		return
	}

	for k, v := range envMap {
		os.Setenv(k, v)
	}
}

func GetEnv(key string) string {
	return os.Getenv(key)
}
