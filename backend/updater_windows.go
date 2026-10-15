//go:build windows

package backend

import (
	"os/exec"
	"syscall"
)

// launchUpdater 启动 updater.bat（★ 无窗口：HideWindow + CREATE_NO_WINDOW）
func launchUpdater(batPath, dir string) error {
	cmd := exec.Command("cmd", "/c", batPath)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd.Start()
}
