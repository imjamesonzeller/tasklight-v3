package tray

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func Setup(app *application.App, windowService WindowServiceInterface, trayIcon []byte) {
	tray := app.NewSystemTray()
	menu := application.NewMenu()

	// ------ ITEMS ------
	menu.Add("Show").OnClick(func(_ *application.Context) {
		windowService.Show("main")
	})
	menu.Add("Settings").OnClick(func(_ *application.Context) {
		windowService.Show("settings")
	})
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		app.Quit()
	})

	tray.SetMenu(menu)
	tray.SetIcon(trayIcon)
}

type WindowServiceInterface interface {
	Show(name string)
}
