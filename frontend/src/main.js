// 前端界面逻辑
import './style.css';

// 注入的前端绑定方法(由构建工具自动生成)
import { RunMatch, SelectFile, GenerateInvoice, SelectSavePath, ImportInvoiceDetail, SelectDir, RunImgConvert, SaveEmailConfig, LoadEmailConfig, SendEmail, EmailPreset, SaveFixedConfig, LoadFixedConfig, CleanTaxID, FindInvoiceTemplate } from '../wailsjs/go/main/App';

const $ = (id) => document.getElementById(id);

// 标签页切换逻辑
document.querySelectorAll('.tab').forEach((btn) => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach((b) => b.classList.remove('active'));
        document.querySelectorAll('.panel').forEach((p) => p.classList.remove('active'));
        btn.classList.add('active');
        $(`tab-${btn.dataset.tab}`).classList.add('active');
    });
});

// 回填映射行管理
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

// 选择文件
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

// 开始核对
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

// 发票处理模块
let invRows = [];
// 初始化预置一行空发票(与旧版一致, 始终可粘贴)
function emptyRow() {
    return { invoice_type: '普通发票', tax_included: '是', buyer: '', tax_id: '',
        is_natural: '', qty: '', amount: '', remark: '', item_name: '', tax_code: '', unit: '', tax_rate: '' };
}
invRows.push(emptyRow());

// 发票类型/含税下拉选项
const INV_TYPES = ['普通发票', '增值税专用发票'];
const TAX_INCS = ['是', '否'];
// 项目名称选项(行级下拉, 从配置读取, 可扩展)
let itemOptions = [];

function invRender() {
    const tbody = document.querySelector('#invTable tbody');
    tbody.innerHTML = '';
    // 行级下拉生成器(选项来自配置)
    const optSel = (field, cur) => {
        const opts = (fixedOptions[field] || []).map((t) => `<option ${t === cur ? 'selected' : ''}>${t}</option>`).join('');
        return `<select class="cell-${field}">${opts}<option value="" ${!cur ? 'selected' : ''}>—</option></select>`;
    };
    invRows.forEach((r, i) => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td class="idx">${String(i + 1).padStart(3, '0')}</td>
            <td>${optSel('invoice_type', r.invoice_type)}</td>
            <td>${optSel('tax_included', r.tax_included)}</td>
            <td>${optSel('item_name', r.item_name)}</td>
            <td><select class="cell-natural">
                <option value="">—</option>
                <option value="是" ${r.is_natural === '是' ? 'selected' : ''}>是</option>
                <option value="否" ${r.is_natural === '否' ? 'selected' : ''}>否</option>
            </select></td>
            <td><input class="cell-buyer" value="${esc(r.buyer)}" placeholder="购买方名称"/></td>
            <td><input class="cell-taxid" value="${esc(r.tax_id)}" placeholder="税号"/></td>
            <td>${optSel('unit', r.unit)}</td>
            <td><input class="cell-qty" value="${esc(r.qty)}" placeholder="数量"/></td>
            <td><input class="cell-amount" value="${esc(r.amount)}" placeholder="金额"/></td>
            <td>${optSel('tax_rate', r.tax_rate)}</td>
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
            invRows[i].invoice_type = tr.querySelector('.cell-invoice_type') ? tr.querySelector('.cell-invoice_type').value : invRows[i].invoice_type;
            invRows[i].tax_included = tr.querySelector('.cell-tax_included') ? tr.querySelector('.cell-tax_included').value : invRows[i].tax_included;
            invRows[i].item_name = tr.querySelector('.cell-item_name') ? tr.querySelector('.cell-item_name').value : invRows[i].item_name;
            invRows[i].tax_code = $('invCode').value || invRows[i].tax_code; // 税收编码用固定内容(批量一致)
            invRows[i].unit = tr.querySelector('.cell-unit') ? tr.querySelector('.cell-unit').value : invRows[i].unit;
            invRows[i].tax_rate = tr.querySelector('.cell-tax_rate') ? tr.querySelector('.cell-tax_rate').value : invRows[i].tax_rate;
            invRows[i].buyer = tr.querySelector('.cell-buyer').value.trim();
            invRows[i].tax_id = tr.querySelector('.cell-taxid').value.trim();
            invRows[i].is_natural = tr.querySelector('.cell-natural').value;
            invRows[i].qty = tr.querySelector('.cell-qty').value.trim();
            invRows[i].amount = tr.querySelector('.cell-amount').value.trim();
            invRows[i].remark = tr.querySelector('.cell-remark').value.trim();
        }
    });
}

