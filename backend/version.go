package backend

// ════════════════════════════════════════════════════════════════
//
//	版本与更新源（★ 发版时只改这里 ✗ 别散落各处）
//
// ════════════════════════════════════════════════════════════════
const (
	// Version 当前版本（与 CHANGELOG / git tag 保持一致）
	Version = "v1.0.6"

	// Repo GitHub 仓库
	Repo = "Akie-tu/table-match-wails"

	// AssetName Release 资产标准名（★ 与 workflow 上传的文件名一致）
	AssetName = "table-match-wails.exe"

	// DefaultMirror ★ GitHub 加速站前缀（自建）
	//   温哥华节点起来后：改 update_config.json 的 mirror 即可 ✗ 不用改代码
	//   留空字符串 = 走 GitHub 直连
	DefaultMirror = "https://gh.chibarin.top/"
)
