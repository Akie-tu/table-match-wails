package main

import (
	"fmt"
	"os"

	"tablematch/backend"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: importtest <明细文件>")
		return
	}
	res, err := backend.ImportInvoiceDetail(os.Args[1])
	if err != nil {
		fmt.Println("❌", err)
		return
	}
	if res == nil {
		fmt.Println("无数据")
		return
	}
	fmt.Printf("导入: %d 条 | 未识别列: %v\n", res.Imported, res.Missing)
	for i, r := range res.Rows {
		if i >= 5 {
			fmt.Println("...")
			break
		}
		fmt.Printf("  %d. 名称=%s 税号=%s 金额=%s 数量=%s 自然人=%s\n",
			i+1, r.Buyer, r.TaxID, r.Amount, r.Qty, r.IsNatural)
	}
}