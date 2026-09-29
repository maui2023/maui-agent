/**
 * MIQA WebUI Controller
 * Connects SSE stream, draws real-time CPU charts, handles terminal output,
 * model switching, and prompt executions.
 */

class MiqaApp {
  constructor() {
    this.activeModel = 'gemini-3.8-flash-high';
    this.models = [];
    this.cpuHistory = new Array(30).fill(0.5);
    this.autoScroll = true;
    this.isExecuting = false;

    this.initDOM();
    this.initClock();
    this.initChart();
    this.initSSE();
    this.fetchInitialData();
    this.bindEvents();
  }

  initDOM() {
    this.headerModel = document.getElementById('header-model-val');
    this.headerPath = document.getElementById('header-path-val');
    this.termHeaderModel = document.getElementById('term-header-model');
    this.terminalScreen = document.getElementById('terminal-screen');
    this.terminalLogs = document.getElementById('terminal-logs');
    this.termStatusText = document.getElementById('term-status-text');
    this.termStatusDot = document.getElementById('term-status-dot');
    this.termCursorTime = document.getElementById('term-cursor-time');
    this.termCursorTag = document.getElementById('term-cursor-tag');
    this.termCursorText = document.getElementById('term-cursor-text');
    this.officeStatusLabel = document.getElementById('office-status-label');
    this.officePulseDot = document.getElementById('office-pulse-dot');
    this.cpuChartCanvas = document.getElementById('cpu-chart');
    this.cpuChartCtx = this.cpuChartCanvas ? this.cpuChartCanvas.getContext('2d') : null;
  }

  initClock() {
    const clockEl = document.getElementById('live-clock');
    const updateTime = () => {
      const now = new Date();
      const timeStr = now.toTimeString().split(' ')[0];
      if (clockEl) clockEl.textContent = timeStr;
      if (this.termCursorTime) this.termCursorTime.textContent = `[${timeStr}]`;
    };
    updateTime();
    setInterval(updateTime, 1000);
  }

