//go:build !windows

package backend

import "os/exec"

// launchUpdater 非 Windows 平台（本地开发/测试用）
func launchUpdater(batPath, dir string) error {
	cmd := exec.Command("bash", batPath)
	cmd.Dir = dir
	return cmd.Start()
}
