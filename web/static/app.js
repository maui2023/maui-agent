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
    this.termActiveModel = document.getElementById('term-active-model-txt');
    this.terminalScreen = document.getElementById('terminal-screen');
    this.terminalLogs = document.getElementById('terminal-logs');
    this.termStatusText = document.getElementById('term-status-text');
    this.termStatusDot = document.getElementById('term-status-dot');
    this.termBannerStatus = document.getElementById('term-banner-status');
    this.officeStatusLabel = document.getElementById('office-status-label');
    this.officePulseDot = document.getElementById('office-pulse-dot');
    this.cpuChartCanvas = document.getElementById('cpu-chart');
    this.cpuChartCtx = this.cpuChartCanvas.getContext('2d');
  }

  initClock() {
    const clockEl = document.getElementById('live-clock');
    const updateTime = () => {
      const now = new Date();
      clockEl.textContent = now.toTimeString().split(' ')[0];
    };
    updateTime();
    setInterval(updateTime, 1000);
  }

  bindEvents() {
    // Terminal Controls
    document.getElementById('btn-toggle-scanlines').addEventListener('click', () => {
      this.terminalScreen.classList.toggle('scanlines');
      this.showToast('Kesan CRT Scanlines diubah');
    });

    document.getElementById('btn-clear-term').addEventListener('click', () => {
      this.terminalLogs.innerHTML = '';
      this.showToast('Terminal dibersihkan');
    });

    document.getElementById('chk-autoscroll').addEventListener('change', (e) => {
      this.autoScroll = e.target.checked;
    });

    // Clickable Path Pill to change workspace path
    const pillPath = document.getElementById('pill-active-path');
    if (pillPath) {
      pillPath.style.cursor = 'pointer';
      pillPath.addEventListener('click', async () => {
        const current = this.headerPath.textContent.trim();
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
              this.headerPath.textContent = data.path;
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
    try {
      const [resStatus, resModels, resLogs] = await Promise.all([
        fetch('/api/status').then(r => r.json()),
        fetch('/api/models').then(r => r.json()),
        fetch('/api/logs').then(r => r.json())
      ]);

      this.activeModel = resStatus.active_model || this.activeModel;
      this.headerModel.textContent = this.activeModel;
      if (this.termActiveModel) this.termActiveModel.textContent = this.activeModel;
      this.headerPath.textContent = resStatus.coding_path || '';

      this.models = resModels || [];
      if (window.pixelOffice) {
        window.pixelOffice.updateFromBackend(this.models, this.activeModel, resStatus.is_executing, resStatus.current_task);
      }

      if (resStatus.is_executing) {
        this.isExecuting = true;
        if (this.termStatusText) {
          this.termStatusText.textContent = `🔴 MELAKSANAKAN ARAHAN: "${resStatus.current_task || ''}"`;
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
      }

      if (resStatus.stats) {
        this.updateSystemMetrics(resStatus.stats);
      }

      // Populate logs
      if (Array.isArray(resLogs)) {
        resLogs.forEach(entry => this.appendTerminalLog(entry));
      }
    } catch (err) {
      console.warn('Gagal memuatkan data awal:', err);
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
      this.isExecuting = true;
      const task = data.task || 'Melaksanakan arahan...';
      const src = data.source || 'Telegram';

      if (this.termStatusText) {
        this.termStatusText.textContent = `🔴 MELAKSANAKAN ARAHAN (${src}): "${task}"`;
        this.termStatusText.classList.add('text-working');
      }
      if (this.termStatusDot) {
        this.termStatusDot.className = 'live-status-dot working-dot';
      }
      if (this.termBannerStatus) {
        this.termBannerStatus.textContent = `Sedang melaksanakan arahan dari ${src}...`;
      }
      if (this.officeStatusLabel) {
        this.officeStatusLabel.textContent = `🔴 Ejen Sedang Bekerja (${data.model || this.activeModel})`;
      }
      if (this.officePulseDot) {
        this.officePulseDot.className = 'pulse-indicator pulse-red';
      }

      if (window.pixelOffice) {
        window.pixelOffice.updateFromBackend(this.models, data.model, true, task);
      }
      this.showToast(`⚡ Arahan diterima dari ${src} untuk model ${data.model || this.activeModel}`);
    });

    sse.addEventListener('agent_finished', (e) => {
      const data = JSON.parse(e.data);
      this.isExecuting = false;

      if (this.termStatusText) {
        this.termStatusText.textContent = '🟢 MENUNGGU ARAHAN DARI TELEGRAM (@miqa_agy_bot) ATAU CLI...';
        this.termStatusText.classList.remove('text-working');
      }
      if (this.termStatusDot) {
        this.termStatusDot.className = 'live-status-dot idle-dot';
      }
      if (this.termBannerStatus) {
        this.termBannerStatus.textContent = 'Menunggu arahan pengekodan dari Telegram (@miqa_agy_bot) atau CLI...';
      }
      if (this.officeStatusLabel) {
        this.officeStatusLabel.textContent = '🟢 Sedia Menunggu Arahan';
      }
      if (this.officePulseDot) {
        this.officePulseDot.className = 'pulse-indicator';
      }

      if (window.pixelOffice) {
        window.pixelOffice.updateFromBackend(this.models, data.model, false, '');
      }
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
      this.headerModel.textContent = data.model;
      this.termActiveModel.textContent = data.model;
      if (window.pixelOffice) {
        window.pixelOffice.updateFromBackend(this.models, data.model, this.isExecuting, '');
      }
    });

    sse.addEventListener('path_changed', (e) => {
      const data = JSON.parse(e.data);
      if (data.path) {
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
    const rawText = entry.Text !== undefined ? String(entry.Text) : '';
    const cleanText = rawText.replace(/\x1B\[[0-9;]*[a-zA-Z]/g, '').trim();

    // Skip entirely empty log lines — never show bare timestamps
    if (!cleanText) return;

    const row = document.createElement('div');
    row.className = 'log-entry';

    const timeSpan = document.createElement('span');
    timeSpan.className = 'log-time';
    timeSpan.textContent = entry.Timestamp || new Date().toTimeString().split(' ')[0];

    const pipeIdx = cleanText.indexOf(' | ');
    if (pipeIdx !== -1) {
      const tagPart = cleanText.substring(0, pipeIdx).trim();
      const msgPart = cleanText.substring(pipeIdx + 3).trim();

      const tagSpan = document.createElement('span');
      let tagClass = 'tag-default';
      const lowerTag = tagPart.toLowerCase();
      if (lowerTag.includes('thinking') || entry.Type === 'thinking') tagClass = 'tag-thinking';
      else if (lowerTag.includes('develop') || entry.Type === 'develop') tagClass = 'tag-develop';
      else if (lowerTag.includes('result') || entry.Type === 'result') tagClass = 'tag-result';
      else if (lowerTag.includes('prompt') || lowerTag.includes('user:') || entry.Type === 'prompt') tagClass = 'tag-prompt';
      else if (lowerTag.includes('system') || entry.Type === 'system') tagClass = 'tag-system';

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
    } else {
      const textSpan = document.createElement('span');
      textSpan.className = `log-${entry.Type || 'stdout'}`;
      textSpan.textContent = cleanText;
      row.appendChild(timeSpan);
      row.appendChild(textSpan);
    }

    this.terminalLogs.appendChild(row);

    // Cap at 300 visible rows
    while (this.terminalLogs.children.length > 300) {
      this.terminalLogs.removeChild(this.terminalLogs.firstChild);
    }

    if (this.autoScroll) {
      this.terminalScreen.scrollTop = this.terminalScreen.scrollHeight;
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
