package main

import (
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"rimchecker/frontend"
	"rimchecker/internal/desktop"
)

func main() {
	if _, err := fs.Stat(frontend.Dist, "dist/index.html"); err != nil {
		log.Fatal("缺少前端构建产物，请先运行 Wails build")
	}
	var app *application.App
	service, err := desktop.New(desktop.Options{SelectFile: func() (string, error) {
		dialog := app.Dialog.OpenFile().SetTitle("选择 RimWorld 日志").AddFilter("日志文件", "*.log;*.txt").AddFilter("全部文件", "*.*")
		if home, err := os.UserHomeDir(); err == nil {
			directory := filepath.Join(home, "AppData", "LocalLow", "Ludeon Studios", "RimWorld by Ludeon Studios")
			// Leave the native default unchanged when the game directory is unavailable.
			if info, err := os.Stat(directory); err == nil && info.IsDir() {
				dialog.SetDirectory(directory)
			}
		}
		return dialog.PromptForSingleSelection()
	}})
	if err != nil {
		log.Fatal(err)
	}
	assets, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		log.Fatal(err)
	}
	app = application.New(application.Options{Name: "RimChecker", Description: "本地 RimWorld 日志诊断", Services: []application.Service{application.NewService(service)}, Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)}, Logger: application.DefaultLogger(slog.LevelError)})
	app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "RimChecker", Width: 1240, Height: 860, MinWidth: 820, MinHeight: 620, BackgroundColour: application.NewRGB(15, 18, 24), URL: "/"})
	defer service.CancelDiagnosis()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
