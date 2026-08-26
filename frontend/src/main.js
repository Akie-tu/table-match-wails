// Wails 前端逻辑
import './style.css';

// Wails 注入的绑定 (v2 自动生成于 frontend/wailsjs/go/main/App.js)
import { RunMatch, SelectFile, GenerateInvoice, SelectSavePath } from '../wailsjs/go/main/App';

const $ = (id) => document.getElementById(id);

// ---------- Tab 切换 ----------
document.querySelectorAll('.tab').forEach((btn) => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach((b) => b.classList.remove('active'));
        document.querySelectorAll('.panel').forEach((p) => p.classList.remove('active'));
        btn.classList.add('active');
        $(`tab-${btn.dataset.tab}`).classList.add('active');
    });
});

// ---------- 映射行 ----------
let mapSeq = 0;
function addMapRow() {
    const div = document.createElement('div');
    div.className = 'row maprow';
    div.innerHTML = `
        <input class="m-src" placeholder="源列"/>
        <span>→</span>
        <input class="m-tgt" placeholder="目标列"/>
        <button class="del" onclick="this.parentElement.remove()">×</button>`;
    $('mapRows').appendChild(div);
}
window.addMapRow = addMapRow;
addMapRow(); // 默认一行
addMapRow();

// ---------- 选文件 ----------
async function pickFile(kind) {
    try {
        const path = await SelectFile();
        if (path) $(kind === 'src' ? 'srcPath' : 'tgtPath').value = path;
    } catch (e) {
        console.error(e);
        alert('选择文件失败: ' + e);
    }
}
window.pickFile = pickFile;

// ---------- 开始核对 ----------
async function runMatch() {
    const src = $('srcPath').value.trim();
    const tgt = $('tgtPath').value.trim();
    if (!src || !tgt) { alert('请选择源表和目标表'); return; }

    const fillMap = [];
    document.querySelectorAll('.maprow').forEach((r) => {
        const s = r.querySelector('.m-src').value.trim();
        const t = r.querySelector('.m-tgt').value.trim();
        if (s && t) fillMap.push([s, t]);
    });
    if (!fillMap.length) { alert('请至少填一个回填映射'); return; }

    const result = $('result');
    result.classList.remove('hidden');
    result.innerHTML = '⏳ 核对中…';
    try {
        const res = await RunMatch(
            src, tgt,
            $('srcKey').value.trim(), $('tgtKey').value.trim(),
            fillMap, $('skuCol').value.trim() || '', $('rmkCol').value.trim() || '',
            $('skipExisting').checked,
        );
        result.classList.remove('hidden');
        result.innerHTML = `
            <div class="ok">✅ 匹配: <b>${res.matched}</b> | 多规格: <b>${res.multi}</b> | 未匹配: <b>${res.notfound.length}</b></div>
            <div class="muted">输出: ${res.out_path}</div>
            ${res.notfound.length ? `<div class="warn">未匹配: ${res.notfound.slice(0, 20).join(', ')}${res.notfound.length > 20 ? ' …' : ''}</div>` : ''}`;
    } catch (e) {
        result.innerHTML = `<div class="err">❌ ${e}</div>`;
    }
}
window.runMatch = runMatch;

// ================= 开票模块 =================
let invRows = [];

// 发票类型/含税下拉选项
const INV_TYPES = ['普通发票', '增值税专用发票'];
const TAX_INCS = ['是', '否'];

