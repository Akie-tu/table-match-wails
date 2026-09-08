package backend

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 发票数据
type Invoice struct {
	InvoiceType string `json:"invoice_type"` // 发票类型
	TaxIncluded string `json:"tax_included"` // 是否含税
	IsNatural   string `json:"is_natural"`   // 自然人标识
	Buyer       string `json:"buyer"`        // 购买方名称
	TaxID       string `json:"tax_id"`       // 纳税人识别号
	Remark      string `json:"remark"`       // 备注
	ItemName    string `json:"item_name"`    // 项目名称
	TaxCode     string `json:"tax_code"`     // 税收编码
	Unit        string `json:"unit"`         // 单位
	Qty         string `json:"qty"`          // 数量
	Amount      string `json:"amount"`       // 金额
	TaxRate     string `json:"tax_rate"`     // 税率
}

// 生成结果(供前端单对象返回)
type InvoiceResult struct {
	Path     string   `json:"path"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"` // 提醒但不拦截(如税号为空)
}

// 基本信息表列号
const (
	CBillSerial   = 1 // 流水号
	CBillType     = 2 // 发票类型
	CBillTaxInc   = 4 // 是否含税
	CBillNatural  = 5 // 自然人标识
	CBillBuyer    = 6 // 购买方名称
	CBillTaxID    = 7 // 纳税人识别号
	CBillRemark   = 23 // 备注
)

// 明细表列号
const (
	CItemSerial  = 1 // 流水号
	CItemName    = 2 // 项目名称
	CItemTaxCode = 3 // 税收编码
	CItemUnit    = 5 // 单位
	CItemQty     = 6 // 数量
	CItemAmount  = 8 // 金额
	CItemRate    = 9 // 税率
)

// 默认固定内容
type FixedContent struct {
	InvoiceType string              `json:"invoice_type"` // 发票类型(默认)
	TaxIncluded string              `json:"tax_included"` // 是否含税(默认)
	ItemName    string              `json:"item_name"`    // 项目名称(默认)
	TaxCode     string              `json:"tax_code"`     // 税收编码(默认)
	Unit        string              `json:"unit"`         // 单位(默认)
	TaxRate     string              `json:"tax_rate"`     // 税率(默认)
	Options     map[string][]string `json:"options"`      // 各字段选项(行级下拉用, 可扩展)
	// Options键: invoice_type/tax_included/item_name/tax_code/unit/tax_rate
}

// 默认固定内容: 脚本本身不携带业务默认值
// 实际值由前端从本地配置文件(fixed_config.json)加载后传入
func DefaultFixed() FixedContent {
	return FixedContent{} // 全部为空
}

// 流水号: 1 -> "001"
func MakeSerial(n int) string {
	return fmt.Sprintf("%03d", n)
}

// 金额两位小数
func normalizeAmount(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return v // 前端已格式化, 这里简单透传(可加小数处理)
}

// MergeInvoices: 按(税号+抬头+发票类型+含税+自然人+项目名称+单位+税率+备注)合并数量金额
// 仅在勾选"合并开票"时调用; 返回合并后的发票列表和合并说明
func MergeInvoices(invoices []*Invoice) ([]*Invoice, []string) {
	if len(invoices) <= 1 {
		return invoices, nil
	}
	type key struct {
		taxID, buyer, invType, taxInc, natural, item, unit, rate, remark string
	}
	idx := map[key]int{} // key -> merged序号
	merged := []*Invoice{}
	var notes []string

	for _, inv := range invoices {
		k := key{
			taxID:   strings.TrimSpace(inv.TaxID),
			buyer:   strings.TrimSpace(inv.Buyer),
			invType: inv.InvoiceType,
			taxInc:  inv.TaxIncluded,
			natural: inv.IsNatural,
			item:    inv.ItemName,
			unit:    inv.Unit,
			rate:    inv.TaxRate,
			remark:  strings.TrimSpace(inv.Remark),
		}
		if ki, ok := idx[k]; ok {
			tgt := merged[ki]
			tgt.Qty = addQty(tgt.Qty, inv.Qty)
			tgt.Amount = addAmount(tgt.Amount, inv.Amount)
			notes = append(notes, fmt.Sprintf("已合并: %s(税号%s)的数量金额并入同行", inv.Buyer, k.taxID))
		} else {
			idx[k] = len(merged)
			cp := *inv
			merged = append(merged, &cp)
		}
	}
	return merged, notes
}

// 数量相加(支持小数; 无法解析则取非空者)
func addQty(a, b string) string {
	fa, fb := toFloat(a), toFloat(b)
	switch {
	case fa == nil && fb == nil:
		return b
	case fa == nil:
		return b
	case fb == nil:
		return a
	}
	return trimFloat(*fa + *fb)
}

// 金额相加(保留2位小数)
func addAmount(a, b string) string {
	fa, fb := toFloat(a), toFloat(b)
	switch {
	case fa == nil && fb == nil:
		return b
	case fa == nil:
		return b
	case fb == nil:
		return a
	}
	return fmt.Sprintf("%.2f", *fa+*fb)
}

func toFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ",", "")
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return nil
	}
	return &f
}

func trimFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", f), "0"), ".")
}

