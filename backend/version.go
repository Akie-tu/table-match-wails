package backend

// ════════════════════════════════════════════════════════════════
//
//	版本与更新源（★ 发版时只改这里 ✗ 别散落各处）
//
// ════════════════════════════════════════════════════════════════
var (
	// Version 当前版本（与 CHANGELOG / git tag 保持一致）
	//   ★ 故意用 var 而非 const：可 -ldflags "-X tablematch/backend.Version=vX" 覆盖，
	//     用来真实演练"旧版本 → 发现新版"的更新链路
	Version = "v1.0.8"

	// Repo GitHub 仓库
	Repo = "Akie-tu/table-match-wails"

	// AssetName Release 资产标准名（★ 与 workflow 上传的文件名一致）
	AssetName = "table-match-wails.exe"

	// DefaultMirror ★ GitHub 加速站前缀（自建 · 温哥华节点）
	//   实测：温哥华 599 KB/s vs 香港 32 KB/s ⇒ 快约 18 倍 ✅
	//   末尾那串是【密径】——加速站只放行 /a7f3k9x2m8qz/ 开头的请求，防白嫖带宽
	//   留空字符串 = 走 GitHub 直连；界面/update_config.json 都能改
	DefaultMirror = "http://104.129.183.180:8088/a7f3k9x2m8qz/"
)