function invRender() {
    const tbody = document.querySelector('#invTable tbody');
    tbody.innerHTML = '';
    invRows.forEach((r, i) => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td class="idx">${String(i + 1).padStart(3, '0')}</td>
            <td><select class="cell-type">${INV_TYPES.map(t => `<option ${t === r.invoice_type ? 'selected' : ''}>${t}</option>`).join('')}</select></td>
            <td><select class="cell-taxinc">${TAX_INCS.map(t => `<option ${t === r.tax_included ? 'selected' : ''}>${t}</option>`).join('')}</select></td>
            <td><input class="cell-buyer" value="${esc(r.buyer)}" placeholder="购买方名称"/></td>
            <td><input class="cell-taxid" value="${esc(r.tax_id)}" placeholder="税号"/></td>
            <td><select class="cell-natural">
                <option value="">—</option>
                <option value="是" ${r.is_natural === '是' ? 'selected' : ''}>是</option>
                <option value="否" ${r.is_natural === '否' ? 'selected' : ''}>否</option>
            </select></td>
            <td><input class="cell-qty" value="${esc(r.qty)}" placeholder="数量"/></td>
            <td><input class="cell-amount" value="${esc(r.amount)}" placeholder="金额"/></td>
            <td><input class="cell-remark" value="${esc(r.remark)}" placeholder="备注"/></td>
            <td><button class="del" onclick="invDelRow(${i})">×</button></td>`;
        tr.onclick = () => {
            document.querySelectorAll('#invTable tbody tr').forEach((x) => x.classList.remove('sel'));
            tr.classList.add('sel');
        };
        tbody.appendChild(tr);
    });
}
function esc(s) { return String(s || '').replace(/"/g, '&quot;').replace(/</g, '&lt;'); }

function invCollect() {
    document.querySelectorAll('#invTable tbody tr').forEach((tr, i) => {
        if (invRows[i]) {
            invRows[i].invoice_type = tr.querySelector('.cell-type').value;
            invRows[i].tax_included = tr.querySelector('.cell-taxinc').value;
            invRows[i].buyer = tr.querySelector('.cell-buyer').value.trim();
            invRows[i].tax_id = tr.querySelector('.cell-taxid').value.trim();
            invRows[i].is_natural = tr.querySelector('.cell-natural').value;
            invRows[i].qty = tr.querySelector('.cell-qty').value.trim();
            invRows[i].amount = tr.querySelector('.cell-amount').value.trim();
            invRows[i].remark = tr.querySelector('.cell-remark').value.trim();
        }
    });
}

// ---------- 批量粘贴(智能识别, 参考旧项目) ----------
// 清洗单元格: 去货币符号/空格/千分位逗号
function cleanCell(s) {
    return String(s == null ? '' : s).trim()
        .replace(/[¥￥$€\u00A0]/g, '')
        .replace(/\s+/g, '')
        .replace(/,/g, '');
}
function isNum(s) { return s !== '' && /^[+-]?\d+(\.\d+)?$/.test(s); }

// 智能识别粘贴: 支持 单列/两列/多列
// 列类型: text(文字) / amount(含小数) / qty(≤4位整数) / tax_id(≥15位长数字) / remark(>4位数字)
function invSmartPaste(lines) {
    const rows = lines.map((l) => l.split('\t').map(cleanCell));
    const ncols = Math.max(...rows.map((r) => r.length));
    if (!ncols) return 0;

    // 每列分类
    const colType = [];
    for (let j = 0; j < ncols; j++) {
        const vals = rows.map((r) => r[j] || '').filter((v) => v !== '');
        if (vals.length && vals.every(isNum)) {
            const hasDec = vals.some((v) => v.includes('.'));
            const maxLen = Math.max(...vals.map((v) => v.length));
            if (hasDec) colType[j] = 'amount';
            else if (maxLen >= 15) colType[j] = 'tax_id';
            else if (maxLen <= 4) colType[j] = 'qty';
            else colType[j] = 'remark';
        } else {
            colType[j] = 'text';
        }
    }

    // 分配列映射: 列索引 -> 字段
    const mapping = {};
    const used = new Set();
    // 名称 = 第一个文字列
    const nameCol = colType.findIndex((t) => t === 'text');
    if (nameCol >= 0) { mapping[nameCol] = 'buyer'; used.add(nameCol); }
    // 金额列(含小数优先)
    colType.forEach((t, i) => { if (t === 'amount' && !used.has(i)) { mapping[i] = 'amount'; used.add(i); } });
    // 数量列
    colType.forEach((t, i) => { if (t === 'qty' && !used.has(i)) { mapping[i] = 'qty'; used.add(i); } });
    // 税号列
    colType.forEach((t, i) => { if (t === 'tax_id' && !used.has(i)) { mapping[i] = 'tax_id'; used.add(i); } });
    // 备注列
    colType.forEach((t, i) => { if (t === 'remark' && !used.has(i)) { mapping[i] = 'remark'; used.add(i); } });
    // 剩余文字列(第2个起): tax_id → remark
    let fi = 0;
    const restFields = ['tax_id', 'remark'];
    colType.forEach((t, i) => { if (t === 'text' && !used.has(i) && fi < restFields.length) { mapping[i] = restFields[fi++]; used.add(i); } });

    // 追加行
    let added = 0;
    rows.forEach((cells) => {
        const row = { invoice_type: $('invType').value, tax_included: $('invTaxInc').value,
            buyer: '', tax_id: '', is_natural: '', qty: '', amount: '', remark: '',
            item_name: '', tax_code: '', unit: '', tax_rate: '' };
        let hasVal = false;
        cells.forEach((c, j) => {
            if (c === '') return;
            const f = mapping[j];
            if (f && row[f] === '') { row[f] = c; hasVal = true; }
        });
        if (hasVal) { invRows.push(row); added++; }
    });
    return added;
}

// Ctrl+V: 焦点在发票Tab任意位置都拦截做智能粘贴
document.addEventListener('keydown', async (e) => {
    if (!(e.ctrlKey || e.metaKey) || e.key.toLowerCase() !== 'v') return;
    const active = document.activeElement;
    if (!active || !active.closest('#tab-invoice')) return;
    e.preventDefault();
    let txt = '';
    try { txt = await navigator.clipboard.readText(); } catch (err) { return; }
    if (!txt) return;
    invCollect();
    const lines = txt.split(/\r?\n/).filter((l) => l.trim() !== '');
    if (!lines.length) return;
    // 单行单列 → 填入当前焦点单元格(若有)
    if (lines.length === 1 && !lines[0].includes('\t') && active.tagName === 'INPUT') {
        const keyMap = { 'cell-buyer': 'buyer', 'cell-taxid': 'tax_id', 'cell-qty': 'qty', 'cell-amount': 'amount', 'cell-remark': 'remark' };
        const key = keyMap[active.className];
        const tr = active.closest('tr');
        if (key && tr) {
            const idx = [...tr.parentElement.children].indexOf(tr);
            if (idx >= 0 && invRows[idx]) {
                invRows[idx][key] = cleanCell(lines[0]);
                invRender();
                return;
            }
        }
    }
    const added = invSmartPaste(lines);
    if (added) { invRender(); alert(`✔ 已粘贴 ${added} 行(智能识别列)`); }
});

// 删除列: 清空选中列所有行的数据
function invClearCol() {
    const col = $('invClearCol').value;
    const keyMap = { '发票类型': 'invoice_type', '含税': 'tax_included', '名称': 'buyer',
        '税号': 'tax_id', '自然人': 'is_natural', '数量': 'qty', '金额': 'amount', '备注': 'remark' };
    const key = keyMap[col];
    if (!key) return;
    if (!confirm(`清空「${col}」列所有行的数据？`)) return;
    invCollect();
    invRows.forEach((r) => { r[key] = ''; });
    invRender();
    alert(`✔ 已清空「${col}」列`);
}
window.invClearCol = invClearCol;

function invAddRow() {
    invCollect();
    invRows.push({ invoice_type: $('invType').value, tax_included: $('invTaxInc').value,
        buyer: '', tax_id: '', is_natural: '', qty: '', amount: '', remark: '',
        item_name: '', tax_code: '', unit: '', tax_rate: '' });
    invRender();
}
window.invAddRow = invAddRow;

function invDelRow(i) { invCollect(); invRows.splice(i, 1); invRender(); }
window.invDelRow = invDelRow;

function invDelSel() {
    const tr = document.querySelector('#invTable tbody tr.sel');
    if (!tr) { alert('请先点选一行'); return; }
    const idx = [...tr.parentElement.children].indexOf(tr);
    invRows.splice(idx, 1);
    invRender();
}
window.invDelSel = invDelSel;

function invClear() {
    if (!confirm('清空全部发票？')) return;
    invRows = [];
    invRender();
}
window.invClear = invClear;

async function pickTemplate() {
    const path = await SelectFile();
    if (path) $('invTpl').value = path;
}
window.pickTemplate = pickTemplate;

async function invGenerate() {
    invCollect();
    if (!invRows.length) { alert('请先添加发票行'); return; }
    const fixed = {
        invoice_type: $('invType').value, tax_included: $('invTaxInc').value,
        item_name: $('invItem').value.trim() || '滤芯', tax_code: $('invCode').value.trim() || '1090130020000000000',
        unit: $('invUnit').value.trim() || '个', tax_rate: $('invRate').value.trim() || '0.01',
    };
    const out = await SelectSavePath('开票导入.xlsx');
    if (!out) return;
    const res = $('invResult');
    res.classList.remove('hidden');
    res.innerHTML = '⏳ 生成中…';
    try {
        const r = await GenerateInvoice(invRows, fixed, $('invTpl').value.trim(), out);
        if (r.errors && r.errors.length) {
            res.innerHTML = `<div class="err">❌ 校验失败:<br/>${r.errors.join('<br/>')}</div>`;
            return;
        }
        res.innerHTML = `<div class="ok">✅ 生成成功: ${r.path}</div>`;
    } catch (e) {
        res.innerHTML = `<div class="err">❌ ${e}</div>`;
    }
}
window.invGenerate = invGenerate;

async function invImportDetail() {
    const path = await SelectFile();
    if (!path) return;
    try {
        // 用 Go 侧 ReadSheet 读明细(首行表头) — 需要新增绑定, 先用简单方式: 提示
        alert('导入明细功能开发中(下一版)');
    } catch (e) {
        alert('导入失败: ' + e);
    }
}
window.invImportDetail = invImportDetail;