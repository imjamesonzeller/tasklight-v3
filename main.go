package main

import (
	"embed"
	_ "embed"
	"github.com/imjamesonzeller/tasklight-v3/tray"
	"log"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

// TODO: Create app icon for this
//
//go:embed frontend/public/wails.png
var trayIcon []byte

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	var app *application.App
	// Initialize services
	greetService := &GreetService{}
	windowService := NewWindowService()
	hotkeyService := NewHotkeyService(windowService)

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app = application.New(application.Options{
		Name:        "tasklight-v3",
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(greetService),
			application.NewService(windowService),
			application.NewService(hotkeyService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	// Hide app from dock and CMD+Tab
	hideAppFromDock()

	// Inject app to hotkey service
	hotkeyService.SetApp(app)

	// Run Hotkey Service in go-func
	go func() {
		runtime.LockOSThread() // <-- Required by macOS for hotkey
		hotkeyService.StartHotkeyListener()
	}()

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	mainWindow := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:         "Input Window",
		Width:         600,
		Height:        200,
		Frameless:     true,
		DisableResize: true,
		AlwaysOnTop:   true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTransparent,
			TitleBar: application.MacTitleBar{
				AppearsTransparent:   true,
				Hide:                 true,
				HideTitle:            true,
				FullSizeContent:      true,
				UseToolbar:           true,
				HideToolbarSeparator: true,
			},
		},
		BackgroundColour: application.NewRGBA(27, 38, 54, 0),
		URL:              "/",
	})

	// Inject main window to window service
	windowService.RegisterWindow("main", mainWindow)

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.EmitEvent("time", now)
			time.Sleep(time.Second)
		}
	}()

	// Creation of Tray Menu
	// TODO: Make this actually useful with like a settings menu thing
	tray.Setup(app, windowService, trayIcon)

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
