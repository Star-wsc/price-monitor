const API_BASE = '/api';
let allProducts = [];
let historyChart = null;

// Toast
function showToast(msg, type = 'info') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = msg;
    container.appendChild(toast);
    setTimeout(() => toast.remove(), 3500);
}

// 加载
async function loadProducts() {
    const grid = document.getElementById('productGrid');
    grid.innerHTML = '<p class="loading">加载中...</p>';
    try {
        const res = await fetch(`${API_BASE}/products`);
        const json = await res.json();
        if (json.code !== 0) { showToast(json.msg, 'error'); return; }
        allProducts = json.data || [];
        updateStats();
        renderProducts();
        document.getElementById('lastUpdate').textContent =
            `数据更新时间：${new Date().toLocaleString('zh-CN')}`;
    } catch (e) { showToast('加载失败: ' + e.message, 'error'); }
}

function updateStats() {
    const total = allProducts.length;
    const alert = allProducts.filter(p => p.target_price > 0 && p.current_price <= p.target_price).length;
    const today = new Date().toDateString();
    const updated = allProducts.filter(p => p.last_check && new Date(p.last_check).toDateString() === today).length;

    // 计算距目标总额
    let savings = 0;
    allProducts.forEach(p => {
        if (p.target_price > 0) {
            savings += Math.max(0, p.current_price - p.target_price);
        }
    });

    document.getElementById('stat-total').textContent = total;
    document.getElementById('stat-alert').textContent = alert;
    document.getElementById('stat-today').textContent = updated;
    document.getElementById('stat-savings').textContent = savings > 0 ? `¥${savings.toFixed(0)}` : '0';
    document.getElementById('mini-total').textContent = total;
    document.getElementById('mini-alert').textContent = alert;
}

function renderProducts() {
    const grid = document.getElementById('productGrid');
    const search = document.getElementById('searchInput').value.toLowerCase();
    const checkedSources = Array.from(document.querySelectorAll('.filter-tag input:checked')).map(c => c.value);

    const filtered = allProducts.filter(p => {
        const matchSearch = p.name.toLowerCase().includes(search);
        const matchSource = checkedSources.includes(p.source);
        return matchSearch && matchSource;
    });

    if (filtered.length === 0) {
        grid.innerHTML = allProducts.length === 0
            ? `<div class="empty-state"><div class="empty-icon"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#d1d5db" stroke-width="1.5"><path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/></svg></div><p>暂无监控商品</p><small>点击左侧「添加商品」开始监控价格</small></div>`
            : `<div class="empty-state"><div class="empty-icon"><svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#d1d5db" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></div><p>没有匹配结果</p></div>`;
        return;
    }

    grid.innerHTML = filtered.map(p => renderCard(p)).join('');
    bindCardEvents();
}

