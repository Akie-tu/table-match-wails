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
	Path   string   `json:"path"`
	Errors []string `json:"errors"`
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
	InvoiceType string
	TaxIncluded string
	ItemName    string
	TaxCode     string
	Unit        string
	TaxRate     string
}

func DefaultFixed() FixedContent {
	return FixedContent{InvoiceType: "普通发票", TaxIncluded: "是", ItemName: "滤芯",
		TaxCode: "1090130020000000000", Unit: "个", TaxRate: "0.01"}
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

// 校验单条发票
func validateInvoice(inv *Invoice, fixed FixedContent, idx int) []string {
	var errs []string
	if strings.TrimSpace(inv.Buyer) == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 购买方名称为空", idx))
	}
	if strings.TrimSpace(inv.Amount) == "" {
		errs = append(errs, fmt.Sprintf("第%d行: 金额为空", idx))
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
	return errs
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// GenerateInvoiceXlsx: 生成开票导入xlsx
// templatePath: 官方模板路径; 返回 (outPath, 错误列表)
func GenerateInvoiceXlsx(invoices []*Invoice, fixed FixedContent, templatePath, outPath string) (string, []string, error) {
	if len(invoices) == 0 {
		return "", nil, fmt.Errorf("发票列表为空")
	}
	if templatePath == "" {
		return "", nil, fmt.Errorf("找不到开票模板文件, 请选择模板")
	}
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return "", nil, fmt.Errorf("打开模板失败: %v", err)
	}
	defer f.Close()

	if !hasSheet(f, "1-发票基本信息") || !hasSheet(f, "2-发票明细信息") {
		return "", nil, fmt.Errorf("模板缺少必需工作表")
	}

	// 校验
	var allErrs []string
	for i, inv := range invoices {
		allErrs = append(allErrs, validateInvoice(inv, fixed, i+1)...)
	}
	if len(allErrs) > 0 {
		return "", allErrs, nil
	}

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
		naturalFlag := "是"
		if isNatural != "是" && taxID != "" {
			naturalFlag = "否"
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
		return "", nil, err
	}
	return outPath, nil, nil
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