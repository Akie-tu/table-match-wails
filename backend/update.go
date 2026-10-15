package backend

// ════════════════════════════════════════════════════════════════
//  内置更新模块 —— 检测 GitHub Release → 走【加速节点】下载 → 替换重启
//  ★ 加速前缀可配（update_config.json 的 mirror）✗ 不写死在代码里
//  ★ 移植自 Python 版 update.py，保留：资产名归口 / HEAD 预检 / 断点续传 /
//    大小校验 / 纯 ASCII updater.bat 延迟替换
// ════════════════════════════════════════════════════════════════

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// UpdateConfig 更新源配置（update_config.json，与 exe 同目录）
type UpdateConfig struct {
	Mirror string `json:"mirror"` // 加速前缀，如 https://gh.chibarin.top/ ；空=直连
	Repo   string `json:"repo"`
	Asset  string `json:"asset"`
}

// UpdateInfo 检查结果
type UpdateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	HasUpdate bool   `json:"has_update"`
	Notes     string `json:"notes"`
	URL       string `json:"url"`
	Mirror    string `json:"mirror"`
	Error     string `json:"error"`
}

// UpdateResult 下载/替换结果
type UpdateResult struct {
	OK    bool   `json:"ok"`
	Size  int64  `json:"size"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

var updateClient = &http.Client{Timeout: 120 * time.Second}

func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func updateCfgPath() string { return filepath.Join(exeDir(), "update_config.json") }

// LoadUpdateConfig 读配置（不存在 → 用默认：自建加速站）
func LoadUpdateConfig() UpdateConfig {
	cfg := UpdateConfig{Mirror: DefaultMirror, Repo: Repo, Asset: AssetName}
	if b, err := os.ReadFile(updateCfgPath()); err == nil {
		var c UpdateConfig
		if json.Unmarshal(b, &c) == nil {
			if strings.TrimSpace(c.Repo) != "" {
				cfg.Repo = strings.TrimSpace(c.Repo)
			}
			if strings.TrimSpace(c.Asset) != "" {
				cfg.Asset = strings.TrimSpace(c.Asset)
			}
			cfg.Mirror = strings.TrimSpace(c.Mirror) // ★ 显式空串 = 直连（✗ 别当"没配"）
		}
	}
	return cfg
}

// SaveUpdateConfig 写回配置
func SaveUpdateConfig(c UpdateConfig) error {
	if strings.TrimSpace(c.Repo) == "" {
		c.Repo = Repo
	}
	if strings.TrimSpace(c.Asset) == "" {
		c.Asset = AssetName
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(updateCfgPath(), b, 0644)
}

// mirrorURL 把加速前缀拼到原始 GitHub URL 前面
//
//	https://gh.chibarin.top/ + https://github.com/x/y/releases/download/v1/f.exe
func mirrorURL(prefix, raw string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return raw
	}
	return strings.TrimRight(prefix, "/") + "/" + raw
}

// candidateURLs 候选通道：加速站优先 → 直连 → 公共镜像兜底
func candidateURLs(cfg UpdateConfig, raw string) []string {
	out := []string{}
	if u := mirrorURL(cfg.Mirror, raw); u != raw {
		out = append(out, u)
	}
	out = append(out, raw)
	// 备用自建源（主用挂掉时回落）+ 公共镜像兜底
	for _, m := range []string{"https://gh.chibarin.top/", "https://ghfast.top/", "https://gh-proxy.com/"} {
		if strings.TrimSpace(cfg.Mirror) == strings.TrimSpace(m) {
			continue
		}
		out = append(out, m+raw)
	}
	return out
}

var verRe = regexp.MustCompile(`v?(\d+(?:\.\d+)*)`)

func parseVer(tag string) []int {
	m := verRe.FindStringSubmatch(tag)
	if m == nil {
		return []int{0}
	}
	parts := strings.Split(m[1], ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

// versionGT a > b ?
func versionGT(a, b string) bool {
	pa, pb := parseVer(a), parseVer(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type ghRelease struct {
	Tag    string    `json:"tag_name"`
	Body   string    `json:"body"`
	Assets []ghAsset `json:"assets"`
}

func getJSON(url string, v any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "table-match-wails-updater")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := updateClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// latestManifestURL 版本清单地址
//
//	★ 关键设计：走 releases/latest/download/latest.json（下载路径）
//	⇒ ① 加速站能代理（✗ api.github.com 会被自建加速站 403）
//	  ② 免 API 限流；③ 永远指向"最新" release，不用先知道 tag
func latestManifestURL(repo string) string {
	return fmt.Sprintf("https://github.com/%s/releases/latest/download/latest.json", repo)
}

// rawManifestURL ★ 主用：raw 路径（自建加速站放行 raw ⇒ 国内可直取）
//
//	CI 发版时会把 latest.json 提交回仓库根目录
func rawManifestURL(repo string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/main/latest.json", repo)
}

// fallbackAssetURL 兜底下载地址（清单里的 url 为空时用）
func fallbackAssetURL(repo, asset string) string {
	return fmt.Sprintf("https://github.com/%s/releases/latest/download/%s", repo, asset)
}

// manifest 版本清单（CI 发版时随 release 一起上传）
type manifest struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	URL     string `json:"url"`
}

// CheckUpdate 检查是否有新版
//
//	① 首选 latest.json（加速站可代理，免 API）
//	② 兜底 GitHub API（按资产名归口）
func CheckUpdate() *UpdateInfo {
	cfg := LoadUpdateConfig()
	info := &UpdateInfo{Current: Version, Mirror: cfg.Mirror}

	// ── ① latest.json ──
	//   ★ 顺序有讲究：自建加速站【只放行 download / raw 两类路径】，
	//     /releases/latest/... 会被它 502 ⇒ 主用 raw，latest/download 仅作直连兜底
	manifestURLs := []string{
		rawManifestURL(cfg.Repo),    // https://raw.githubusercontent.com/<repo>/main/latest.json
		latestManifestURL(cfg.Repo), // https://github.com/<repo>/releases/latest/download/latest.json
	}
	for _, murl := range manifestURLs {
		for _, u := range candidateURLs(cfg, murl) {
			var m manifest
			if err := getJSON(u, &m); err != nil || strings.TrimSpace(m.Version) == "" {
				continue
			}
			info.Latest = strings.TrimSpace(m.Version)
			info.Notes = strings.TrimSpace(m.Notes)
			info.URL = strings.TrimSpace(m.URL)
			if info.URL == "" {
				info.URL = fallbackAssetURL(cfg.Repo, cfg.Asset)
			}
			info.HasUpdate = versionGT(info.Latest, Version)
			return info
		}
	}

	// ── ② GitHub API 兜底 ──
	api := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=15", cfg.Repo)
	var rels []ghRelease
	var lastErr error
	for _, u := range candidateURLs(cfg, api) {
		if err := getJSON(u, &rels); err == nil {
			lastErr = nil
			break
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		info.Error = "查询更新失败（网络或加速站不可用）：" + lastErr.Error()
		return info
	}
	for _, r := range rels {
		for _, a := range r.Assets {
			if strings.EqualFold(a.Name, cfg.Asset) {
				info.Latest = r.Tag
				info.Notes = strings.TrimSpace(r.Body)
				info.URL = a.URL
				info.HasUpdate = versionGT(r.Tag, Version)
				return info
			}
		}
	}
	info.Error = "未找到含 " + cfg.Asset + " 的发布"
	return info
}

func headOK(url string) bool {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return true
	}
	req.Header.Set("User-Agent", "table-match-wails-updater")
	resp, err := updateClient.Do(req)
	if err != nil {
		return true // 网络异常别误判成"资产没上传"
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200 || resp.StatusCode == 206 || resp.StatusCode == 302
}

func downloadOne(url, dest string, progress func(int64, int64, string)) (int64, error) {
	var from int64
	if fi, err := os.Stat(dest); err == nil {
		from = fi.Size()
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "table-match-wails-updater")
	if from > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", from))
	}
	resp, err := updateClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	total := resp.ContentLength
	if resp.StatusCode == 206 {
		total += from
	} else {
		from = 0 // 服务器不支持 Range ⇒ 重头下
	}
	flag := os.O_CREATE | os.O_WRONLY
	if from > 0 {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	f, err := os.OpenFile(dest, flag, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	got := from
	buf := make([]byte, 128*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return got, werr
			}
			got += int64(n)
			if progress != nil {
				progress(got, total, "下载中")
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return got, rerr
		}
	}
	if total > 0 && got != total {
		return got, fmt.Errorf("大小不符（%d/%d）", got, total)
	}
	return got, nil
}

// downloadAsset 多通道下载（加速优先），带 HEAD 预检 + 断点续传 + 大小校验
func downloadAsset(cfg UpdateConfig, url, dest string, progress func(int64, int64, string)) (int64, error) {
	if url == "" {
		return 0, errors.New("下载地址为空")
	}
	if !headOK(url) {
		return 0, errors.New("新版本正在构建中，请几分钟后再试（资产尚未上传）")
	}
	var lastErr error
	for _, u := range candidateURLs(cfg, url) {
		n, err := downloadOne(u, dest, progress)
		if err == nil {
			return n, nil
		}
		lastErr = err
		os.Remove(dest) // 换通道重下，避免半成品错拼
	}
	return 0, lastErr
}

// winBase 取 Windows 路径的文件名（同时认 \ 与 /）
func winBase(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// winJoin 拼 Windows 路径（避免宿主为 Linux 时拼出 C:/app/xx）
func winJoin(dir, name string) string {
	dir = strings.TrimRight(dir, "\\/")
	return dir + "\\" + name
}

// makeUpdaterBat 生成 updater.bat
//
//	★ 必须纯 ASCII：Windows CMD 默认 GBK，写中文会乱码/执行失败 ⚠️
//	逻辑：等主进程退出(≤120s) → 若当前 exe 名≠标准名则删掉它 → move 覆盖 → 重启
func makeUpdaterBat(newExe, runningExe, targetExe, dir string) string {
	q := func(s string) string { return `"` + s + `"` }
	// ⚠️ tasklist 的 IMAGENAME 只认【纯文件名】(给全路径匹配不上)
	//    且路径分隔符要按 Windows 来 ⇒ 别直接用 filepath.Base/Join(宿主是 Linux 时会出错)
	runName := winBase(runningExe)
	resFile := winJoin(dir, "updater_result.txt")

	var b strings.Builder
	b.WriteString("@echo off\r\nsetlocal\r\n")
	b.WriteString("rem updater for table-match-wails (auto-generated, ASCII only)\r\n")
	b.WriteString("cd /d " + q(dir) + "\r\n")
	b.WriteString("set /a tries=0\r\n:loop\r\n")
	b.WriteString(`tasklist /FI "IMAGENAME eq ` + runName + `" 2>nul | find /I "` + runName + `" >nul` + "\r\n")
	b.WriteString("if %errorlevel%==0 (\r\n  set /a tries+=1\r\n  if %tries% GEQ 120 goto force\r\n")
	b.WriteString("  timeout /t 1 /nobreak >nul\r\n  goto loop\r\n)\r\n")
	b.WriteString(":force\r\n")
	b.WriteString("if /I not " + q(runningExe) + "==" + q(targetExe) + " del " + q(runningExe) + " >nul 2>&1\r\n")
	b.WriteString("move /y " + q(newExe) + " " + q(targetExe) + " >nul 2>&1\r\n")
	b.WriteString("if %errorlevel%==0 goto restart\r\n")
	b.WriteString("echo FAILED_TO_REPLACE > " + q(resFile) + "\r\nexit /b 1\r\n")
	b.WriteString(":restart\r\necho OK > " + q(resFile) + "\r\n")
	b.WriteString("start \"\" " + q(targetExe) + "\r\nexit /b 0\r\n")
	return b.String()
}

// DownloadAndUpdate 下载 → 生成 updater.bat → 启动替换（调用方随后退出程序）
func DownloadAndUpdate(url string, progress func(int64, int64, string)) *UpdateResult {
	cfg := LoadUpdateConfig()
	dir := exeDir()

	running, err := os.Executable()
	if err != nil {
		running = filepath.Join(dir, cfg.Asset)
	}
	target := filepath.Join(dir, cfg.Asset) // ★ 标准名（用户改过 exe 名也归位）
	newExe := filepath.Join(dir, "new_"+cfg.Asset)

	if progress != nil {
		progress(0, 0, "准备下载…")
	}
	n, err := downloadAsset(cfg, url, newExe, progress)
	if err != nil {
		return &UpdateResult{Error: err.Error()}
	}
	batPath := filepath.Join(dir, "updater.bat")
	if err := os.WriteFile(batPath, []byte(makeUpdaterBat(newExe, running, target, dir)), 0644); err != nil {
		return &UpdateResult{Error: "写入 updater.bat 失败：" + err.Error()}
	}
	if progress != nil {
		progress(n, n, "启动替换…")
	}
	if err := launchUpdater(batPath, dir); err != nil {
		return &UpdateResult{Error: "启动更新器失败：" + err.Error()}
	}
	return &UpdateResult{OK: true, Size: n, Path: target}
}
