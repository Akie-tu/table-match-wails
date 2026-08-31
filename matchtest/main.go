package main

import (
	"fmt"
	"os"
	"time"

	"tablematch/backend"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("用法: matchtest <源表> <目标表>")
		return
	}
	start := time.Now()
	fillMap := [][2]string{
		{"店铺", "店铺"}, {"下单时间", "下单时间"}, {"商家编码", "商家编码"},
	}
	res, err := backend.RunMatch(os.Args[1], os.Args[2], "运单号", "运单号", fillMap, "", "", false)
	if err != nil {
		fmt.Println("❌ 错误:", err)
		return
	}
	fmt.Printf("耗时: %v\n", time.Since(start))
	fmt.Printf("匹配: %d, 多规格: %d, 未匹配: %d\n", res.Matched, res.Multi, len(res.NotFound))
	fmt.Printf("NotFound is nil? %v\n", res.NotFound == nil)
	if len(res.NotFound) > 0 {
		fmt.Printf("未匹配示例: %v\n", res.NotFound[:min(5, len(res.NotFound))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