// 批量粘贴功能(严格按旧版逻辑)
// 清洗: 去引号/货币符号/零宽, 裁尾部空列
function cleanCell(s) {
    return String(s == null ? '' : s)
        .replace(/[\u200b\u200c\u200d\ufeff\u3000]/g, '')
        .replace(/[¥￥$€]/g, '')
        .stripQ()
        .trim();
}
String.prototype.stripQ = function () { return this.replace(/^["'""]+|["'""]+$/g, ''); };

// 数字判断(允许逗号/连字符/空格) — 与旧版判断逻辑一致
function looksLikeNumber(s) {
    if (s === '') return false;
    const s2 = String(s).replace(/[,，\-]/g, '').replace(/\s+/g, '');
    if (s2 === '') return false;
    return !isNaN(Number(s2));
}

// 粘贴: 发票页激活时全局拦截; 但焦点在输入框/文本域时放行默认粘贴(填充当前格)
document.addEventListener('paste', (e) => {
    const active = document.activeElement;
    // 焦点在输入框/文本域/下拉 → 交给默认行为(单格填充)
    if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA' || active.tagName === 'SELECT')) return;
    const invPanel = $('tab-invoice');
    if (!invPanel || !invPanel.classList.contains('active')) return;
    e.preventDefault();
    let txt = '';
    try { txt = (e.clipboardData || window.clipboardData).getData('text'); } catch (err) { return; }
    if (!txt) return;
    invCollect();

    // 清洗数据(与旧版粘贴逻辑一致)
    const rawRows = txt.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
    const rows = [];
    for (const r of rawRows) {
        const rr = r.split('\t').map(cleanCell);
        while (rr.length && rr[rr.length - 1] === '') rr.pop(); // 去尾部空列
        if (rr.some((c) => c !== '')) rows.push(rr);
    }
    if (!rows.length) return;

    // 起始行: 总是从第0行开始填充(用户要求: 任何时候粘贴从第一行开始)
    const startRow = 0;

    // 判断起始列(与旧版逻辑一致)
    const keyCols = ['invoice_type', 'tax_included', 'buyer', 'tax_id', 'is_natural', 'qty', 'amount', 'remark'];
    const ncols = Math.max(...rows.map((r) => r.length));
    let startKey;

    if (ncols === 2 && rows.every((r) => r.every((c) => c === '' || looksLikeNumber(c)))) {
        // 两列纯数字: 有小数列→金额(6), 整数列→数量(5)
        const dec = [false, false];
        for (const r of rows) {
            for (let j = 0; j < 2; j++) {
                if (r[j] && r[j].includes('.')) dec[j] = true;
            }
        }
        startKey = (dec[0] && !dec[1]) ? 6 : 5;
    } else if (ncols >= 2) {
        startKey = 2; // 多列从名称开始
    } else if (rows.every((r) => r.every((c) => c === '' || looksLikeNumber(c)))) {
        // 单列纯数字
        const hasDec = rows.some((r) => r[0] && r[0].includes('.'));
        if (hasDec) {
            startKey = 6; // 金额
        } else {
            // >4位(订单号/长编码) → 备注(7), 否则数量(5)
            let maxDigits = 0;
            for (const r of rows) {
                const d = String(r[0]).replace(/\D/g, '');
                if (d) maxDigits = Math.max(maxDigits, d.length);
            }
            startKey = maxDigits > 4 ? 7 : 5;
        }
    } else {
        startKey = 2; // 单列文字→名称
    }

    // 扩展行: 已有行填充, 超出才新增(与旧版一致)
    const need = startRow + rows.length;
    while (invRows.length < need) {
        invRows.push({ invoice_type: $('invType').value, tax_included: $('invTaxInc').value,
            buyer: '', tax_id: '', is_natural: '', qty: '', amount: '', remark: '',
            item_name: '', tax_code: '', unit: '', tax_rate: '' });
    }

    // 逐格写入
    let filled = 0;
    for (let i = 0; i < rows.length; i++) {
        const row = rows[i];
        for (let j = 0; j < row.length; j++) {
            const k = startKey + j;
            if (k >= keyCols.length) break;
            const idx = startRow + i;
            const val = row[j];
            if (!val) continue; // 跳过空值, 不覆盖已有
            const key = keyCols[k];
            if (key === 'is_natural') {
                invRows[idx][key] = val === '是' ? '是' : '';
            } else if (key === 'tax_id') {
                // 税号: 仅保留数字和英文(去掉 - 空格 等符号)
                invRows[idx][key] = val.replace(/[^0-9A-Za-z]/g, '');
            } else {
                invRows[idx][key] = val;
            }
            filled++;
        }
    }
    invRender();
    if (filled) alert(`✔ 已粘贴 ${rows.length} 行`);
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

function invDelRow(i) { invCollect(); invRows.splice(i, 1); if (!invRows.length) invRows.push(emptyRow()); invRender(); }
window.invDelRow = invDelRow;

function invDelSel() {
    const tr = document.querySelector('#invTable tbody tr.sel');
    if (!tr) { alert('请先点选一行'); return; }
    const idx = [...tr.parentElement.children].indexOf(tr);
    invRows.splice(idx, 1);
    if (!invRows.length) invRows.push(emptyRow());
    invRender();
}
window.invDelSel = invDelSel;

function invClear() {
    if (!confirm('清空全部发票？')) return;
    // 清空后保留一行空行(始终可直接粘贴, 不用手动新增)
    invRows = [{ invoice_type: $('invType').value, tax_included: $('invTaxInc').value,
        buyer: '', tax_id: '', is_natural: '', qty: '', amount: '', remark: '',
        item_name: '', tax_code: '', unit: '', tax_rate: '' }];
    invRender();
}
window.invClear = invClear;

async function pickTemplate() {
    const path = await SelectFile();
    if (path) $('invTpl').value = path;
}
window.pickTemplate = pickTemplate;

// 各字段选项(行级下拉, 从配置读取, 可扩展)
let fixedOptions = {
    invoice_type: ['普通发票', '增值税专用发票'],
    tax_included: ['是', '否'],
    item_name: [], tax_code: [], unit: [], tax_rate: [],
};

// 填充下拉选项
function fillSelect(id, opts, cur) {
    const sel = $(id);
    sel.innerHTML = opts.map((t) => `<option ${t === cur ? 'selected' : ''}>${t}</option>`).join('') +
        `<option value="" ${!cur ? 'selected' : ''}>—</option>`;
    if (cur) sel.value = cur;
}

// 加载固定内容(从本地配置)
async function invLoadFixed() {
    try {
        const cfg = await LoadFixedConfig();
        if (cfg) {
            // 选项(每个字段多选项)
            if (cfg.options) {
                for (const k of ['invoice_type', 'tax_included', 'item_name', 'tax_code', 'unit', 'tax_rate']) {
                    if (cfg.options[k] && cfg.options[k].length) fixedOptions[k] = cfg.options[k];
                }
            }
            fillSelect('invType', fixedOptions.invoice_type, cfg.invoice_type || fixedOptions.invoice_type[0]);
            fillSelect('invTaxInc', fixedOptions.tax_included, cfg.tax_included || '是');
            fillSelect('invItem', fixedOptions.item_name, cfg.item_name || '');
            fillSelect('invCode', fixedOptions.tax_code, cfg.tax_code || '');
            fillSelect('invUnit', fixedOptions.unit, cfg.unit || '');
            fillSelect('invRate', fixedOptions.tax_rate, cfg.tax_rate || '');
        }
    } catch (e) { /* 无配置 */ }
    // 自动查找本地发票模板(文件名含"批量开票-导入开票模板")
    try {
        const tpl = await FindInvoiceTemplate();
        if (tpl) $('invTpl').value = tpl;
    } catch (e) { /* 忽略 */ }
    invRender();
}
window.invLoadFixed = invLoadFixed;

// 应用选项(格式: 税率=0.01,0.03 单位=个,套 ...)
function invApplyOpts() {
    const txt = $('invOpts').value.trim();
    const keyMap = { '发票类型': 'invoice_type', '含税': 'tax_included', '项目名称': 'item_name',
        '税收编码': 'tax_code', '单位': 'unit', '税率': 'tax_rate',
        'invoice_type': 'invoice_type', 'tax_included': 'tax_included', 'item_name': 'item_name',
        'tax_code': 'tax_code', 'unit': 'unit', 'tax_rate': 'tax_rate' };
    if (txt) {
        // 支持 空格或换行分隔的 字段=值1,值2
        txt.split(/\s+/).forEach((pair) => {
            const eq = pair.indexOf('=');
            if (eq < 0) return;
            const k = keyMap[pair.slice(0, eq).trim()];
            const vals = pair.slice(eq + 1).split(/[,，]/).map((s) => s.trim()).filter((s) => s);
            if (k && vals.length) fixedOptions[k] = vals;
        });
    }
    // 刷新下拉(保留当前选中)
    invLoadFixed();
    alert('✅ 选项已应用(保存固定内容后永久生效)');
}
window.invApplyOpts = invApplyOpts;

// 保存固定内容到本地配置
async function invSaveFixed() {
    const cfg = {
        invoice_type: $('invType').value || '', tax_included: $('invTaxInc').value || '',
        item_name: $('invItem').value || '', tax_code: $('invCode').value || '',
        unit: $('invUnit').value || '', tax_rate: $('invRate').value || '',
        options: fixedOptions,
    };
    try {
        await SaveFixedConfig(cfg);
        alert('✅ 固定内容已保存到本地配置');
    } catch (e) {
        alert('保存失败: ' + e);
    }
}
window.invSaveFixed = invSaveFixed;

async function invGenerate() {
    invCollect();
    if (!invRows.length) { alert('请先添加发票行'); return; }
    const fixed = {
        invoice_type: $('invType').value, tax_included: $('invTaxInc').value,
        item_name: $('invItem').value.trim(), tax_code: $('invCode').value.trim(),
        unit: $('invUnit').value.trim(), tax_rate: $('invRate').value.trim(),
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
    const res = $('invResult');
    res.classList.remove('hidden');
    res.innerHTML = '⏳ 导入中…';
    try {
        const r = await ImportInvoiceDetail(path);
        if (!r || !r.rows || !r.rows.length) {
            res.innerHTML = `<div class="err">❌ 未解析到数据(请检查表头: 应含 抬头/税号/金额/数量/抬头类型)</div>`;
            return;
        }
        // 导入的行直接替换当前表格(从第一行开始)
        invRows = r.rows.map((x) => ({
            invoice_type: x.invoice_type || $('invType').value, tax_included: $('invTaxInc').value,
            buyer: x.buyer || '', tax_id: (x.tax_id || '').replace(/[^0-9A-Za-z]/g, ''), is_natural: x.is_natural || '',
            qty: x.qty || '', amount: x.amount || '', remark: x.remark || '',
            item_name: '', tax_code: '', unit: '', tax_rate: '',
        }));
        invRender();
        let msg = `✅ 已导入 ${r.imported} 条发票明细`;
        if (r.missing && r.missing.length) msg += `\n⚠️ 未识别列: ${r.missing.join(', ')}`;
        res.innerHTML = `<div class="ok">${msg.replace(/\n/g, '<br/>')}</div>`;
    } catch (e) {
        res.innerHTML = `<div class="err">❌ ${e}</div>`;
    }
}
window.invImportDetail = invImportDetail;

// 图片转JPEG功能模块
async function pickImgDir(kind) {
    const dir = await SelectDir();
    if (dir) {
        if (kind === 'src') {
            $('imgSrc').value = dir;
            if (!$('imgOut').value.trim()) $('imgOut').value = dir.replace(/[\\/]+$/, '') + '_jpg';
        } else {
            $('imgOut').value = dir;
        }
    }
}
window.pickImgDir = pickImgDir;

async function runImgConvert() {
    const src = $('imgSrc').value.trim();
    if (!src) { alert('请选择源文件夹'); return; }
    const out = $('imgOut').value.trim() || src.replace(/[\\/]+$/, '') + '_jpg';
    const q = parseInt($('imgQ').value) || 92;
    const res = $('imgResult');
    res.classList.remove('hidden');
    res.innerHTML = '⏳ 转换中…';
    try {
        const r = await RunImgConvert(src, out, q);
        res.innerHTML = `
            <div class="ok">✅ 完成: 总 <b>${r.total}</b> | 转换 <b>${r.converted}</b> | 复制 <b>${r.copied}</b> | 失败 <b>${r.failed}</b></div>
            <div class="muted">输出: ${out}</div>
            ${r.errors && r.errors.length ? `<div class="warn">${r.errors.slice(0, 10).join('<br/>')}</div>` : ''}`;
    } catch (e) {
        res.innerHTML = `<div class="err">❌ ${e}</div>`;
    }
}
window.runImgConvert = runImgConvert;

// ================= 邮箱发送模块 =================
let mailCfg = null;
let mailAttachments = [];

async function mailInit() {
    try {
        mailCfg = await LoadEmailConfig();
        if (mailCfg && mailCfg.sender_email) {
            $('mailCfgState').textContent = `已配置: ${mailCfg.sender_email}`;
            $('mailEmail').value = mailCfg.sender_email || '';
            $('mailCode').value = mailCfg.auth_code || '';
            $('mailName').value = mailCfg.sender_name || 'chibarin';
        }
    } catch (e) { /* 未配置 */ }
}
window.mailInit = mailInit;

async function mailPreset() {
    const p = $('mailProvider').value;
    const r = await EmailPreset(p);
    if (mailCfg) { mailCfg.smtp_host = r.host; mailCfg.smtp_port = r.port; }
}
window.mailPreset = mailPreset;

// 保存邮箱配置(页面表单)
async function mailSave() {
    const email = $('mailEmail').value.trim();
    const code = $('mailCode').value.trim();
    if (!email || !code) { alert('请填写发件邮箱和授权码'); return; }
    const p = $('mailProvider').value;
    const r = await EmailPreset(p);
    mailCfg = {
        sender_email: email, auth_code: code,
        smtp_host: r.host, smtp_port: r.port,
        sender_name: $('mailName').value.trim() || 'chibarin',
    };
    try {
        await SaveEmailConfig(mailCfg);
        $('mailCfgState').textContent = `已配置: ${email}`;
        alert('✅ 邮箱配置已保存');
    } catch (e) {
        alert('保存失败: ' + e);
    }
}
window.mailSave = mailSave;

function mailClearCfg() {
    mailCfg = null;
    $('mailEmail').value = ''; $('mailCode').value = '';
    $('mailCfgState').textContent = '未配置';
    try { SaveEmailConfig({ sender_email: '', auth_code: '', smtp_host: '', smtp_port: '', sender_name: '' }); } catch (e) { /* 忽略 */ }
    alert('已清除邮箱配置');
}
window.mailClearCfg = mailClearCfg;

async function mailPickAttach() {
    const path = await SelectFile();
    if (path) {
        mailAttachments.push(path);
        $('mailAttach').value = mailAttachments.map((p) => p.split(/[\\/]/).pop()).join('; ');
    }
}
window.mailPickAttach = mailPickAttach;

function mailClearAttach() {
    mailAttachments = [];
    $('mailAttach').value = '';
}
window.mailClearAttach = mailClearAttach;

async function mailSend() {
    if (!mailCfg || !mailCfg.sender_email) { alert('请先设置邮箱'); return; }
    const to = $('mailTo').value.trim();
    const subject = $('mailSubject').value.trim();
    const body = $('mailBody').value;
    if (!to) { alert('请填写收件人'); return; }
    const res = $('mailResult');
    res.classList.remove('hidden');
    res.innerHTML = '⏳ 发送中…';
    try {
        const r = await SendEmail(mailCfg, to, subject, body, mailAttachments);
        res.innerHTML = r.ok
            ? `<div class="ok">✅ ${r.msg}</div>`
            : `<div class="err">❌ ${r.msg}</div>`;
    } catch (e) {
        res.innerHTML = `<div class="err">❌ ${e}</div>`;
    }
}
window.mailSend = mailSend;

// 页面加载后: 渲染初始空行 + 加载邮箱配置 + 加载固定内容配置
document.addEventListener('DOMContentLoaded', () => { invRender(); mailInit(); invLoadFixed(); });