const API_BASE = '/api';
let historyStack = [];
let context = {};
let liveInterval = null;
let eventSource = null;

// Генерируем и сохраняем Web User ID (отрицательное число)
function getWebUserID() {
    let uid = localStorage.getItem('web_user_id');
    if (!uid) {
        uid = -1 * Math.floor(Math.random() * 1000000000);
        localStorage.setItem('web_user_id', uid);
    }
    return uid;
}

// Инициализация SSE (Server-Sent Events) для получения алертов
function initSSE() {
    if (eventSource) {
        eventSource.close();
    }
    
    // Передаем X-User-Id в заголовке нельзя напрямую через EventSource API браузера.
    // Поэтому в случае Vanilla JS и EventSource мы можем использовать куки или пробросить как query параметр.
    // Для нашего сервера (middleware authMiddleware) мы можем временно пропатчить его на чтение параметра (или использовать заголовки через fetch, но SSE не поддерживает заголовки).
    // Важно: чтобы Auth работал для SSE, нужно либо передавать userID в URL:
    eventSource = new EventSource(`${API_BASE}/notifications/stream?uid=${getWebUserID()}`);

    eventSource.addEventListener('alert', function(e) {
        try {
            const data = JSON.parse(e.data);
            showToast(data);
        } catch (err) {
            console.error("Ошибка парсинга SSE алерта", err);
        }
    });

    eventSource.onerror = function() {
        console.warn("SSE соединение разорвано. Попытка переподключения...");
    };
}

