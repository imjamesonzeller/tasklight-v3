package config

import (
	"github.com/imjamesonzeller/tasklight-v3/settingsservice"
)

var AppConfig *settingsservice.ApplicationSettings

func Init(settings *settingsservice.ApplicationSettings) {
	AppConfig = settings
}
