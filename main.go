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
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	*app.App
	cfg config.Config
	ctx context.Context
}

func newApp(cfg config.Config) *App {
	client := miniflux.NewClient(cfg.ServerUrl, cfg.ApiKey, cfg.AllowInvalidCerts)
	return &App{
		App: app.New(client, cfg.CacheExpiryDays),
		cfg: cfg,
	}
}

func (a *App) Show() {
	wailsruntime.WindowShow(a.ctx)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	dir := a.cfg.CacheDir
	if dir == "" {
		var err error
		dir, err = cache.DefaultDir()
		if err != nil {
			fmt.Printf("Warning: could not determine cache dir: %v\n", err)
			return
		}
	}
	if err := a.App.Open(dir); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}
}

func (a *App) shutdown(_ context.Context) {
	a.App.Close()
}

func (a *App) GetConfig() config.Config {
	return a.cfg
}

func (a *App) SaveConfig(cfg config.Config) error {
	path, err := config.GetConfigFilepath()
	if err != nil {
		return err
	}
	if err := config.Save(cfg, path); err != nil {
		return err
	}
	a.cfg = cfg
	app.ApplyConfig(a.App, cfg.CacheExpiryDays)
	return nil
}

func (a *App) OpenURL(url string) {
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

	a := newApp(cfg)

	err = wails.Run(&options.App{
		Title:  "anus",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: frontend.FS,
		},
		BackgroundColour:         &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		StartHidden:              true,
		EnableDefaultContextMenu: true,
		OnStartup:                a.startup,
		OnShutdown:               a.shutdown,
		Bind:                     []interface{}{a},
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