function renderCard(p) {
    const isReached = p.target_price > 0 && p.current_price <= p.target_price;
    const lastCheck = p.last_check ? new Date(p.last_check).toLocaleString('zh-CN', {month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'}) : '从未';
    const sourceLabel = {jd:'京东', taobao:'淘宝', tmall:'天猫', generic:'通用'}[p.source] || p.source;
    const diff = p.target_price > 0 ? (p.current_price - p.target_price).toFixed(2) : null;

    return `
    <div class="product-card" data-id="${p.id}" data-source="${p.source}">
        <div class="card-header"></div>
        <div class="card-body">
            <div class="card-top">
                <div class="card-img-wrap">
                    <img src="${p.image_url || '/static/img/placeholder.png'}" class="card-img" onerror="this.src='/static/img/placeholder.png'">
                </div>
                <div class="card-meta">
                    <span class="card-source source-${p.source}">${sourceLabel}</span>
                    <div class="card-name" title="${p.name}">${p.name}</div>
                </div>
            </div>
            <div class="price-section">
                <div class="price-main">
                    <span class="price-current">¥${p.current_price.toFixed(2)}</span>
                    ${isReached ? '<span class="badge-down">降至目标价</span>' : ''}
                </div>
                <div class="price-row">
                    ${p.target_price > 0
                        ? `<span class="price-target ${isReached ? 'reached' : ''}">目标: ¥${p.target_price} ${diff > 0 ? `(还差¥${diff})` : '(已达成)'}</span>`
                        : '<span class="price-target">未设目标价</span>'
                    }
                </div>
            </div>
            <div class="card-footer">
                <button class="btn btn-card-refresh" data-id="${p.id}">刷新</button>
                <button class="btn btn-card-history" data-id="${p.id}">历史</button>
                <button class="btn btn-card-delete" data-id="${p.id}">删除</button>
            </div>
        </div>
    </div>`;
}

function bindCardEvents() {
    document.querySelectorAll('.btn-card-refresh').forEach(btn => {
        btn.addEventListener('click', () => refreshProduct(btn.dataset.id));
    });
    document.querySelectorAll('.btn-card-history').forEach(btn => {
        btn.addEventListener('click', () => showHistory(btn.dataset.id));
    });
    document.querySelectorAll('.btn-card-delete').forEach(btn => {
        btn.addEventListener('click', () => deleteProduct(btn.dataset.id));
    });
}

// 添加商品
async function addProduct() {
    const url = document.getElementById('productUrl').value.trim();
    const targetPrice = document.getElementById('targetPrice').value.trim();
    if (!url) { showToast('请输入商品链接', 'error'); return; }

    const btn = document.getElementById('confirmAdd');
    btn.disabled = true;
    btn.textContent = '添加中...';

    try {
        const formData = new FormData();
        formData.append('url', url);
        if (targetPrice) formData.append('target_price', targetPrice);

        const res = await fetch(`${API_BASE}/products`, { method: 'POST', body: formData });
        const json = await res.json();
        if (json.code === 0) {
            closeAddModal();
            document.getElementById('productUrl').value = '';
            document.getElementById('targetPrice').value = '';
            showToast('添加成功', 'success');
            loadProducts();
        } else {
            showToast(json.msg, 'error');
        }
    } catch (e) { showToast('添加失败: ' + e.message, 'error'); }
    finally { btn.disabled = false; btn.textContent = '确认添加'; }
}

async function deleteProduct(id) {
    if (!confirm('确定删除此商品？')) return;
    try {
        const res = await fetch(`${API_BASE}/products/${id}`, { method: 'DELETE' });
        const json = await res.json();
        if (json.code === 0) { showToast('删除成功', 'success'); loadProducts(); }
        else { showToast(json.msg, 'error'); }
    } catch (e) { showToast('删除失败: ' + e.message, 'error'); }
}

async function refreshProduct(id) {
    const btn = document.querySelector(`.btn-card-refresh[data-id="${id}"]`);
    if (btn) { btn.disabled = true; btn.textContent = '刷新中...'; }
    try {
        const res = await fetch(`${API_BASE}/products/${id}/refresh`, { method: 'POST' });
        const json = await res.json();
        if (json.code === 0) {
            showToast(`刷新成功: ¥${json.data.current_price.toFixed(2)}`, 'success');
            loadProducts();
        } else { showToast(json.msg, 'error'); }
    } catch (e) { showToast('刷新失败: ' + e.message, 'error'); }
    if (btn) { btn.disabled = false; btn.textContent = '刷新'; }
}

async function refreshAll() {
    const btn = document.getElementById('refreshAll');
    btn.disabled = true;
    const orig = btn.innerHTML;
    btn.innerHTML = '<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="animation:spin 1s linear infinite"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg> 刷新中...';

    try {
        const res = await fetch(`${API_BASE}/products`);
        const json = await res.json();
        const products = json.data || [];
        let ok = 0, fail = 0;
        for (const p of products) {
            const r = await fetch(`${API_BASE}/products/${p.id}/refresh`, { method: 'POST' });
            const j = await r.json();
            if (j.code === 0) ok++; else fail++;
        }
        showToast(`完成: ${ok}成功 ${fail}失败`, ok > 0 ? 'success' : 'error');
        loadProducts();
    } catch (e) { showToast('刷新失败: ' + e.message, 'error'); }
    btn.disabled = false;
    btn.innerHTML = orig;
}

// 历史弹窗
async function showHistory(id) {
    const modal = document.getElementById('historyModal');
    modal.classList.add('show');

    try {
        const res = await fetch(`${API_BASE}/products/${id}/history?days=30`);
        const json = await res.json();
        if (json.code !== 0) { showToast(json.msg, 'error'); return; }

        const history = json.data || [];
        const product = allProducts.find(p => p.id == id);
        document.getElementById('historyTitle').textContent = product?.name?.substring(0, 40) || '价格历史';

        if (history.length === 0) {
            document.getElementById('historyKpis').innerHTML = '<div style="grid-column:1/-1;text-align:center;color:var(--text-3);padding:20px">暂无历史数据</div>';
            document.getElementById('historyChart').innerHTML = '';
            return;
        }

        const prices = history.map(h => h.price);
        const min = Math.min(...prices);
        const max = Math.max(...prices);
        const avg = (prices.reduce((a, b) => a + b, 0) / prices.length).toFixed(2);
        const latest = prices[prices.length - 1];

        document.getElementById('historyKpis').innerHTML = `
            <div class="hkpi"><div class="hkpi-val" style="color:var(--success)">¥${min.toFixed(2)}</div><div class="hkpi-label">最低价</div></div>
            <div class="hkpi"><div class="hkpi-val" style="color:var(--danger)">¥${max.toFixed(2)}</div><div class="hkpi-label">最高价</div></div>
            <div class="hkpi"><div class="hkpi-val">¥${avg}</div><div class="hkpi-label">平均价</div></div>
            <div class="hkpi"><div class="hkpi-val">¥${latest.toFixed(2)}</div><div class="hkpi-label">当前价</div></div>`;

        if (!historyChart) historyChart = echarts.init(document.getElementById('historyChart'));

        const dates = history.map(h => new Date(h.checked_at).toLocaleString('zh-CN', {month:'2-digit',day:'2-digit',hour:'2-digit'}));
        const priceData = history.map(h => ({ value: h.price, ts: h.checked_at }));

        historyChart.setOption({
            grid: { left: 55, right: 20, top: 10, bottom: 35 },
            tooltip: {
                trigger: 'axis',
                formatter: params => {
                    const d = new Date(priceData[params[0].dataIndex].ts).toLocaleString('zh-CN');
                    return `<div style="font-size:12px;color:#666">${d}</div><b style="font-size:14px;color:#f43f5e">¥${params[0].value}</b>`;
                }
            },
            xAxis: {
                type: 'category', data: dates,
                axisLabel: { fontSize: 10, color: '#9090a0', rotate: 30 },
                axisLine: { lineStyle: { color: '#e8e3ff' } },
                axisTick: { show: false }
            },
            yAxis: {
                type: 'value',
                axisLabel: { formatter: '¥{value}', fontSize: 11, color: '#9090a0' },
                splitLine: { lineStyle: { color: '#f0effe' } },
                axisLine: { show: false },
                axisTick: { show: false }
            },
            series: [{
                type: 'line', data: prices,
                smooth: 0.4,
                symbol: 'circle', symbolSize: 5,
                lineStyle: { color: '#8b5cf6', width: 2.5 },
                itemStyle: { color: '#8b5cf6', borderWidth: 2, borderColor: '#fff' },
                areaStyle: {
                    color: {
                        type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(139,92,246,0.15)' },
                            { offset: 1, color: 'rgba(139,92,246,0)' }
                        ]
                    }
                }
            }]
        });
        window.addEventListener('resize', () => historyChart?.resize());
    } catch (e) { showToast('获取历史失败: ' + e.message, 'error'); }
}

// 弹窗
function openAddModal() {
    document.getElementById('addModal').classList.add('show');
    document.getElementById('productUrl').focus();
}
function closeAddModal() { document.getElementById('addModal').classList.remove('show'); }
function closeHistoryModal() { document.getElementById('historyModal').classList.remove('show'); }

// 事件
document.getElementById('addBtn').addEventListener('click', openAddModal);
document.getElementById('confirmAdd').addEventListener('click', addProduct);
document.getElementById('cancelAdd').addEventListener('click', closeAddModal);
document.getElementById('closeAddModal').addEventListener('click', closeAddModal);
document.getElementById('closeHistory').addEventListener('click', closeHistoryModal);
document.getElementById('refreshAll').addEventListener('click', refreshAll);
document.getElementById('searchInput').addEventListener('input', renderProducts);

document.querySelectorAll('.filter-tag input').forEach(c => {
    c.addEventListener('change', () => {
        c.closest('.filter-tag').classList.toggle('active', c.checked);
        renderProducts();
    });
});

document.getElementById('addModal').addEventListener('click', e => { if (e.target.id === 'addModal') closeAddModal(); });
document.getElementById('historyModal').addEventListener('click', e => { if (e.target.id === 'historyModal') closeHistoryModal(); });
document.getElementById('productUrl').addEventListener('keydown', e => { if (e.key === 'Enter') addProduct(); });

// 加载时设置filter-tag active状态
document.querySelectorAll('.filter-tag').forEach(tag => {
    const cb = tag.querySelector('input');
    if (cb && cb.checked) tag.classList.add('active');
});

// 启动
loadProducts();
