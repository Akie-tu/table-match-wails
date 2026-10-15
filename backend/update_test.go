package backend

import (
	"fmt"
	"strings"
	"testing"
)

// 回归测试：版本比较 + updater.bat 生成（★ 纯逻辑，不依赖网络）

func TestUpdateLogic(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v1.0.6", "v1.0.5", true},
		{"v1.0.5", "v1.0.6", false},
		{"v1.0.10", "v1.0.9", true},
		{"v1.0.6", "v1.0.6", false},
		{"v2.0.0", "v1.9.9", true},
	}
	for _, c := range cases {
		got := versionGT(c.a, c.b)
		mark := "OK"
		if got != c.want {
			mark = "FAIL"
		}
		fmt.Printf("[%s] versionGT(%s, %s) = %v (want %v)\n", mark, c.a, c.b, got, c.want)
	}

	bat := makeUpdaterBat(`C:\app\new_table-match-wails.exe`, `C:\app\table-match-wails.exe`, `C:\app\table-match-wails.exe`, `C:\app`)
	fmt.Println("---- updater.bat ----")
	fmt.Println(bat)
	bad := 0
	for i, r := range bat {
		if r > 127 {
			bad++
			fmt.Printf("NON-ASCII @%d: %q\n", i, r)
		}
	}
	fmt.Printf("非ASCII字符数 = %d (必须 0)\n", bad)
	for _, k := range []string{"tasklist", "move /y", `start ""`, "timeout /t 1", "goto loop"} {
		fmt.Printf("含 %-14s: %v\n", k, strings.Contains(bat, k))
	}
}
