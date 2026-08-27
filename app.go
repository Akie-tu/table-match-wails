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

// 导入发票明细: 读Excel/CSV自动匹配列(抬头→名称/税号→识别号/金额→金额/数量→数量/类型=个人→自然人)
func (a *App) ImportInvoiceDetail(path string) (*backend.ImportResult, error) {
	return backend.ImportInvoiceDetail(path)
}

// 选择文件夹(图片转换源/输出)
func (a *App) SelectDir() (string, error) {
	opts := runtime.OpenDialogOptions{Title: "选择文件夹"}
	return runtime.OpenDirectoryDialog(a.ctx, opts)
}

// 批量图片转JPG
func (a *App) RunImgConvert(srcRoot, outRoot string, quality int) (*backend.ImgConvertResult, error) {
	return backend.RunImgConvert(srcRoot, outRoot, quality)
}

// 邮箱: 保存配置
func (a *App) SaveEmailConfig(cfg *backend.EmailConfig) error {
	return backend.SaveEmailConfig(cfg)
}

// 邮箱: 读取配置
func (a *App) LoadEmailConfig() (*backend.EmailConfig, error) {
	return backend.LoadEmailConfig()
}

// 邮箱: 发送(带附件)
func (a *App) SendEmail(cfg *backend.EmailConfig, to, subject, body string, attachments []string) (*backend.EmailResult, error) {
	return backend.SendEmail(cfg, to, subject, body, attachments)
}

// 邮箱: 预设(单对象返回)
func (a *App) EmailPreset(provider string) backend.PresetResult {
	return backend.EmailPreset(provider)
}

// 固定内容: 保存到本地配置文件
func (a *App) SaveFixedConfig(cfg *backend.FixedContent) error {
	return backend.SaveFixedConfig(cfg)
}

// 固定内容: 从本地配置文件读取(无配置返回空)
func (a *App) LoadFixedConfig() (*backend.FixedContent, error) {
	return backend.LoadFixedConfig(), nil
}

// 税号清洗: 仅保留数字和英文
func (a *App) CleanTaxID(s string) (string, error) {
	return backend.CleanTaxID(s), nil
}

// 自动查找本地发票模板(文件名含"批量开票-导入开票模板")
func (a *App) FindInvoiceTemplate() (string, error) {
	return backend.FindInvoiceTemplate(), nil
}