package backend

import (
	"encoding/csv"
	"io"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 导入明细结果
type ImportResult struct {
	Rows    []*Invoice `json:"rows"`    // 解析出的发票(未含固定内容, 由前端合并)
	Imported int       `json:"imported"` // 导入条数
	Missing []string   `json:"missing"`  // 未识别列
}

// 列匹配规则(关键词)
var importRules = map[string][]string{
	"buyer":        {"抬头", "购买方名称", "购方名称", "买方名称", "发票抬头"},
	"tax_id":       {"税号", "识别号", "纳税人"},
	"amount":       {"发票金额", "金额", "价税合计", "合计金额"},
	"qty":          {"商品数量", "数量"},
	"type":         {"抬头类型", "购方类型", "买方类型", "抬头类别", "客户类型"},
	"invoice_type": {"发票类型"},
	"remark":       {"备注"},
}

// 读取表格/文本文件为二维数据(首行是表头)
// 先探测真实格式(文件头): ZIP(PK)=xlsx, 否则按CSV解析 —— 防 .csv 扩展名实际是 Excel 文件
func ReadDetailFile(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	head := make([]byte, 4)
	if _, err := f.Read(head); err != nil {
		return nil, err
	}
	// 回退到文件开头
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	isXlsx := len(head) >= 4 && head[0] == 'P' && head[1] == 'K' // xlsx/zip 魔数 PK\x03\x04
	if isXlsx {
		xlsx, err := excelize.OpenFile(path)
		if err != nil {
			return nil, err
		}
		defer xlsx.Close()
		sheet := xlsx.GetSheetList()[0]
		rows, err := xlsx.GetRows(sheet)
		if err != nil {
			return nil, err
		}
		return rows, nil
	}
	r := csv.NewReader(f)
	r.LazyQuotes = true
	var rows [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

// 列匹配: 表头行 → 列索引映射 {field: colIndex}
func MatchDetailColumns(headers []string) map[string]int {
	// 类型列优先(防"抬头"抢"抬头类型")
	typeFound := -1
	for i, h := range headers {
		hl := strings.ToLower(h)
		for _, k := range importRules["type"] {
			if strings.Contains(hl, strings.ToLower(k)) {
				typeFound = i
				break
			}
		}
		if typeFound >= 0 {
			break
		}
	}
	colMap := map[string]int{"type": typeFound}
	for field, kws := range importRules {
		if field == "type" {
			continue
		}
		found := -1
		for i, h := range headers {
			if i == typeFound {
				continue
			}
			if field == "buyer" && (strings.Contains(h, "类型") || strings.Contains(h, "类别")) {
				continue
			}
			hl := strings.ToLower(h)
			for _, k := range kws {
				if strings.Contains(hl, strings.ToLower(k)) {
					found = i
					break
				}
			}
			if found >= 0 {
				break
			}
		}
		colMap[field] = found
	}
	return colMap
}

// 发票类型映射: 电子普通→普通发票, 电子专用/专用→增值税专用发票
func mapInvoiceType(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return ""
	}
	if strings.Contains(t, "专用") {
		return "增值税专用发票"
	}
	if strings.Contains(t, "普通") {
		return "普通发票"
	}
	return t
}

// 找表头行: 扫描前6行, 找含表头关键词最多的行(模板表头可能在R3)
func FindHeaderRow(rows [][]string) int {
	bestRow, bestScore := 0, 0
	limit := len(rows)
	if limit > 6 {
		limit = 6
	}
	for r := 0; r < limit; r++ {
		score := 0
		for _, c := range rows[r] {
			for _, kws := range importRules {
				for _, k := range kws {
					if strings.Contains(strings.ToLower(c), strings.ToLower(k)) {
						score++
						break
					}
				}
			}
		}
		if score > bestScore {
			bestScore = score
			bestRow = r
		}
	}
	return bestRow
}

// 导入明细: 读文件 → 找表头 → 匹配列 → 解析发票行
func ImportInvoiceDetail(path string) (*ImportResult, error) {
	rows, err := ReadDetailFile(path)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, nil // 无数据
	}
	hr := FindHeaderRow(rows)
	headers := make([]string, len(rows[hr]))
	for i, h := range rows[hr] {
		headers[i] = strings.TrimSpace(h)
	}
	colMap := MatchDetailColumns(headers)

	// 未识别列
	var missing []string
	label := map[string]string{"buyer": "购买方名称", "tax_id": "纳税人识别号",
		"amount": "金额", "qty": "数量", "type": "抬头类型"}
	for f, l := range label {
		if colMap[f] < 0 {
			missing = append(missing, l)
		}
	}

	res := &ImportResult{Missing: missing}
	for i := hr + 1; i < len(rows); i++ {
		r := rows[i]
		get := func(field string) string {
			ci := colMap[field]
			if ci < 0 || ci >= len(r) {
				return ""
			}
			return strings.TrimSpace(r[ci])
		}
		buyer := get("buyer")
		if buyer == "" {
			continue
		}
		inv := &Invoice{
			Buyer:       buyer,
			TaxID:       get("tax_id"),
			Qty:         get("qty"),
			Amount:      get("amount"),
			Remark:      get("remark"),
			IsNatural:   "",
			InvoiceType: mapInvoiceType(get("invoice_type")),
		}
		// 抬头类型=个人 → 自然人"是"
		t := get("type")
		if t != "" && (strings.Contains(t, "个人") || strings.Contains(t, "自然人")) {
			inv.IsNatural = "是"
		}
		res.Rows = append(res.Rows, inv)
	}
	res.Imported = len(res.Rows)
	return res, nil
}