// Функция показа всплывающих уведомлений (Toast)
function showToast(data) {
    const toastContainer = document.getElementById('toast-container');
    if (!toastContainer) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${data.type}`; // 'alarm', 'emergency', 'resolved'
    
    let icon = 'fa-info-circle';
    if (data.type === 'emergency') icon = 'fa-exclamation-triangle';
    if (data.type === 'alarm') icon = 'fa-bell';
    if (data.type === 'resolved') icon = 'fa-check-circle';

    let html = `
        <div class="toast-header">
            <i class="fas ${icon}"></i> 
            <strong>${data.machine_id}</strong>
            <button onclick="this.parentElement.parentElement.remove()"><i class="fas fa-times"></i></button>
        </div>
        <div class="toast-body">
            ${data.message}
        </div>
    `;

    if (data.alarms && data.alarms.length > 0) {
        html += `<div class="toast-alarms"><ul>`;
        data.alarms.forEach(a => {
            html += `<li>[${a.error_code}] <b>${a.error_type_description}</b>: ${a.error_message}</li>`;
        });
        html += `</ul></div>`;
    }

    toast.innerHTML = html;
    toastContainer.appendChild(toast);

    // Удаляем автоматически через 10 секунд
    setTimeout(() => {
        if(toast.parentElement) toast.remove();
    }, 10000);
}

// Универсальный обработчик запросов к API
async function fetchAPI(url, options = {}) {
    try {
        if (!options.headers) options.headers = {};
        options.headers['X-User-Id'] = getWebUserID();

        const res = await fetch(url, options);
        const text = await res.text();
        let data = null;
        
        if (text) {
            try { data = JSON.parse(text); } catch (e) { }
        }

        if (!res.ok) {
            throw new Error((data && data.error) || `HTTP Error ${res.status}`);
        }
        return data || {};
    } catch (err) {
        console.error(err);
        alert("Ошибка: " + err.message);
        return null;
    }
}

// Навигация
function navTo(view, data = null) {
    const activeView = document.querySelector('.view.active');
    if (activeView && view !== 'home') {
        const currentViewId = activeView.id.replace('view-', '');
        if (historyStack.length === 0 || historyStack[historyStack.length - 1].view !== currentViewId) {
            historyStack.push({ view: currentViewId, data: { ...context } });
        }
    }
    
    document.querySelectorAll('.view').forEach(el => el.classList.remove('active'));
    const targetEl = document.getElementById(`view-${view}`);
    if (targetEl) targetEl.classList.add('active');
    
    document.getElementById('back-btn').classList.toggle('hidden', view === 'home');
    
    if (data) Object.assign(context, data);
    
    stopLiveMode();
    loadViewData(view);
}

function goBack() {
    if (historyStack.length > 0) {
        const prev = historyStack.pop();
        Object.assign(context, prev.data);
        const viewToLoad = prev.view;
        
        document.querySelectorAll('.view').forEach(el => el.classList.remove('active'));
        document.getElementById(`view-${viewToLoad}`).classList.add('active');
        document.getElementById('back-btn').classList.toggle('hidden', viewToLoad === 'home');
        
        stopLiveMode();
        loadViewData(viewToLoad);
    } else {
        navTo('home');
    }
}

// Рендер элементов списка
function renderList(elementId, items, htmlFunc, onClickFunc) {
    const el = document.getElementById(elementId);
    el.innerHTML = '';
    if (!Array.isArray(items) || items.length === 0) {
        el.innerHTML = '<div class="text-center text-muted" style="padding: 20px; background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.08); border-radius: 12px;">Список пуст</div>';
        return;
    }
    items.forEach(item => {
        const div = document.createElement('div');
        div.className = 'list-item';
        div.innerHTML = `<div>${htmlFunc(item)}</div><i class="fas fa-chevron-right text-muted"></i>`;
        div.onclick = () => onClickFunc(item);
        el.appendChild(div);
    });
}

// Загрузка данных
async function loadViewData(view) {
    if (view === 'home') {
        const res = await fetchAPI(`${API_BASE}/profile`);
        if (res) {
            document.getElementById('stat-targets').innerText = res.targets_count || 0;
            document.getElementById('stat-services').innerText = res.services_count || 0;
        }
    } 
    else if (view === 'targets') {
        const targets = await fetchAPI(`${API_BASE}/targets`);
        renderList('targets-list', targets, t => `<div class="item-title"><i class="fas fa-database text-yellow"></i> ${t.Name}</div><div class="item-subtitle">${t.Broker} | ${t.Topic}</div>`, t => navTo('target-detail', { currentTarget: t }));
    }
    else if (view === 'target-detail') {
        const t = await fetchAPI(`${API_BASE}/targets/${context.currentTarget.ID}`);
        if (t) {
            context.currentTarget = t;
            document.getElementById('target-info').innerHTML = `<strong>${t.Name}</strong><br><small class="text-muted">${t.Broker}<br>${t.Topic}</small>`;
            const keys = [{ID: 0, Key: "📂 Без фильтра (По умолчанию)"}, ...(t.Keys || [])];
            renderList('keys-list', keys, k => `<div class="item-title"><i class="fas fa-key ${k.ID===0?'text-muted':'text-yellow'}"></i> ${k.Key}</div>`, k => navTo('key-detail', { keyId: k.ID, keyName: k.Key }));
        }
    }
    else if (view === 'key-detail') {
        document.getElementById('live-key-name').innerText = context.keyName;
        document.getElementById('btn-delete-key').classList.toggle('hidden', context.keyId === 0);
    }
    else if (view === 'services') {
        const svcs = await fetchAPI(`${API_BASE}/services`);
        renderList('services-list', svcs, s => `<div class="item-title"><i class="fas fa-server text-yellow"></i> ${s.Name}</div><div class="item-subtitle">${s.BaseURL}</div>`, s => navTo('service-detail', { currentService: s }));
    }
    else if (view === 'service-detail') {
        const s = await fetchAPI(`${API_BASE}/services/${context.currentService.ID}`);
        if (s) {
            context.currentService = s;
            document.getElementById('service-info').innerHTML = `<strong>${s.Name}</strong><br><small class="text-muted">${s.BaseURL}</small>`;
            const machines = await fetchAPI(`${API_BASE}/services/${s.ID}/machines`);
            renderList('machines-list', machines, m => {
                let statusClass = m.status === 'connected' ? (m.mode==='polling'?'status-poll':'status-ok') : 'status-err';
                return `<div class="item-title"><span class="status-indicator ${statusClass}"></span> ${m.endpoint}</div><div class="item-subtitle">${m.model} (${m.series})</div>`;
            }, m => navTo('machine-detail', { machId: m.id }));
        }
    }
    else if (view === 'machine-detail') {
        const m = await fetchAPI(`${API_BASE}/services/${context.currentService.ID}/machines/${context.machId}`);
        if (m) {
            let statusIcon = m.status === 'connected' ? (m.mode==='polling'?'🔄':'🟢') : '🔴';
            document.getElementById('machine-info').innerHTML = `<strong>${statusIcon} ${m.endpoint}</strong><br><small class="text-muted">ID: ${m.id}<br>Model: ${m.model} (${m.series})<br>Timeout: ${m.timeout}ms</small>`;
            const btn = document.getElementById('btn-poll-toggle');
            if (m.mode === 'polling') {
                btn.innerHTML = '<i class="fas fa-stop-circle text-danger"></i> Остановить опрос';
                btn.onclick = () => stopPoll();
            } else {
                btn.innerHTML = '<i class="fas fa-play-circle text-yellow"></i> Запустить опрос';
                btn.onclick = () => startPoll();
            }
            document.getElementById('gcode-display').classList.add('hidden');
            document.getElementById('gcode-display').innerText = '';
        }
    }
}

// ---------------------------
// Действия Kafka
// ---------------------------
function openTargetForm(isEdit = false) {
    context.isEdit = isEdit;
    if (!isEdit) {
        document.getElementById('target-form-title').innerText = 'Добавить Target';
        document.getElementById('target-name').value = '';
        document.getElementById('target-broker').value = '';
        document.getElementById('target-topic').value = '';
    } else {
        document.getElementById('target-form-title').innerText = 'Изменить Target';
        document.getElementById('target-name').value = context.currentTarget.Name;
        document.getElementById('target-broker').value = context.currentTarget.Broker;
        document.getElementById('target-topic').value = context.currentTarget.Topic;
    }
    navTo('target-form');
}

async function saveTarget(e) {
    e.preventDefault();
    const payload = {
        name: document.getElementById('target-name').value,
        broker: document.getElementById('target-broker').value,
        topic: document.getElementById('target-topic').value
    };
    const url = context.isEdit ? `${API_BASE}/targets/${context.currentTarget.ID}` : `${API_BASE}/targets`;
    const res = await fetchAPI(url, { method: context.isEdit ? 'PUT' : 'POST', body: JSON.stringify(payload) });
    if (res) goBack();
}

async function deleteTarget() {
    if(confirm('Точно удалить этот Target?')) {
        const res = await fetchAPI(`${API_BASE}/targets/${context.currentTarget.ID}`, { method: 'DELETE' });
        if(res) {
            historyStack.pop(); 
            navTo('targets');
        }
    }
}

async function createKey(e) {
    e.preventDefault();
    const res = await fetchAPI(`${API_BASE}/targets/${context.currentTarget.ID}/keys`, {
        method: 'POST', body: JSON.stringify({ key: document.getElementById('key-value').value })
    });
    if (res) { e.target.reset(); goBack(); }
}

async function deleteKey() {
    if(confirm('Точно удалить ключ?')) {
        const res = await fetchAPI(`${API_BASE}/keys/${context.keyId}`, { method: 'DELETE' });
        if(res) {
            historyStack.pop(); 
            navTo('target-detail');
        }
    }
}

// ---------------------------
// Консоль Kafka (Live / Single)
// ---------------------------
async function fetchMessageToConsole() {
    const cons = document.getElementById('live-console-display');
    const indicator = document.getElementById('console-status');
    
    try {
        const res = await fetch(`/api/monitoring/message?targetId=${context.currentTarget.ID}&keyId=${context.keyId}`, {
            headers: { 'X-User-Id': getWebUserID() }
        });
        const json = await res.json();
        
        if (!res.ok) {
            cons.innerText = `❌ ${json.error || 'Ошибка'}`;
            indicator.innerHTML = '<i class="fas fa-circle text-danger"></i> Ошибка';
        } else {
            cons.innerText = JSON.stringify(json.data, null, 2);
            indicator.innerHTML = `<i class="fas fa-circle text-yellow"></i> Обновлено: ${new Date().toLocaleTimeString()}`;
        }
    } catch(e) {
        cons.innerText = "❌ Ошибка сети";
        indicator.innerHTML = '<i class="fas fa-circle text-danger"></i> Нет связи';
    }
}

function openLiveConsole(isLiveMode) {
    navTo('live-console');
    
    document.getElementById('console-title').innerText = isLiveMode ? '🔴 LIVE' : '📨 MSG';
    document.getElementById('console-status').innerHTML = '<i class="fas fa-circle"></i> Запрос...';
    
    if (document.getElementById('live-console-display').innerText.trim() === '') {
        document.getElementById('live-console-display').innerText = '';
    }

    fetchMessageToConsole();

    if (isLiveMode) {
        liveInterval = setInterval(fetchMessageToConsole, 1500);
    }
}

function stopLiveMode() {
    if (liveInterval) clearInterval(liveInterval);
    liveInterval = null;
    document.getElementById('live-console-display').innerText = '';
}

// ---------------------------
// Действия Services
// ---------------------------
function openServiceForm(isEdit = false) {
    context.isEdit = isEdit;
    if (!isEdit) {
        document.getElementById('service-form-title').innerText = 'Добавить Service';
        document.getElementById('svc-name').value = '';
        document.getElementById('svc-url').value = '';
        document.getElementById('svc-key').value = '';
    } else {
        document.getElementById('service-form-title').innerText = 'Изменить Service';
        document.getElementById('svc-name').value = context.currentService.Name;
        document.getElementById('svc-url').value = context.currentService.BaseURL;
        document.getElementById('svc-key').value = context.currentService.APIKey;
    }
    navTo('service-form');
}

async function saveService(e) {
    e.preventDefault();
    const payload = {
        name: document.getElementById('svc-name').value,
        baseUrl: document.getElementById('svc-url').value,
        apiKey: document.getElementById('svc-key').value
    };
    const url = context.isEdit ? `${API_BASE}/services/${context.currentService.ID}` : `${API_BASE}/services`;
    const res = await fetchAPI(url, { method: context.isEdit ? 'PUT' : 'POST', body: JSON.stringify(payload) });
    if (res) goBack();
}

async function deleteService() {
    if(confirm('Точно удалить этот Service?')) {
        const res = await fetchAPI(`${API_BASE}/services/${context.currentService.ID}`, { method: 'DELETE' });
        if(res) {
            historyStack.pop(); 
            navTo('services');
        }
    }
}

// ---------------------------
// Действия Machines
// ---------------------------
async function createMachine(e) {
    e.preventDefault();
    const payload = {
        endpoint: document.getElementById('mach-endpoint').value,
        timeout: parseInt(document.getElementById('mach-timeout').value),
        model: document.getElementById('mach-model').value,
        series: document.getElementById('mach-series').value
    };
    const res = await fetchAPI(`${API_BASE}/services/${context.currentService.ID}/machines`, { method: 'POST', body: JSON.stringify(payload) });
    if (res) { e.target.reset(); goBack(); }
}

async function deleteMachine() {
    if(confirm('Удалить подключение?')) {
        const res = await fetchAPI(`${API_BASE}/services/${context.currentService.ID}/machines/${context.machId}`, { method: 'DELETE' });
        if(res) {
            historyStack.pop();
            navTo('service-detail');
        }
    }
}

function startPoll() {
    openModal('Интервал опроса (мс)', true, async (val) => {
        const res = await fetchAPI(`${API_BASE}/services/${context.currentService.ID}/machines/${context.machId}/poll`, {
            method: 'POST', body: JSON.stringify({ interval: parseInt(val) })
        });
        if (res) loadViewData('machine-detail');
    });
}

async function stopPoll() {
    const res = await fetchAPI(`${API_BASE}/services/${context.currentService.ID}/machines/${context.machId}/poll`, { method: 'DELETE' });
    if (res) loadViewData('machine-detail');
}

async function downloadProgram() {
    const btn = document.getElementById('btn-gcode');
    const display = document.getElementById('gcode-display');
    
    btn.disabled = true;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Загрузка с ЧПУ...';
    display.classList.remove('hidden');
    display.innerText = '⏳ Подключение к станку и скачивание...';

    try {
        const res = await fetch(`${API_BASE}/services/${context.currentService.ID}/machines/${context.machId}/program`, {
            headers: { 'X-User-Id': getWebUserID() }
        });
        
        if (!res.ok) {
            const data = await res.json();
            throw new Error(data.error || 'Ошибка скачивания');
        }
        
        const text = await res.text();
        display.innerText = text;
        
        const blob = new Blob([text], { type: 'text/plain' });
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = `GCODE_${context.machId}.NC`;
        a.click();
    } catch(e) {
        display.innerText = "❌ " + e.message;
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<i class="fas fa-file-code"></i> Запросить G-CODE';
    }
}

// ---------------------------
// Модальные окна
// ---------------------------
function openModal(title, showInput, onConfirm) {
    document.getElementById('modal-title').innerText = title;
    const input = document.getElementById('modal-input');
    input.style.display = showInput ? 'block' : 'none';
    input.value = '5000';
    document.getElementById('modal').classList.remove('hidden');
    document.getElementById('modal-confirm').onclick = () => {
        onConfirm(input.value);
        closeModal();
    };
}
function closeModal() { document.getElementById('modal').classList.add('hidden'); }

window.onload = () => {
    initSSE();
    navTo('home');
};