package backend

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 表格核对: 用源表某字段去目标表检索, 回填字段
type MatchResult struct {
	Matched   int      `json:"matched"`
	Multi     int      `json:"multi"`
	NotFound  []string `json:"notfound"`
	OutPath   string   `json:"out_path"`
}

// FindHeaderRow: 前6行内找含关键词的表头行
func findHeaderRow(ws *excelize.File, sheet string, keywords []string) (int, int, error) {
	rows, err := ws.GetRows(sheet)
	if err != nil {
		return 0, 0, err
	}
	limit := len(rows)
	if limit > 6 {
		limit = 6
	}
	// 找最大列数
	maxCol := 0
	for r := 0; r < limit; r++ {
		if len(rows[r]) > maxCol {
			maxCol = len(rows[r])
		}
	}
	for r := 0; r < limit; r++ {
		hit := 0
		row := rows[r]
		for _, k := range keywords {
			kl := strings.ToLower(k)
			for c := 0; c < len(row); c++ {
				if strings.Contains(strings.ToLower(row[c]), kl) {
					hit++
					break
				}
			}
		}
		if hit >= len(keywords) {
			return r + 1, maxCol, nil // 1-based 行号
		}
	}
	return 0, 0, fmt.Errorf("找不到表头(含 %v)", keywords)
}

// ColIndexByHeader: 表头行中找列号(0-based)
func colIndexByHeader(rows [][]string, headerRow int, headerName string) int {
	if headerRow-1 >= len(rows) {
		return -1
	}
	for c, v := range rows[headerRow-1] {
		if strings.TrimSpace(v) == headerName {
			return c
		}
	}
	return -1
}

// ListSheets: 列出工作簿所有sheet名
func ListSheets(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.GetSheetList(), nil
}

// ReadAllRows: 读取指定sheet全部行([][]string)
func ReadAllRows(path, sheet string) ([][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if sheet == "" {
		sheet = f.GetSheetList()[0]
	}
	return f.GetRows(sheet)
}

// RunMatch: 表格核对(见上方MatchResult)
func RunMatch(srcPath, tgtPath, srcKey, tgtKey string, fillMap [][2]string,
	skuCol, remarkCol string, skipExisting bool) (*MatchResult, error) {
	// 目标表
	tf, err := excelize.OpenFile(tgtPath)
	if err != nil {
		return nil, fmt.Errorf("打开目标表失败: %v", err)
	}
	defer tf.Close()
	tSheet := tf.GetSheetList()[0]
	tRows, err := tf.GetRows(tSheet)
	if err != nil {
		return nil, err
	}
	thr, _, err := findHeaderRow(tf, tSheet, []string{tgtKey})
	if err != nil {
		return nil, err
	}
	keyCol := colIndexByHeader(tRows, thr, tgtKey)
	if keyCol < 0 {
		return nil, fmt.Errorf("目标表没有列 '%s'", tgtKey)
	}
	// 目标列映射 {源列: []目标列索引}
	tgtCols := map[string][]int{}
	for _, fm := range fillMap {
		ci := colIndexByHeader(tRows, thr, fm[1])
		if ci >= 0 {
			tgtCols[fm[0]] = append(tgtCols[fm[0]], ci)
		}
	}
	skuIdx := -1
	if skuCol != "" {
		skuIdx = colIndexByHeader(tRows, thr, skuCol)
	}

	// 建立索引: 键(逗号/空格分隔) -> 行
	idx := map[string][][]string{}
	for r := thr; r < len(tRows); r++ {
		row := tRows[r]
		if keyCol >= len(row) {
			continue
		}
		k := row[keyCol]
		if strings.TrimSpace(k) == "" {
			continue
		}
		parts := strings.FieldsFunc(k, func(r rune) bool {
			return r == ',' || r == '，' || r == ' '
		})
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				idx[p] = append(idx[p], row)
			}
		}
	}

	// 源表
	sf, err := excelize.OpenFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("打开源表失败: %v", err)
	}
	defer sf.Close()
	sSheet := sf.GetSheetList()[0]
	sRows, _ := sf.GetRows(sSheet)
	shr, _, err := findHeaderRow(sf, sSheet, []string{srcKey})
	if err != nil {
		return nil, err
	}
	srcKeyCol := colIndexByHeader(sRows, shr, srcKey)
	srcFillCols := map[string]int{}
	for _, fm := range fillMap {
		ci := colIndexByHeader(sRows, shr, fm[0])
		if ci >= 0 {
			srcFillCols[fm[0]] = ci
		}
	}
	remarkIdx := -1
	if remarkCol != "" {
		remarkIdx = colIndexByHeader(sRows, shr, remarkCol)
	}

	res := &MatchResult{}
	// 逐行处理并写回源表
	for r := shr; r < len(sRows); r++ {
		row := sRows[r]
		if srcKeyCol >= len(row) {
			continue
		}
		k := strings.TrimSpace(row[srcKeyCol])
		if k == "" {
			continue
		}
		hits := idx[k]
		if len(hits) == 0 {
			res.NotFound = append(res.NotFound, k)
			continue
		}
		res.Matched++
		// 回填
		for _, fm := range fillMap {
			sci := srcFillCols[fm[0]]
			tcis := tgtCols[fm[0]]
			if sci < 0 || len(tcis) == 0 {
				continue
			}
			cell := row[sci]
			if skipExisting && strings.TrimSpace(cell) != "" {
				continue
			}
			// 写回(需要Cell坐标)
			cellName, _ := excelize.CoordinatesToCellName(sci+1, r+1)
			val := ""
			if tcis[0] < len(hits[0]) {
				val = hits[0][tcis[0]]
			}
			if val != "" {
				sf.SetCellValue(sSheet, cellName, val)
			}
		}
		// 多规格标记
		if remarkIdx >= 0 && skuIdx >= 0 && len(hits) >= 2 {
			uniq := map[string]bool{}
			for _, h := range hits {
				if skuIdx < len(h) {
					s := strings.TrimSpace(h[skuIdx])
					if s != "" {
						uniq[s] = true
					}
				}
			}
			if len(uniq) >= 2 {
				rName, _ := excelize.CoordinatesToCellName(remarkIdx+1, r+1)
				sf.SetCellValue(sSheet, rName, "多规格")
				res.Multi++
			}
		}
	}

	// 保存(源表另存)
	out := strings.TrimSuffix(srcPath, ".xlsx") + "_结果.xlsx"
	if err := sf.SaveAs(out); err != nil {
		return nil, err
	}
	res.OutPath = out
	return res, nil
}