  bindEvents() {
    // Terminal Controls
    const btnScan = document.getElementById('btn-toggle-scanlines');
    if (btnScan) {
      btnScan.addEventListener('click', () => {
        this.terminalScreen.classList.toggle('scanlines');
        this.showToast('Kesan CRT Scanlines diubah');
      });
    }

    const btnClear = document.getElementById('btn-clear-term');
    if (btnClear) {
      btnClear.addEventListener('click', () => {
        if (this.terminalLogs) this.terminalLogs.innerHTML = '';
        this.showToast('Terminal dibersihkan');
      });
    }

    const chkAuto = document.getElementById('chk-autoscroll');
    if (chkAuto) {
      chkAuto.addEventListener('change', (e) => {
        this.autoScroll = e.target.checked;
      });
    }

    // Clickable Path Pill to change workspace path
    const pillPath = document.getElementById('pill-active-path');
    if (pillPath) {
      pillPath.style.cursor = 'pointer';
      pillPath.addEventListener('click', async () => {
        const current = this.headerPath ? this.headerPath.textContent.trim() : '';
        const newPath = prompt('Tukar Laluan Pengekodan (Workspace Path):', current);
        if (newPath && newPath.trim() && newPath.trim() !== current) {
          try {
            const resp = await fetch('/api/set-path', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ path: newPath.trim() })
            });
            if (resp.ok) {
              const data = await resp.json();
              if (this.headerPath) this.headerPath.textContent = data.path;
              this.showToast(`✅ Laluan kod dikemaskini: ${data.path}`);
            } else {
              const errTxt = await resp.text();
              alert(`Ralat menukar laluan: ${errTxt}`);
            }
          } catch (e) {
            alert('Gagal menyambung ke pelayan: ' + e.message);
          }
        }
      });
    }

    // Allow changing model globally from pixel_office.js
    window.selectModelFromUI = (modelId) => {
      this.changeActiveModel(modelId);
    };
  }

  async fetchInitialData() {
    // 1. Fetch & display logs immediately so terminal is never blank
    try {
      const resLogs = await fetch('/api/logs').then(r => r.json());
      if (Array.isArray(resLogs) && resLogs.length > 0) {
        if (this.terminalLogs) {
          this.terminalLogs.innerHTML = '';
          resLogs.forEach(entry => this.appendTerminalLog(entry));
        }
      }
    } catch (err) {
      console.warn('Gagal memuatkan log awal:', err);
    }

    // 2. Fetch system status
    try {
      const resStatus = await fetch('/api/status').then(r => r.json());
      if (resStatus) {
        this.activeModel = resStatus.active_model || this.activeModel;
        if (this.headerModel) this.headerModel.textContent = this.activeModel;
        if (this.termHeaderModel) this.termHeaderModel.textContent = this.activeModel;
        if (this.headerPath) this.headerPath.textContent = resStatus.coding_path || '';

        if (resStatus.is_executing) {
          this.setExecutingState(resStatus.current_task || 'Melaksanakan arahan...', 'Telegram');
        } else {
          this.setIdleState();
        }

        if (resStatus.stats) {
          this.updateSystemMetrics(resStatus.stats);
        }
      }
    } catch (err) {
      console.warn('Gagal memuatkan status awal:', err);
    }

    // 3. Fetch model list
    try {
      const resModels = await fetch('/api/models').then(r => r.json());
      if (Array.isArray(resModels)) {
        this.models = resModels;
        if (window.pixelOffice) {
          window.pixelOffice.updateFromBackend(this.models, this.activeModel, this.isExecuting, '');
        }
      }
    } catch (err) {
      console.warn('Gagal memuatkan senarai model:', err);
    }
  }

  setExecutingState(task, source) {
    this.isExecuting = true;
    const src = source || 'Telegram';
    if (this.termStatusText) {
      this.termStatusText.textContent = `🔴 MELAKSANAKAN ARAHAN (${src}): "${task}"`;
      this.termStatusText.classList.add('text-working');
    }
    if (this.termStatusDot) {
      this.termStatusDot.className = 'live-status-dot working-dot';
    }
    if (this.officeStatusLabel) {
      this.officeStatusLabel.textContent = `🔴 Ejen Sedang Bekerja (${this.activeModel})`;
    }
    if (this.officePulseDot) {
      this.officePulseDot.className = 'pulse-indicator pulse-red';
    }
    if (this.termCursorTag) {
      this.termCursorTag.textContent = `${this.activeModel}: active`;
    }
    if (this.termCursorText) {
      this.termCursorText.textContent = `Sedang memproses arahan: "${task}"`;
    }
    if (window.pixelOffice) {
      window.pixelOffice.updateFromBackend(this.models, this.activeModel, true, task);
    }
  }

  setIdleState() {
    this.isExecuting = false;
    if (this.termStatusText) {
      this.termStatusText.textContent = '🟢 MENUNGGU ARAHAN DARI TELEGRAM (@miqa_agy_bot) ATAU CLI...';
      this.termStatusText.classList.remove('text-working');
    }
    if (this.termStatusDot) {
      this.termStatusDot.className = 'live-status-dot idle-dot';
    }
    if (this.officeStatusLabel) {
      this.officeStatusLabel.textContent = '🟢 Sedia Menunggu Arahan';
    }
    if (this.officePulseDot) {
      this.officePulseDot.className = 'pulse-indicator';
    }
    if (this.termCursorTag) {
      this.termCursorTag.textContent = `${this.activeModel}: live`;
    }
    if (this.termCursorText) {
      this.termCursorText.textContent = 'Menunggu arahan dari Telegram (@miqa_agy_bot)...';
    }
    if (window.pixelOffice) {
      window.pixelOffice.updateFromBackend(this.models, this.activeModel, false, '');
    }
  }

  initSSE() {
    const sse = new EventSource('/api/events');

    sse.addEventListener('connected', () => {
      console.log('SSE Terhubung');
    });

    sse.addEventListener('stats_tick', (e) => {
      try {
        const stats = JSON.parse(e.data);
        this.updateSystemMetrics(stats);
      } catch (err) {}
    });

    sse.addEventListener('log', (e) => {
      try {
        const entry = JSON.parse(e.data);
        this.appendTerminalLog(entry);
      } catch (err) {}
    });

    sse.addEventListener('agent_started', (e) => {
      const data = JSON.parse(e.data);
      const task = data.task || 'Melaksanakan arahan...';
      const src = data.source || 'Telegram';
      this.setExecutingState(task, src);
      this.showToast(`⚡ Arahan diterima dari ${src} untuk model ${data.model || this.activeModel}`);
    });

    sse.addEventListener('agent_finished', (e) => {
      const data = JSON.parse(e.data);
      this.setIdleState();
      this.showToast(`✅ Tugasan selesai (${data.duration || ''})`);
    });

    sse.addEventListener('models_updated', (e) => {
      try {
        const states = JSON.parse(e.data);
        this.models = Object.values(states);
        if (window.pixelOffice) {
          window.pixelOffice.updateFromBackend(this.models, this.activeModel, this.isExecuting, '');
        }
      } catch (err) {}
    });

    sse.addEventListener('model_changed', (e) => {
      const data = JSON.parse(e.data);
      this.activeModel = data.model;
      if (this.headerModel) this.headerModel.textContent = data.model;
      if (this.termHeaderModel) this.termHeaderModel.textContent = data.model;
      if (window.pixelOffice) {
        window.pixelOffice.updateFromBackend(this.models, data.model, this.isExecuting, '');
      }
    });

    sse.addEventListener('path_changed', (e) => {
      const data = JSON.parse(e.data);
      if (data.path && this.headerPath) {
        this.headerPath.textContent = data.path;
      }
    });
  }

  async changeActiveModel(modelId) {
    try {
      const res = await fetch('/api/set-model', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: modelId })
      });
      if (res.ok) {
        this.activeModel = modelId;
        this.headerModel.textContent = modelId;
        if (this.termActiveModel) this.termActiveModel.textContent = modelId;
        if (window.pixelOffice) {
          window.pixelOffice.updateFromBackend(this.models, modelId, this.isExecuting, '');
        }
        this.showToast(`🤖 Model aktif ditukar kepada: ${modelId}`);
      }
    } catch (err) {
      this.showToast(`❌ Gagal menukar model: ${err.message}`);
    }
  }

  appendTerminalLog(entry) {
    if (!entry) return;
    const rawText = entry.text !== undefined ? String(entry.text) : (entry.Text !== undefined ? String(entry.Text) : '');
    const cleanText = rawText.replace(/\x1B\[[0-9;]*[a-zA-Z]/g, '').trim();

    // Skip entirely empty log lines — never show bare timestamps
    if (!cleanText) return;

    const row = document.createElement('div');
    row.className = 'log-entry';

    const entryType = entry.type || entry.Type || 'stdout';
    const rawTime = entry.timestamp || entry.Timestamp || new Date().toTimeString().split(' ')[0];
    const entryTime = rawTime.startsWith('[') ? rawTime : `[${rawTime}]`;

    const timeSpan = document.createElement('span');
    timeSpan.className = 'log-time';
    timeSpan.textContent = entryTime;

    const pipeIdx = cleanText.indexOf(' | ');
    if (pipeIdx !== -1) {
      const tagPart = cleanText.substring(0, pipeIdx).trim();
      const msgPart = cleanText.substring(pipeIdx + 3).trim();

      const tagSpan = document.createElement('span');
      let tagClass = 'tag-default';
      const lowerTag = tagPart.toLowerCase();
      if (lowerTag.includes('thinking') || entryType === 'thinking') tagClass = 'tag-thinking';
      else if (lowerTag.includes('develop') || entryType === 'develop') tagClass = 'tag-develop';
      else if (lowerTag.includes('result') || entryType === 'result') tagClass = 'tag-result';
      else if (lowerTag.includes('prompt') || lowerTag.includes('user:') || entryType === 'prompt') tagClass = 'tag-prompt';
      else if (lowerTag.includes('system') || entryType === 'system') tagClass = 'tag-system';

      tagSpan.className = `log-tag ${tagClass}`;
      tagSpan.textContent = tagPart;

      const sepSpan = document.createElement('span');
      sepSpan.className = 'log-sep';
      sepSpan.textContent = ' │ ';

      const msgSpan = document.createElement('span');
      msgSpan.className = `log-msg ${tagClass}-msg`;
      msgSpan.textContent = msgPart;

      row.appendChild(timeSpan);
      row.appendChild(tagSpan);
      row.appendChild(sepSpan);
      row.appendChild(msgSpan);

      // Update Cursor Line at bottom of terminal
      if (this.termCursorText) {
        this.termCursorText.textContent = msgPart;
      }
      if (this.termCursorTag) {
        this.termCursorTag.textContent = tagPart;
      }

      // Update Pixel Office bubble dynamically
      if (window.pixelOffice && (tagClass === 'tag-thinking' || tagClass === 'tag-develop' || tagClass === 'tag-result')) {
        let actionDesc = msgPart;
        if (tagClass === 'tag-thinking') actionDesc = 'Berfikir: ' + msgPart;
        else if (tagClass === 'tag-develop') actionDesc = 'Menulis: ' + msgPart;
        else if (tagClass === 'tag-result') actionDesc = 'Hasil: ' + msgPart;
        window.pixelOffice.updateAgentAction(this.activeModel, tagClass, actionDesc);
      }
    } else {
      const textSpan = document.createElement('span');
      textSpan.className = `log-${entryType}`;
      textSpan.textContent = cleanText;
      row.appendChild(timeSpan);
      row.appendChild(textSpan);

      if (this.termCursorText) {
        this.termCursorText.textContent = cleanText;
      }
    }

    if (this.terminalLogs) {
      this.terminalLogs.appendChild(row);

      // Cap at 300 visible rows
      while (this.terminalLogs.children.length > 300) {
        this.terminalLogs.removeChild(this.terminalLogs.firstChild);
      }

      if (this.autoScroll) {
        this.terminalLogs.scrollTop = this.terminalLogs.scrollHeight;
      }
    }
  }

  updateSystemMetrics(stats) {
    if (!stats) return;

    // CPU Load
    if (stats.CPULoad1m) {
      document.getElementById('cpu-load-val').textContent = stats.CPULoad1m;
      const numLoad = parseFloat(stats.CPULoad1m) || 0.5;
      this.cpuHistory.push(numLoad);
      if (this.cpuHistory.length > 30) this.cpuHistory.shift();
      this.renderCpuChart();
    }

    // RAM
    if (stats.MemTotalGB > 0) {
      const used = stats.MemUsedGB.toFixed(1);
      const total = stats.MemTotalGB.toFixed(1);
      const pct = Math.min(100, Math.round((stats.MemUsedGB / stats.MemTotalGB) * 100));
      document.getElementById('ram-used-val').textContent = `${used} GB`;
      document.getElementById('ram-progress-fill').style.width = `${pct}%`;
      document.getElementById('ram-sub-text').textContent = `${used} GB digunakan daripada ${total} GB (${pct}%)`;
    }

    // Disk
    if (stats.DiskTotGB > 0) {
      const free = stats.DiskFreeGB.toFixed(1);
      const total = stats.DiskTotGB.toFixed(1);
      const pctFree = Math.min(100, Math.round((stats.DiskFreeGB / stats.DiskTotGB) * 100));
      document.getElementById('disk-free-val').textContent = `${free} GB`;
      document.getElementById('disk-progress-fill').style.width = `${pctFree}%`;
      document.getElementById('disk-sub-text').textContent = `${free} GB bebas daripada ${total} GB`;
    }

    // Sessions & Steps
    if (stats.TotalSessions !== undefined) {
      document.getElementById('stat-sessions-count').textContent = stats.TotalSessions;
      document.getElementById('stat-steps-count').textContent = stats.TotalSteps;
      document.getElementById('sessions-total-val').textContent = stats.TotalSteps;
    }
  }

  initChart() {
    this.renderCpuChart();
  }

  renderCpuChart() {
    const ctx = this.cpuChartCtx;
    const w = this.cpuChartCanvas.width;
    const h = this.cpuChartCanvas.height;

    ctx.clearRect(0, 0, w, h);

    // Grid lines
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
    ctx.lineWidth = 1;
    for (let y = 10; y < h; y += 20) {
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(w, y);
      ctx.stroke();
    }

    // Waveform curve
    const maxVal = Math.max(3.0, ...this.cpuHistory);
    const stepX = w / (this.cpuHistory.length - 1);

    ctx.beginPath();
    this.cpuHistory.forEach((val, i) => {
      const normY = h - (val / maxVal) * (h - 16) - 8;
      const x = i * stepX;
      if (i === 0) ctx.moveTo(x, normY);
      else ctx.lineTo(x, normY);
    });

    // Neon Gradient fill under curve
    const grad = ctx.createLinearGradient(0, 0, 0, h);
    grad.addColorStop(0, 'rgba(0, 215, 255, 0.35)');
    grad.addColorStop(1, 'rgba(0, 215, 255, 0.0)');

    ctx.lineTo(w, h);
    ctx.lineTo(0, h);
    ctx.closePath();
    ctx.fillStyle = grad;
    ctx.fill();

    // Line stroke
    ctx.beginPath();
    this.cpuHistory.forEach((val, i) => {
      const normY = h - (val / maxVal) * (h - 16) - 8;
      const x = i * stepX;
      if (i === 0) ctx.moveTo(x, normY);
      else ctx.lineTo(x, normY);
    });
    ctx.strokeStyle = '#00d7ff';
    ctx.lineWidth = 2.5;
    ctx.stroke();
  }

  showToast(message) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(10px)';
      toast.style.transition = 'all 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 3200);
  }
}

window.addEventListener('DOMContentLoaded', () => {
  window.miqaApp = new MiqaApp();
});
