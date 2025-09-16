package startupservice

import (
	"github.com/protonmail/go-autostart"
	"os"
)

type StartupService struct {
	app *autostart.App
}

func NewStartupService() *StartupService {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "/Applications/Tasklight.app/Contents/MacOS/tasklight-v3"
	}

	app := &autostart.App{
		Name:        "Tasklight",
		DisplayName: "Tasklight",
		Exec:        []string{execPath},
	}

	return &StartupService{app: app}
}

func (s *StartupService) EnableLaunchAtLogin() error {
	return s.app.Enable()
}

func (s *StartupService) DisableLaunchAtLogin() error {
	return s.app.Disable()
}

func (s *StartupService) IsEnabled() bool {
	return s.app.IsEnabled()
}
