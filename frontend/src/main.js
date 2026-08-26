// Wails 前端逻辑
import './style.css';

// Wails 注入的绑定 (v2 自动生成于 frontend/wailsjs/go/main/App.js)
import { RunMatch } from '../wailsjs/go/main/App';

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
        // Wails 运行时文件对话框 (v2: runtime.OpenFileDialog)
        const { OpenFileDialog } = await import('../wailsjs/runtime/runtime.js');
        const opts = { filters: [{ displayName: 'Excel', pattern: '*.xlsx;*.xlsm;*.csv' }] };
        const path = await OpenFileDialog(opts);
        if (path) $(kind === 'src' ? 'srcPath' : 'tgtPath').value = path;
    } catch (e) {
        console.error(e);
        alert('文件对话框不可用(需在Wails环境运行)');
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