// 校验单条发票
func validateInvoice(inv *Invoice, fixed FixedContent, idx int) (errs []string, warns []string) {
	if strings.TrimSpace(inv.Buyer) == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 购买方名称为空", idx))
	}
	if strings.TrimSpace(inv.Amount) == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 金额为空", idx))
	}
	// 非自然人税号为空: 仅提醒不拦截(部分发票确实无税号, 可正常开具)
	if strings.TrimSpace(inv.IsNatural) != "是" && strings.TrimSpace(inv.TaxID) == "" {
		warns = append(warns, fmt.Sprintf("第%d行: 未提供纳税人识别号(可正常开具, 如需可补填后重新生成)", idx))
	}
	// 灵活版校验(项目/编码/单位/税率)
	item := firstNonEmpty(inv.ItemName, fixed.ItemName)
	code := firstNonEmpty(inv.TaxCode, fixed.TaxCode)
	unit := firstNonEmpty(inv.Unit, fixed.Unit)
	rate := firstNonEmpty(inv.TaxRate, fixed.TaxRate)
	if item == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 项目名称为空", idx))
	}
	if code == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 税收编码为空", idx))
	}
	if unit == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 单位为空", idx))
	}
	if rate == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 税率为空", idx))
	}
	return errs, warns
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// 生成开票导入文件: 填数据到官方模板并另存
// 参数: 官方模板路径; 返回(输出路径, 错误列表)
func GenerateInvoiceXlsx(invoices []*Invoice, fixed FixedContent, templatePath, outPath string, mergeAll bool) (string, []string, []string, error) {
	if len(invoices) == 0 {
		return "", nil, nil, fmt.Errorf("发票列表为空")
	}
	var mergeNotes []string
	if mergeAll {
		invoices, mergeNotes = MergeInvoices(invoices)
	}
	if templatePath == "" {
		return "", nil, nil, fmt.Errorf("找不到开票模板文件, 请选择模板")
	}
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("打开模板失败: %v", err)
	}
	defer f.Close()

	if !hasSheet(f, "1-发票基本信息") || !hasSheet(f, "2-发票明细信息") {
		return "", nil, nil, fmt.Errorf("模板缺少必需工作表")
	}

	// 校验(errs拦截 / warns仅提醒不拦截)
	var allErrs, allWarns []string
	for i, inv := range invoices {
		e, w := validateInvoice(inv, fixed, i+1)
		allErrs = append(allErrs, e...)
		allWarns = append(allWarns, w...)
	}
	if len(allErrs) > 0 {
		return "", allErrs, nil, nil
	}
	allWarns = append(mergeNotes, allWarns...)

	// 清空模板已有数据(第4行起)
	for _, sheet := range []string{"1-发票基本信息", "2-发票明细信息"} {
		rows, err := f.GetRows(sheet)
		if err == nil && len(rows) > 3 {
			for r := len(rows); r >= 4; r-- {
				f.RemoveRow(sheet, r)
			}
		}
	}

	// 填数据
	for i, inv := range invoices {
		row := i + 4 // 第4行起
		serial := MakeSerial(i + 1)

		it := firstNonEmpty(inv.InvoiceType, fixed.InvoiceType)
		taxinc := firstNonEmpty(inv.TaxIncluded, fixed.TaxIncluded)
		item := firstNonEmpty(inv.ItemName, fixed.ItemName)
		code := firstNonEmpty(inv.TaxCode, fixed.TaxCode)
		unit := firstNonEmpty(inv.Unit, fixed.Unit)
		rate := firstNonEmpty(inv.TaxRate, fixed.TaxRate)

		isNatural := strings.TrimSpace(inv.IsNatural)
		taxID := strings.TrimSpace(inv.TaxID)
		// 修复: 原逻辑默认"是"且靠税号反推, 用户选"否"但没填税号会被静默改成自然人
		// 现在: 默认"否", 仅当行级明确选择"是/个人/自然人"才写"是"
		naturalFlag := "否"
		if isNatural == "是" {
			naturalFlag = "是"
		}

		// 基本信息表
		f.SetCellValue("1-发票基本信息", colName(CBillSerial, row), serial)
		f.SetCellValue("1-发票基本信息", colName(CBillType, row), it)
		f.SetCellValue("1-发票基本信息", colName(CBillTaxInc, row), taxinc)
		f.SetCellValue("1-发票基本信息", colName(CBillNatural, row), naturalFlag)
		f.SetCellValue("1-发票基本信息", colName(CBillBuyer, row), strings.TrimSpace(inv.Buyer))
		if taxID != "" {
			f.SetCellValue("1-发票基本信息", colName(CBillTaxID, row), taxID)
		}
		if rmk := strings.TrimSpace(inv.Remark); rmk != "" {
			f.SetCellValue("1-发票基本信息", colName(CBillRemark, row), rmk)
		}

		// 明细表
		f.SetCellValue("2-发票明细信息", colName(CItemSerial, row), serial)
		f.SetCellValue("2-发票明细信息", colName(CItemName, row), item)
		f.SetCellValue("2-发票明细信息", colName(CItemTaxCode, row), code)
		f.SetCellValue("2-发票明细信息", colName(CItemUnit, row), unit)
		if q := strings.TrimSpace(inv.Qty); q != "" {
			f.SetCellValue("2-发票明细信息", colName(CItemQty, row), q)
		}
		f.SetCellValue("2-发票明细信息", colName(CItemAmount, row), normalizeAmount(inv.Amount))
		f.SetCellValue("2-发票明细信息", colName(CItemRate, row), rate)
	}

	if outPath == "" {
		outPath = "开票导入.xlsx"
	}
	if err := f.SaveAs(outPath); err != nil {
		return "", nil, nil, err
	}
	return outPath, nil, allWarns, nil
}

func hasSheet(f *excelize.File, name string) bool {
	for _, s := range f.GetSheetList() {
		if s == name {
			return true
		}
	}
	return false
}

func colName(col, row int) string {
	c, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return ""
	}
	return c
}