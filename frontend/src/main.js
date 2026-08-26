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
            <td><input class="cell-natural" value="${esc(r.is_natural)}" placeholder="是/否"/></td>
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
            invRows[i].is_natural = tr.querySelector('.cell-natural').value.trim();
            invRows[i].qty = tr.querySelector('.cell-qty').value.trim();
            invRows[i].amount = tr.querySelector('.cell-amount').value.trim();
            invRows[i].remark = tr.querySelector('.cell-remark').value.trim();
        }
    });
}

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
        const [path, errs, err] = await GenerateInvoice(invRows, fixed, $('invTpl').value.trim(), out);
        if (errs && errs.length) {
            res.innerHTML = `<div class="err">❌ 校验失败:<br/>${errs.join('<br/>')}</div>`;
            return;
        }
        if (err) {
            res.innerHTML = `<div class="err">❌ ${err}</div>`;
            return;
        }
        res.innerHTML = `<div class="ok">✅ 生成成功: ${path}</div>`;
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