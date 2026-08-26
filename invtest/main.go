package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
	"tablematch/backend"
)

func main() {
	// 测试开票生成
	if len(os.Args) < 2 {
		fmt.Println("用法: invtest <模板路径> [输出路径]")
		return
	}
	tpl := os.Args[1]
	out := "/tmp/go_invoice_test.xlsx"
	if len(os.Args) > 2 {
		out = os.Args[2]
	}

	fixed := backend.DefaultFixed()

	invoices := []*backend.Invoice{
		{
			InvoiceType: "增值税专用发票", TaxIncluded: "否", Buyer: "测试公司A",
			TaxID: "91370306MA3TBJ0T8E", Qty: "1", Amount: "122.00",
			ItemName: "滤芯", TaxCode: "1090130020000000000", Unit: "个", TaxRate: "0.13",
		},
		{
			Buyer: "张三", IsNatural: "是", Qty: "2", Amount: "52.20",
			ItemName: "滤芯", TaxCode: "1090130020000000000", Unit: "个", TaxRate: "0.01",
		},
	}

	path, errs, err := backend.GenerateInvoiceXlsx(invoices, fixed, tpl, out)
	if err != nil {
		fmt.Println("❌ 错误:", err)
		return
	}
	if len(errs) > 0 {
		fmt.Println("❌ 校验失败:")
		for _, e := range errs {
			fmt.Println("  ", e)
		}
		return
	}
	fmt.Println("✅ 生成:", path)

	// 验证
	f, _ := excelize.OpenFile(path)
	defer f.Close()
	// 直接按列号读
	for r := 4; r <= 5; r++ {
		t := cell(f, "1-发票基本信息", 2, r)
		ti := cell(f, "1-发票基本信息", 4, r)
		b := cell(f, "1-发票基本信息", 6, r)
		n := cell(f, "1-发票基本信息", 5, r)
		fmt.Printf("R%d: 类型=%s 含税=%s 名称=%s 自然人=%s\n", r, t, ti, b, n)
	}
	fmt.Println("sheet数:", len(f.GetSheetList()))
}

func cell(f *excelize.File, sheet string, col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	v, _ := f.GetCellValue(sheet, name)
	return v
}