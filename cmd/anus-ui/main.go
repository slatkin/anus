package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/pkg/browser"
	"github.com/slatkin/anus/frontend"
	"github.com/slatkin/anus/internal/cache"
	"github.com/slatkin/anus/pkg/app"
	"github.com/slatkin/anus/pkg/config"
	"github.com/slatkin/anus/pkg/miniflux"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

// uiApp wraps app.App for Wails binding, adding OpenURL and lifecycle hooks.
type uiApp struct {
	*app.App
	cfg config.Config
}

func newUIApp(cfg config.Config) *uiApp {
	client := miniflux.NewClient(cfg.ServerUrl, cfg.ApiKey, cfg.AllowInvalidCerts)
	return &uiApp{
		App: app.New(client, cfg.CacheExpiryDays, cfg.RememberReadPosition),
		cfg: cfg,
	}
}

func (u *uiApp) startup(_ context.Context) {
	dir := u.cfg.CacheDir
	if dir == "" {
		var err error
		dir, err = cache.DefaultDir()
		if err != nil {
			fmt.Printf("Warning: could not determine cache dir: %v\n", err)
			return
		}
	}
	if err := u.App.Open(dir); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}
}

func (u *uiApp) shutdown(_ context.Context) {
	u.App.Close()
}

func (u *uiApp) OpenURL(url string) {
	browser.OpenURL(url) //nolint
}

func main() {
	initFlag := flag.Bool("init", false, "Initialize default configuration file")
	flag.Parse()

	if *initFlag {
		path, err := config.Init()
		if err != nil {
			fmt.Printf("Error initializing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote default configuration file to %s\n", path)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	ui := newUIApp(cfg)

	err = wails.Run(&options.App{
		Title:  "anus",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: frontend.FS,
		},
		BackgroundColour:         &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		EnableDefaultContextMenu: true,
		OnStartup:                ui.startup,
		OnShutdown:               ui.shutdown,
		Bind:                     []interface{}{ui},
		Linux: &linux.Options{
			ProgramName:         "anus",
			WindowIsTranslucent: true,
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
