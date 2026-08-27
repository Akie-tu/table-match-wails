package main

import (
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"tablematch/backend"
)

//go:embed all:frontend/dist
var assets embed.FS

// 动态API处理器: 拦截 /api/* 路由, 其余交给静态资源
type apiHandler struct {
	assets http.Handler
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/api/theme" && r.Method == http.MethodGet:
		// 获取外观设置
		cfg := backend.LoadTheme()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	case path == "/api/theme" && r.Method == http.MethodPost:
		// 保存外观设置
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "读取请求失败", http.StatusBadRequest)
			return
		}
		var cfg backend.ThemeWithImage
		if err := json.Unmarshal(body, &cfg); err != nil {
			http.Error(w, "参数解析失败", http.StatusBadRequest)
			return
		}
		if err := backend.SaveTheme(&cfg.ThemeSettings); err != nil {
			http.Error(w, "保存失败", http.StatusInternalServerError)
			return
		}
		// 保存背景图片路径(若有)
		if cfg.ImagePath != "" {
			backend.SaveThemeImage(cfg.ImagePath)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	case path == "/api/theme/css" && r.Method == http.MethodGet:
		// 返回CSS变量文本
		cfg := backend.LoadTheme()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(backend.ThemeToCSS(cfg)))
	case path == "/api/theme/imagepath" && r.Method == http.MethodGet:
		// 返回背景图片本地路径
		p := backend.LoadThemeImage()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": p})
	case path == "/api/bg":
		// 提供背景图片(读本地文件, 避免file://被WebView2拦截)
		p := backend.LoadThemeImage()
		if p == "" {
			http.NotFound(w, r)
			return
		}
		data, err := os.ReadFile(p)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ct := "image/jpeg"
		if strings.HasSuffix(strings.ToLower(p), ".png") {
			ct = "image/png"
		} else if strings.HasSuffix(strings.ToLower(p), ".gif") {
			ct = "image/gif"
		} else if strings.HasSuffix(strings.ToLower(p), ".webp") {
			ct = "image/webp"
		}
		w.Header().Set("Content-Type", ct)
		w.Write(data)
	default:
		// 静态资源
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}
		h.assets.ServeHTTP(w, r)
	}
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "电商工具",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
			Handler: &apiHandler{
				assets: http.FileServer(http.FS(assets)),
			},
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}