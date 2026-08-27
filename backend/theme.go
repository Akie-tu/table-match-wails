package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// 外观设置(背景等)
type ThemeSettings struct {
	Background  string `json:"background"`  // 背景色(hex, 如 #1b2632)
	CardColor   string `json:"card_color"`  // 卡片色
	TextColor   string `json:"text_color"`  // 文字色
	AccentColor string `json:"accent_color"` // 强调色
	DarkMode    bool   `json:"dark_mode"`   // 深色模式
	Blur        int    `json:"blur"`        // 卡片模糊度(0-20px)
	Opacity     int    `json:"opacity"`     // 卡片透明度(10-90%)
}

// 默认外观(深色)
func DefaultTheme() ThemeSettings {
	return ThemeSettings{
		Background:  "#1b2632",
		CardColor:   "#243140",
		TextColor:   "#e6edf3",
		AccentColor: "#4cc38a",
		DarkMode:    true,
		Blur:        10,
		Opacity:     75,
	}
}

// 外观配置文件路径(与固定内容同目录)
func themeConfigPath() string {
	return filepath.Join(appConfigDir(), "theme_config.json")
}

// 保存外观设置
func SaveTheme(cfg *ThemeSettings) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := themeConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// 读取外观设置(无配置时返回默认)
func LoadTheme() ThemeSettings {
	cfg := DefaultTheme()
	data, err := os.ReadFile(themeConfigPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

// 生成CSS变量文本(前端注入用)
func ThemeToCSS(cfg ThemeSettings) string {
	return "--bg: " + cfg.Background + ";\n" +
		"--card: " + cfg.CardColor + ";\n" +
		"--text: " + cfg.TextColor + ";\n" +
		"--accent: " + cfg.AccentColor + ";"
}

// 外观设置含图片背景
type ThemeWithImage struct {
	ThemeSettings
	BackgroundImage string `json:"background_image"` // 背景图片API地址
	ImagePath       string `json:"image_path"`       // 本地图片路径(下载后)
}

// 保存背景图片路径(单独文件, 供启动恢复)
func SaveThemeImage(path string) error {
	data, _ := json.Marshal(map[string]string{"path": path})
	return os.WriteFile(filepath.Join(appConfigDir(), "theme_image.json"), data, 0o600)
}

// 读取背景图片路径
func LoadThemeImage() string {
	data, err := os.ReadFile(filepath.Join(appConfigDir(), "theme_image.json"))
	if err != nil {
		return ""
	}
	var m map[string]string
	_ = json.Unmarshal(data, &m)
	// 检查文件是否还存在
	if m["path"] != "" {
		if _, err := os.Stat(m["path"]); err == nil {
			return m["path"]
		}
	}
	return ""
}

// 下载图片API背景: 下载到本地目录并返回本地路径
// url: 图片API地址(如 https://www.dmoe.cc/random.php)
func DownloadBackgroundImage(url string) (string, error) {
	if url == "" {
		return "", nil
	}
	// 创建背景缓存目录(与配置同目录)
	dir := filepath.Join(appConfigDir(), "bg")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// 下载
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("下载图片失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("图片接口返回状态: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// 保存为背景图(固定名, 覆盖旧图)
	ext := ".jpg"
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "png") {
		ext = ".png"
	} else if strings.Contains(ct, "gif") {
		ext = ".gif"
	} else if strings.Contains(ct, "webp") {
		ext = ".webp"
	}
	path := filepath.Join(dir, "background"+ext)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}