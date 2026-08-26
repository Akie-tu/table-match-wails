# 电商工具 Go + Wails 重构版 (试验)

用 Go + Wails v2 + Excelize 重构的电商工具，替代 Python/tkinter 版。

## 优势
- 单文件 EXE 更小 (预计 8-12MB vs Python 19MB)
- 原生编译，更少杀软误报
- Excelize 直接兼容快麦等导出的 xlsx (Python openpyxl 报 Fill 错误)
- 现代 Web 前端界面 (HTML/JS 暗色主题)

## 结构
```
main.go          Wails 入口
app.go           App 绑定(暴露给前端的方法)
backend/
  match.go       表格核对核心逻辑 (Excelize)
frontend/        HTML/JS 前端界面
.github/workflows/build-wails.yml  Windows EXE 构建 (tag w* 触发)
```

## 本地开发
```bash
# 后端逻辑测试(无需GUI)
go run ./test <源表> <目标表> <源键> <目标键> [映射 源:目标...]

# Wails 全量构建
wails build
```

## 已实测
- 快麦售后明细 (235行/42列) 表格核对: 216 行全部匹配 ✓
- Wails 前端构建 ✓

BY 大萝北拔萝卜