package backend

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// 回归测试：合并开票（★ 备注/订单号不进合并键）
// 用户原话："合并开票可以舍弃订单编号"

func mkInv(buyer, taxID, qty, amount, remark string) *Invoice {
	return &Invoice{
		InvoiceType: "增值税专用发票", TaxIncluded: "否", Buyer: buyer, TaxID: taxID,
		Qty: qty, Amount: amount, Remark: remark,
		ItemName: "滤芯", TaxCode: "1090130020000000000", Unit: "个", TaxRate: "0.13",
	}
}

func TestMergeIgnoresRemark(t *testing.T) {
	rows := []*Invoice{
		mkInv("测试公司A", "91370306MA3TBJ0T8E", "1", "122.00", "DD001"),
		mkInv("测试公司A", "91370306MA3TBJ0T8E", "2", "244.00", "DD002"), // 只有订单号不同
		mkInv("测试公司B", "91370306MA3TBJ0T9E", "1", "50.00", "DD003"),  // 抬头不同 ⇒ 不合并
	}
	merged, notes := MergeInvoices(rows)
	fmt.Printf("合并前 %d 行 → 合并后 %d 行, 说明 %v\n", len(rows), len(merged), notes)
	if len(merged) != 2 {
		t.Fatalf("期望合并成 2 张发票，实际 %d", len(merged))
	}
	if merged[0].Qty != "3" {
		t.Errorf("数量应相加为 3，实际 %q", merged[0].Qty)
	}
	if merged[0].Amount != "366.00" {
		t.Errorf("金额应相加为 366.00，实际 %q", merged[0].Amount)
	}
	if !strings.Contains(merged[0].Remark, "DD001") || !strings.Contains(merged[0].Remark, "DD002") {
		t.Errorf("合并后备注应保留两个单号，实际 %q", merged[0].Remark)
	}
	if merged[1].Buyer != "测试公司B" {
		t.Errorf("第二张应是测试公司B，实际 %q", merged[1].Buyer)
	}
}

func TestJoinRemarks(t *testing.T) {
	if got := joinRemarks("DD001", "DD002"); got != "DD001；DD002" {
		t.Errorf("拼接错误: %q", got)
	}
	if got := joinRemarks("DD001；DD002", "DD002"); got != "DD001；DD002" {
		t.Errorf("应去重: %q", got)
	}
	long := strings.Repeat("A", 200)
	got := joinRemarks(long, "DD002")
	if len([]rune(got)) > 200 || !strings.Contains(got, "等2单") {
		t.Errorf("超长应收敛: %q", got)
	}
}

// 端到端：真生成 xlsx 并读回校验（用仓库里的官方模板）
func TestGenerateMergedXlsx(t *testing.T) {
	tpl, err := filepath.Abs(filepath.Join("..", "invoice_template.xlsx"))
	if err != nil {
		t.Skip("模板路径解析失败")
	}
	rows := []*Invoice{
		mkInv("测试公司A", "91370306MA3TBJ0T8E", "1", "122.00", "DD001"),
		mkInv("测试公司A", "91370306MA3TBJ0T8E", "2", "244.00", "DD002"),
		mkInv("测试公司B", "91370306MA3TBJ0T9E", "1", "50.00", "DD003"),
	}
	fixed := FixedContent{InvoiceType: "增值税专用发票", TaxIncluded: "否",
		ItemName: "滤芯", TaxCode: "1090130020000000000", Unit: "个", TaxRate: "0.13"}
	out := filepath.Join(t.TempDir(), "merged.xlsx")
	path, errs, warns, err := GenerateInvoiceXlsx(rows, fixed, tpl, out, true)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("校验未过: %v", errs)
	}
	fmt.Println("提醒:", warns)
	fmt.Println("产物:", path)

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("打开产物失败: %v", err)
	}
	defer f.Close()

	get := func(sheet string, col, row int) string {
		v, _ := f.GetCellValue(sheet, colName(col, row))
		return strings.TrimSpace(v)
	}
	// 第4、5行 = 合并后的两张发票
	fmt.Printf("基本信息行4: 抬头=%s 税号=%s 备注=%s\n",
		get("1-发票基本信息", CBillBuyer, 4), get("1-发票基本信息", CBillTaxID, 4), get("1-发票基本信息", CBillRemark, 4))
	fmt.Printf("明细行4:   数量=%s 金额=%s\n", get("2-发票明细信息", CItemQty, 4), get("2-发票明细信息", CItemAmount, 4))
	fmt.Printf("基本信息行5: 抬头=%s\n", get("1-发票基本信息", CBillBuyer, 5))
	fmt.Printf("行6(应为空): 抬头=%q\n", get("1-发票基本信息", CBillBuyer, 6))

	if get("1-发票基本信息", CBillBuyer, 4) != "测试公司A" {
		t.Errorf("行4抬头错误")
	}
	if get("2-发票明细信息", CItemQty, 4) != "3" {
		t.Errorf("行4数量应为 3，实际 %q", get("2-发票明细信息", CItemQty, 4))
	}
	if get("2-发票明细信息", CItemAmount, 4) != "366.00" {
		t.Errorf("行4金额应为 366.00，实际 %q", get("2-发票明细信息", CItemAmount, 4))
	}
	if r := get("1-发票基本信息", CBillRemark, 4); !strings.Contains(r, "DD001") || !strings.Contains(r, "DD002") {
		t.Errorf("行4备注应含两个单号，实际 %q", r)
	}
	if get("1-发票基本信息", CBillBuyer, 5) != "测试公司B" {
		t.Errorf("行5抬头错误")
	}
	if get("1-发票基本信息", CBillBuyer, 6) != "" {
		t.Errorf("第6行不该有数据（合并后只有2张）")
	}
}
