package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// 固定内容配置文件名
const FixedConfigFile = "fixed_config.json"

// 用户配置目录(创建): Windows用APPDATA, 其他用~/.config
func userConfigDir() string {
	base := ""
	if app := os.Getenv("APPDATA"); app != "" {
		base = app
	} else if home, err := os.UserHomeDir(); err == nil {
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "tablematch")
	os.MkdirAll(dir, 0o700)
	return dir
}

// 配置目录: 优先程序(exe)同目录, 无法写入时回退用户配置目录
func appConfigDir() string {
	// 尝试程序同目录
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		// 测试可写
		testFile := filepath.Join(dir, ".write_test")
		if err := os.WriteFile(testFile, []byte("x"), 0o600); err == nil {
			os.Remove(testFile)
			return dir
		}
	}
	// 当前目录
	if dir, err := os.Getwd(); err == nil {
		testFile := filepath.Join(dir, ".write_test")
		if err := os.WriteFile(testFile, []byte("x"), 0o600); err == nil {
			os.Remove(testFile)
			return dir
		}
	}
	// 回退用户配置目录
	return userConfigDir()
}

// 固定内容配置路径: 程序当前目录(可写时), 否则用户配置目录
func fixedConfigPath() string {
	return filepath.Join(appConfigDir(), FixedConfigFile)
}

// 保存固定内容到本地配置文件
func SaveFixedConfig(cfg *FixedContent) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := fixedConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// 从本地配置文件读取固定内容; 无配置文件时返回空值(不内置业务默认值)
func LoadFixedConfig() *FixedContent {
	cfg := &FixedContent{} // 默认全空: 脚本本身不携带滤芯/编码等业务数据
	data, err := os.ReadFile(fixedConfigPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

// 税号清洗: 仅保留数字和英文字母(去掉 - 空格 等符号)
func CleanTaxID(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			out = append(out, r)
		}
	}
	return string(out)
}

// 自动查找本地发票模板: 在程序目录/当前目录找文件名含"批量开票-导入开票模板"的xlsx
func FindInvoiceTemplate() string {
	dirs := []string{}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				continue
			}
			if strings.Contains(name, "批量开票-导入开票模板") && strings.HasSuffix(strings.ToLower(name), ".xlsx") {
				return filepath.Join(dir, name)
			}
		}
	}
	return ""
}