package main

import (
	"context"
	"fmt"

	"tablematch/backend"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// 表格核对: 前端调用入口
// fillMap: [["源列","目标列"], ...]
func (a *App) RunMatch(srcPath, tgtPath, srcKey, tgtKey string,
	fillMap [][2]string, skuCol, remarkCol string, skipExisting bool) (*backend.MatchResult, error) {
	return backend.RunMatch(srcPath, tgtPath, srcKey, tgtKey, fillMap, skuCol, remarkCol, skipExisting)
}

// 读Excel全部行(供前端预览/展示)
func (a *App) ReadSheet(path, sheet string) ([][]string, error) {
	return backend.ReadAllRows(path, sheet)
}

// 列出sheet名
func (a *App) ListSheets(path string) ([]string, error) {
	return backend.ListSheets(path)
}