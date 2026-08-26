package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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

// SelectFile: 原生文件选择对话框(前端调用)
func (a *App) SelectFile() (string, error) {
	opts := runtime.OpenDialogOptions{
		Title: "选择文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel/CSV", Pattern: "*.xlsx;*.xlsm;*.csv"},
			{DisplayName: "所有文件", Pattern: "*.*"},
		},
	}
	return runtime.OpenFileDialog(a.ctx, opts)
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

// 开票: 生成xlsx (返回单对象, 避免多返回值JS解构问题)
func (a *App) GenerateInvoice(invoices []*backend.Invoice, fixed backend.FixedContent, templatePath, outPath string) (*backend.InvoiceResult, error) {
	path, errs, err := backend.GenerateInvoiceXlsx(invoices, fixed, templatePath, outPath)
	if err != nil {
		return nil, err
	}
	return &backend.InvoiceResult{Path: path, Errors: errs}, nil
}

// 选择保存路径(开票输出)
func (a *App) SelectSavePath(suggestName string) (string, error) {
	opts := runtime.SaveDialogOptions{
		Title:           "保存开票文件",
		DefaultFilename: suggestName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel", Pattern: "*.xlsx"},
		},
	}
	return runtime.SaveFileDialog(a.ctx, opts)
}