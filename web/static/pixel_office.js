/**
 * MIQA Pixel Office Engine v6.0 — True-to-Reference Warm Isometric Office
 *
 * Faithfully matches the reference image:
 *   - Rich isometric parquet floor tiles ("kotak pixel lantai") in warm golden oak
 *   - 4-desk connected wooden pod where ALL 4 AI models sit together:
 *       1. Gemini Flash (Top-Left): Green sweater developer facing viewer, sandy-brown hair,
 *          white speech bubble "Fetching web content", CRT monitor, keyboard & curly mouse wire.
 *       2. GPT-OSS (Front-Left): Purple shirt, dark skin, black swivel chair with 5-star wheels,
 *          typing on keyboard, CRT monitor with dark green terminal code screen.
 *       3. Claude Sonnet (Front-Right): Purple suit woman with long auburn hair, black swivel chair,
 *          typing on keyboard, CRT monitor showing colorful browser/document layout.
 *       4. Ollama Local (Top-Right): Sitting at back-right desk, brass table lamp with glowing
 *          yellow bell shade between desks, CRT monitor, sleeping Zzz or active typing.
 *   - 3 secondary ambient desk pods (empty, ready for future models) on the floor tiles.
 *   - Cozy dark navy/purple walls with baseboard trims.
 *   - Bookshelves with colorful books, whiteboard with charts, potted topiary plants.
 *   - Windows on left wall with diagonal warm sunlight beams & floating dust particles.
 */

class PixelOffice {
  constructor(canvasId) {
    this.canvas = document.getElementById(canvasId);
    if (!this.canvas) return;
    this.ctx = this.canvas.getContext('2d');
    this.ctx.imageSmoothingEnabled = false;

    this.W = this.canvas.width;   // 1100
    this.H = this.canvas.height;  // 520

    // Isometric tile grid parameters
    this.TW = 56; // tile width in pixels
    this.TH = 28; // tile height in pixels (2:1 isometric ratio)

    // Floor origin (top back corner of the room in canvas coords)
    this.OX = 530;
    this.OY = 74;

    this.tick = 0;
    this.hoveredAgent = null;
    this._hitAreas = [];
    this.sparks = [];
    this.dust = [];

    // ── 4 AI Models seated at the main 4-desk pod ────────────────────────────
    this.agents = [
      {
        id: 'gemini-3.8-flash-high',
        name: 'Gemini Flash',
        role: 'Lead Dev',
        seat: 'back-left',
        // Visual attributes matching green-sweater developer in reference
        hairColor: '#9c6738',
        skinColor: '#fed7aa',
        clothColor: '#38a169',
        color: '#00d7ff',
        avatarEmoji: '⚡',
        status: 'working',
        credits: 100,
        task: 'Fetching web content',
        bubble: 'Fetching web content'
      },
      {
        id: 'gpt-oss-120b-medium',
        name: 'GPT-OSS',
        role: 'DevOps / Code',
        seat: 'front-left',
        // Visual attributes matching purple-shirt developer in reference
        hairColor: '#18181b',
        skinColor: '#7c4a24',
        clothColor: '#7c3aed',
        color: '#50fa7b',
        avatarEmoji: '🤖',
        status: 'sleeping',
        credits: 100,
        task: '',
        bubble: 'Compiling Daemon'
      },
      {
        id: 'claude-sonnet-4-6',
        name: 'Claude Sonnet',
        role: 'Reviewer / Analyst',
        seat: 'front-right',
        // Visual attributes matching purple-suit woman in reference
        hairColor: '#9a3412',
        skinColor: '#fed7aa',
        clothColor: '#9333ea',
        color: '#ffb86c',
        avatarEmoji: '📜',
        status: 'sleeping',
        credits: 100,
        task: '',
        bubble: 'Reviewing Code'
      },
      {
        id: 'ollama-local',
        name: 'Ollama Local',
        role: 'Offline Inference',
        seat: 'back-right',
        // Visual attributes matching top-right desk model
        hairColor: '#334155',
        skinColor: '#fde68a',
        clothColor: '#2563eb',
        color: '#bd93f9',
        avatarEmoji: '🦙',
        status: 'sleeping',
        credits: 100,
        task: '',
        bubble: 'Local Model Ready'
      }
    ];

    this.activeModel = 'gemini-3.8-flash-high';

    // ── Floating dust motes in sunlight ─────────────────────────────────────
    for (let i = 0; i < 50; i++) {
      this.dust.push({
        x: 40 + Math.random() * 520,
        y: 60 + Math.random() * 380,
        vx: 0.08 + Math.random() * 0.18,
        vy: 0.05 + Math.random() * 0.12,
        r: 0.8 + Math.random() * 1.5,
        a: 0.15 + Math.random() * 0.45,
        phase: Math.random() * Math.PI * 2
      });
    }

    this.initEvents();
    this.animate();
  }

  // ── Isometric Coordinate Transform ─────────────────────────────────────────
  // Given tile coordinates (col, row) and pixel elevation (elevPx)
  iso(col, row, elevPx = 0) {
    return {
      x: this.OX + (col - row) * (this.TW / 2),
      y: this.OY + (col + row) * (this.TH / 2) - elevPx
    };
  }

  // ── Mouse & Interaction Events ─────────────────────────────────────────────
  initEvents() {
    this.canvas.addEventListener('mousemove', (e) => {
      const r = this.canvas.getBoundingClientRect();
      const mx = (e.clientX - r.left) * (this.W / r.width);
      const my = (e.clientY - r.top) * (this.H / r.height);
      this.hoveredAgent = null;
      for (const h of this._hitAreas) {
        if (mx >= h.x && mx <= h.x + h.w && my >= h.y && my <= h.y + h.h) {
          this.hoveredAgent = h.agent;
          break;
        }
      }
      this.canvas.style.cursor = this.hoveredAgent ? 'pointer' : 'default';
    });

    this.canvas.addEventListener('click', () => {
      if (this.hoveredAgent && window.selectModelFromUI) {
        window.selectModelFromUI(this.hoveredAgent.id);
      }
    });
  }

  updateFromBackend(modelsList, activeModelId, isExecuting, currentTask) {
    if (activeModelId) this.activeModel = activeModelId;
    this.agents.forEach(ag => {
      if (ag.id === this.activeModel) {
        ag.status = isExecuting ? 'working' : 'idle';
        ag.task = isExecuting ? (currentTask || ag.bubble) : (ag.id === 'gemini-3.8-flash-high' ? 'Fetching web content' : '');
      } else {
        ag.status = 'sleeping';
        ag.task = '';
      }
    });
    this.renderRosterCards();
  }

  updateAgentAction(modelId, actionType, actionText) {
    const targetId = modelId || this.activeModel;
    const ag = this.agents.find(a => a.id === targetId);
    if (ag) {
      ag.status = 'working';
      let clean = String(actionText || 'Bekerja...');
      if (clean.length > 34) clean = clean.slice(0, 31) + '...';
      ag.task = clean;
    }
  }

  renderRosterCards() {
    const el = document.getElementById('agent-roster-grid');
    if (!el) return;
    el.innerHTML = this.agents.map(ag => {
      const act = ag.id === this.activeModel;
      let sc = 'status-sleeping', st = '💤 Tidur';
      if (ag.status === 'working') { sc = 'status-working'; st = '⚡ Bekerja'; }
      else if (ag.status === 'idle') { sc = 'status-idle'; st = '☕ Sedia'; }
      else if (ag.credits <= 0) { sc = 'status-exhausted'; st = '🍕 Habis'; }
      return `<div class="agent-mini-card ${act ? 'active-card' : ''}" onclick="window.selectModelFromUI('${ag.id}')">
        <div class="card-top-row"><span class="agent-avatar-badge">${ag.avatarEmoji}</span><span class="status-badge ${sc}">${st}</span></div>
        <div><div class="agent-name-text">${ag.name}</div><div class="agent-role-text">${ag.role}</div></div>
        <div class="agent-credit-bar"><span>Kredit: ${ag.credits}%</span><div class="mini-credit-fill"><div class="mini-credit-inner ${ag.credits <= 20 ? 'low' : ''}" style="width:${ag.credits}%"></div></div></div>
      </div>`;
    }).join('');
  }

  // ── Main Animation Loop ────────────────────────────────────────────────────
  animate() {
    this.tick++;
    this.render();
    requestAnimationFrame(() => this.animate());
  }

  render() {
    const c = this.ctx;
    this._hitAreas = [];
    c.clearRect(0, 0, this.W, this.H);

    // 1. Walls and floor base
    this.drawRoomShell(c);

    // 2. Parquet Floor Tiles ("kotak pixel lantai")
    this.drawParquetFloor(c);

    // 3. Wall furniture & decorations (bookshelves, windows, whiteboard, plants)
    this.drawWallDecorations(c);

    // 4. Sunlight beams through the windows
    this.drawSunlightBeams(c);

    // 5. Desk Pod 2: Top-left empty secondary workstation (near bookshelf)
    this.drawSecondaryDeskGroup(c, 1.5, 2.0, 'MEJA 2', false);

    // 6. Desk Pod 3: Right side empty secondary workstation (near whiteboard)
    this.drawSecondaryDeskGroup(c, 10.0, 3.0, 'MEJA 3', false);

    // 7. Desk Pod 4: Foreground empty secondary workstation
    this.drawSecondaryDeskGroup(c, 9.5, 8.5, 'MEJA 4', false);

    // 8. Desk Pod 1 (MAIN 4-DESK CLUSTER): ALL 4 AGENTS SITTING HERE
    // Placed squarely on floor tiles at col: 4.8, row: 5.6
    this.drawMainDeskCluster(c, 4.8, 5.6);

    // 9. Floating dust motes and typing sparks
    this.drawParticles(c);
  }

  // ══════════════════════════════════════════════════════════════════════════
  // ROOM SHELL & BASEBOARDS
  // ══════════════════════════════════════════════════════════════════════════
  drawRoomShell(c) {
    const { OX, OY, W, H } = this;

    // Room background fill (dark cozy atmosphere)
    c.fillStyle = '#161224';
    c.fillRect(0, 0, W, H);

    // Left wall: from top-left (0,0) to back corner (OX, OY) down to left floor base (0, 339)
    const leftBaseY = OY + (OX * 0.5); // ~339
    c.fillStyle = '#221936';
    c.beginPath();
    c.moveTo(0, 0);
    c.lineTo(OX, 0);
    c.lineTo(OX, OY);
    c.lineTo(0, leftBaseY);
    c.closePath();
    c.fill();

    // Right wall: from (OX, 0) to top-right (W, 0) down to right floor base (W, 359)
    const rightBaseY = OY + ((W - OX) * 0.5); // ~359
    c.fillStyle = '#1b142d';
    c.beginPath();
    c.moveTo(OX, 0);
    c.lineTo(W, 0);
    c.lineTo(W, rightBaseY);
    c.lineTo(OX, OY);
    c.closePath();
    c.fill();

    // Corner seam between walls
    c.strokeStyle = '#120d1c';
    c.lineWidth = 2;
    c.beginPath();
    c.moveTo(OX, 0);
    c.lineTo(OX, OY);
    c.stroke();

    // Left baseboard (rich dark mahogany)
    c.strokeStyle = '#4a2c17';
    c.lineWidth = 6;
    c.beginPath();
    c.moveTo(0, leftBaseY);
    c.lineTo(OX, OY);
    c.stroke();
    c.strokeStyle = '#6d4223';
    c.lineWidth = 2;
    c.beginPath();
    c.moveTo(0, leftBaseY - 2);
    c.lineTo(OX, OY - 2);
    c.stroke();

    // Right baseboard
    c.strokeStyle = '#381f0f';
    c.lineWidth = 6;
    c.beginPath();
    c.moveTo(OX, OY);
    c.lineTo(W, rightBaseY);
    c.stroke();
    c.strokeStyle = '#543017';
    c.lineWidth = 2;
    c.beginPath();
    c.moveTo(OX, OY - 2);
    c.lineTo(W, rightBaseY - 2);
    c.stroke();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // PARQUET FLOOR TILES ("kotak pixel lantai")
  // ══════════════════════════════════════════════════════════════════════════
  drawParquetFloor(c) {
    const { TW, TH, W, H } = this;
    const COLS = 20;
    const ROWS = 20;

    // Two warm golden-amber wood tile tones
    const TILE_LIGHT = '#d29853'; // warm golden honey oak
    const TILE_DARK  = '#bb7e39'; // richer golden amber teak
    const TILE_LINE  = '#945c22'; // distinct pixel tile grid line

    for (let r = 0; r < ROWS; r++) {
      for (let col = 0; col < COLS; col++) {
        const top = this.iso(col, r);

        // Cull tiles entirely outside the canvas
        if (top.y > H + 40 || top.x < -80 || top.x > W + 80) continue;

        const x = top.x;
        const y = top.y;

        const isLight = (col + r) % 2 === 0;
        c.fillStyle = isLight ? TILE_LIGHT : TILE_DARK;

        // Draw diamond tile ("kotak pixel lantai")
        c.beginPath();
        c.moveTo(x, y);
        c.lineTo(x + TW / 2, y + TH / 2);
        c.lineTo(x, y + TH);
        c.lineTo(x - TW / 2, y + TH / 2);
        c.closePath();
        c.fill();

        // Parquet wood plank grain stripes inside tile
        c.save();
        c.strokeStyle = isLight ? '#c48946' : '#ad702e';
        c.lineWidth = 1;
        if (isLight) {
          // Plank grain running parallel to row axis
          c.beginPath();
          c.moveTo(x - TW / 4, y + TH / 4);
          c.lineTo(x + TW / 4, y + 3 * TH / 4);
          c.moveTo(x - TW / 8, y + TH / 8);
          c.lineTo(x + 3 * TW / 8, y + 5 * TH / 8);
          c.stroke();
        } else {
          // Plank grain running parallel to col axis
          c.beginPath();
          c.moveTo(x + TW / 4, y + TH / 4);
          c.lineTo(x - TW / 4, y + 3 * TH / 4);
          c.moveTo(x + TW / 8, y + TH / 8);
          c.lineTo(x - 3 * TW / 8, y + 5 * TH / 8);
          c.stroke();
        }
        c.restore();

        // Tile border outline (crisp pixel border)
        c.strokeStyle = TILE_LINE;
        c.lineWidth = 0.8;
        c.stroke();
      }
    }
  }

  // ══════════════════════════════════════════════════════════════════════════
  // WALL DECORATIONS — Windows, Bookshelves, Whiteboard, Plants
  // ══════════════════════════════════════════════════════════════════════════
  drawWallDecorations(c) {
    // Left wall:
    // Bookshelf far left
    this.drawBookshelf(c, 8, 40, 50, 180);
    // Window 1
    this.drawWindow(c, 68, 26, 76, 145);
    // Bookshelf between windows
    this.drawBookshelf(c, 154, 150, 48, 80);
    // Window 2
    this.drawWindow(c, 160, 6, 74, 125);

    // Corner potted topiary plant
    this.drawTopiaryTree(c, 445, 12, 38);

    // Right wall:
    // Large architecture/metrics whiteboard
    this.drawWhiteboard(c, 560, 14, 200, 126);
    // Bookshelf right 1
    this.drawBookshelf(c, 796, 55, 56, 190);
    // Bookshelf right 2
    this.drawBookshelf(c, 868, 98, 58, 205);
    // Potted plant on far right
    this.drawTopiaryTree(c, 965, 195, 36);
  }

  drawWindow(c, x, y, w, h) {
    c.save();
    c.fillStyle = '#6b3c18'; c.fillRect(x, y, w, h);
    c.fillStyle = '#4a250c'; c.fillRect(x + 3, y + 3, w - 6, h - 6);

    // Bright morning sunlight glass
    const g = c.createLinearGradient(x, y, x, y + h);
    g.addColorStop(0, '#fffbe6');
    g.addColorStop(0.5, '#fed766');
    g.addColorStop(1, '#f6ba13');
    c.fillStyle = g;
    c.fillRect(x + 6, y + 6, w - 12, h - 12);

    // Window frame dividers
    c.fillStyle = '#6b3c18';
    c.fillRect(x + w / 2 - 2, y + 6, 4, h - 12);
    c.fillRect(x + 6, y + h * 0.44, w - 12, 4);

    // Venetian blinds
    for (let b = 0; b < 6; b++) {
      c.fillStyle = b % 2 === 0 ? '#fdf8ea' : '#ebdcc0';
      c.fillRect(x + 6, y + 6 + b * 5, w - 12, 4);
    }
    // Sill
    c.fillStyle = '#5c3112'; c.fillRect(x - 2, y + h - 4, w + 4, 5);
    c.restore();
  }

  drawBookshelf(c, x, y, w, h) {
    c.save();
    c.fillStyle = '#6e3c1a'; c.fillRect(x, y, w, h);
    c.fillStyle = '#3a1e0b'; c.fillRect(x + 3, y + 3, w - 6, h - 6);

    const shelves = 4;
    const shH = Math.floor((h - 8) / shelves);
    const bkCols = ['#c0392b','#27ae60','#2980b9','#f1c40f','#e67e22','#8e44ad','#1abc9c','#ecf0f1','#d35400','#e74c3c','#3498db'];

    for (let s = 0; s < shelves; s++) {
      const sy = y + 4 + s * shH;
      c.fillStyle = '#7a4521'; c.fillRect(x + 2, sy + shH - 4, w - 4, 4);

      let bx = x + 4, bi = (s * 3) % bkCols.length;
      while (bx < x + w - 5) {
        const bw = 3 + (bi % 3);
        const bh = shH - 5 - (bi % 4);
        c.fillStyle = bkCols[bi % bkCols.length];
        c.fillRect(bx, sy + shH - 4 - bh, bw, bh);
        c.fillStyle = 'rgba(255,255,255,0.3)';
        c.fillRect(bx + 1, sy + shH - 4 - bh + 2, 1, bh - 3);
        bx += bw + 1; bi++;
      }
    }
    c.fillStyle = '#5c3112'; c.fillRect(x - 2, y - 4, w + 4, 5);
    c.restore();
  }

  drawWhiteboard(c, x, y, w, h) {
    c.save();
    c.fillStyle = '#525b6d'; c.fillRect(x, y, w, h);
    c.fillStyle = '#f8f9fa'; c.fillRect(x + 5, y + 5, w - 10, h - 10);

    // Title
    c.font = '6px "Press Start 2P",monospace';
    c.fillStyle = '#1e293b'; c.textAlign = 'left';
    c.fillText('ARCHITECTURE & METRICS', x + 10, y + 17);

    c.strokeStyle = '#d1d5db'; c.lineWidth = 1;
    c.beginPath(); c.moveTo(x + 8, y + 21); c.lineTo(x + w - 8, y + 21); c.stroke();

    // Pie chart
    const pcx = x + 38, pcy = y + 64, pr = 25;
    const slices = [['#22c55e', 0, 0.85], ['#f97316', 0.85, 1.55], ['#eab308', 1.55, 2.0]];
    slices.forEach(([col, s, e]) => {
      c.fillStyle = col; c.beginPath(); c.moveTo(pcx, pcy);
      c.arc(pcx, pcy, pr, Math.PI * s, Math.PI * e); c.closePath(); c.fill();
    });
    c.strokeStyle = '#fff'; c.lineWidth = 1.5;
    c.beginPath(); c.arc(pcx, pcy, pr, 0, Math.PI * 2); c.stroke();

    // Bar chart
    const bcx = x + 86, bcy = y + 90;
    [['#3b82f6', 22], ['#10b981', 38], ['#f59e0b', 28], ['#8b5cf6', 44], ['#ef4444', 18]].forEach(([col, bh], i) => {
      c.fillStyle = col; c.fillRect(bcx + i * 12, bcy - bh, 10, bh);
    });
    c.strokeStyle = '#94a3b8'; c.lineWidth = 1;
    c.beginPath(); c.moveTo(bcx - 3, bcy); c.lineTo(bcx + 66, bcy); c.stroke();

    // Line trend chart
    c.strokeStyle = '#2563eb'; c.lineWidth = 2; c.beginPath();
    [[152,34],[162,25],[172,29],[182,18],[192,22],[202,14]].forEach(([dx, dy], i) => {
      i === 0 ? c.moveTo(x + dx, y + dy) : c.lineTo(x + dx, y + dy);
    });
    c.stroke();

    // Pen tray
    c.fillStyle = '#94a3b8'; c.fillRect(x + w / 2 - 25, y + h - 6, 50, 4);
    c.restore();
  }

  drawTopiaryTree(c, x, y, sz) {
    c.save();
    // Terracotta pot
    c.fillStyle = '#bd532b';
    c.beginPath();
    c.moveTo(x - 12, y + 18); c.lineTo(x + 12, y + 18);
    c.lineTo(x + 9, y + 38);  c.lineTo(x - 9, y + 38);
    c.closePath(); c.fill();
    c.fillStyle = '#943e1c'; c.fillRect(x - 14, y + 16, 28, 4);

    // Trunk
    c.fillStyle = '#5c3314'; c.fillRect(x - 2, y - 4, 5, 22);

    // Spherical foliage
    [[-4,-8,sz*0.52,'#2d6a4f'],[4,-8,sz*0.48,'#27ae60'],[-1,-14,sz*0.42,'#48c774'],
     [-5,-2,sz*0.32,'#219653'],[5,-3,sz*0.30,'#2ecc71']
    ].forEach(([ox, oy, r, col]) => {
      c.fillStyle = col; c.beginPath(); c.arc(x+ox, y+oy, r, 0, Math.PI*2); c.fill();
    });
    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // SUNLIGHT BEAMS
  // ══════════════════════════════════════════════════════════════════════════
  drawSunlightBeams(c) {
    c.save();
    // Beam 1 (from window 1)
    let g1 = c.createLinearGradient(95, 100, 410, 480);
    g1.addColorStop(0, 'rgba(255, 235, 140, 0.28)');
    g1.addColorStop(0.4, 'rgba(255, 215, 90, 0.12)');
    g1.addColorStop(1, 'transparent');
    c.fillStyle = g1;
    c.beginPath();
    c.moveTo(68, 70); c.lineTo(148, 126); c.lineTo(410, 520); c.lineTo(210, 520);
    c.closePath(); c.fill();

    // Beam 2 (from window 2)
    let g2 = c.createLinearGradient(190, 40, 540, 480);
    g2.addColorStop(0, 'rgba(255, 238, 150, 0.24)');
    g2.addColorStop(0.45, 'rgba(255, 220, 100, 0.10)');
    g2.addColorStop(1, 'transparent');
    c.fillStyle = g2;
    c.beginPath();
    c.moveTo(160, 15); c.lineTo(236, 62); c.lineTo(550, 500); c.lineTo(370, 500);
    c.closePath(); c.fill();
    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // SECONDARY DESK PODS (Empty workstations on floor tiles)
  // ══════════════════════════════════════════════════════════════════════════
  drawSecondaryDeskGroup(c, col, row, label, occupied) {
    const dElev = 22;
    const w = 2.4, d = 1.3;

    // Floor shadow
    const floorCenter = this.iso(col + w / 2, row + d / 2, 0);
    c.save();
    c.fillStyle = 'rgba(0,0,0,0.18)';
    c.beginPath();
    c.ellipse(floorCenter.x, floorCenter.y, this.TW * 1.3, this.TH * 0.9, 0, 0, Math.PI * 2);
    c.fill();
    c.restore();

    // Desk 3D Box
    this.drawIsometricBox(c, col, row, w, d, dElev, '#b67e3e', '#7f4e1f', '#966028');

    // Desk items: Papers & empty CRT monitor
    const p1 = this.iso(col + 0.5, row + 0.3, dElev);
    this.drawRetroCRTMonitor(c, p1.x, p1.y - 12, 'idle', '#10b981', false);

    const p2 = this.iso(col + 1.8, row + 0.3, dElev);
    this.drawRetroCRTMonitor(c, p2.x, p2.y - 12, 'idle', '#3b82f6', false);

    // Office swivel chair sitting on floor
    const ch = this.iso(col + 1.1, row + d + 0.5, 0);
    this.drawSwivelChair(c, ch.x, ch.y, 'front');

    // Desk label
    const lbl = this.iso(col + w / 2, row + d + 0.6, 0);
    this.drawDeskLabel(c, lbl.x, lbl.y + 6, label, occupied);
  }

  // ══════════════════════════════════════════════════════════════════════════
  // MAIN 4-DESK CLUSTER — EXACTLY MATCHING THE USER'S REFERENCE IMAGE
  //
  // Layout: 4 interconnected wooden desks forming a 2x2 pod.
  //   - Top-Left Desk:   Gemini Flash (green shirt developer, facing forward/SE)
  //   - Top-Right Desk:  Ollama Local (facing forward/SW) + Yellow Desk Lamp
  //   - Bottom-Left Desk: GPT-OSS (purple shirt, facing back/NW towards screen)
  //   - Bottom-Right Desk: Claude Sonnet (purple suit woman, facing back/NW)
  // ══════════════════════════════════════════════════════════════════════════
  drawMainDeskCluster(c, baseCol, baseRow) {
    const deskW = 1.7; // width of each individual desk in tile units
    const deskD = 1.3; // depth of each individual desk in tile units
    const dElev = 24;  // height/elevation of desk surface in pixels

    // Coordinates of the 4 desks in tile grid:
    // Top-Left:     col: baseCol,         row: baseRow
    // Top-Right:    col: baseCol + deskW, row: baseRow
    // Bottom-Left:  col: baseCol,         row: baseRow + deskD
    // Bottom-Right: col: baseCol + deskW, row: baseRow + deskD

    const cTL = baseCol;
    const rTL = baseRow;
    const cTR = baseCol + deskW;
    const rTR = baseRow;
    const cBL = baseCol;
    const rBL = baseRow + deskD;
    const cBR = baseCol + deskW;
    const rBR = baseRow + deskD;

    // Floor shadow under the entire 4-desk pod
    const podCenter = this.iso(baseCol + deskW, baseRow + deskD, 0);
    c.save();
    c.fillStyle = 'rgba(0, 0, 0, 0.22)';
    c.beginPath();
    c.ellipse(podCenter.x, podCenter.y, this.TW * 2.2, this.TH * 1.6, 0, 0, Math.PI * 2);
    c.fill();
    c.restore();

    // ── STEP 1: BACK CHAIRS & BACK CHARACTERS (Gemini & Ollama) ─────────────
    // Drawn first so they appear behind the wooden desks

    // 1A. Gemini Flash (Back-Left Desk):
    const agGemini = this.agents[0];
    const geminiChairPos = this.iso(cTL + 0.6, rTL - 0.4, 0);
    this.drawSwivelChair(c, geminiChairPos.x, geminiChairPos.y, 'back');
    const geminiBodyPos = this.iso(cTL + 0.6, rTL + 0.1, 8);
    this.drawAgentFacingFront(c, agGemini, geminiBodyPos.x, geminiBodyPos.y - 12, agGemini.status === 'working');

    // Register hit area for Gemini
    this._hitAreas.push({
      x: geminiBodyPos.x - 20, y: geminiBodyPos.y - 40, w: 40, h: 54, agent: agGemini
    });

    // 1B. Ollama Local (Back-Right Desk):
    const agOllama = this.agents[3];
    const ollamaChairPos = this.iso(cTR + 1.1, rTR - 0.4, 0);
    this.drawSwivelChair(c, ollamaChairPos.x, ollamaChairPos.y, 'back');
    const ollamaBodyPos = this.iso(cTR + 1.1, rTR + 0.1, 8);
    this.drawAgentFacingFront(c, agOllama, ollamaBodyPos.x, ollamaBodyPos.y - 12, agOllama.status === 'working');

    // Register hit area for Ollama
    this._hitAreas.push({
      x: ollamaBodyPos.x - 20, y: ollamaBodyPos.y - 40, w: 40, h: 54, agent: agOllama
    });

    // ── STEP 2: TOP DESKS (Top-Left and Top-Right) ─────────────────────────
    // Draw Desk TL
    this.drawIsometricBox(c, cTL, rTL, deskW, deskD, dElev, '#c68a49', '#8a5624', '#a46a30');
    // Draw Desk TR
    this.drawIsometricBox(c, cTR, rTR, deskW, deskD, dElev, '#c88c4b', '#8c5826', '#a66c32');

    // ── STEP 3: ITEMS ON TOP DESKS ─────────────────────────────────────────

    // 3A. CRT Monitor on Gemini's desk (to his right, angled)
    const geminiMonPos = this.iso(cTL + 1.3, rTL + 0.3, dElev);
    this.drawRetroCRTMonitorBack(c, geminiMonPos.x, geminiMonPos.y - 10);

    // Keyboard & Mouse with curled wire on Gemini's desk
    const geminiKbPos = this.iso(cTL + 0.7, rTL + 0.7, dElev);
    this.drawKeyboardAndMouse(c, geminiKbPos.x, geminiKbPos.y, agGemini.status === 'working', agGemini.color);

    // 3B. Glowing Brass Table Lamp in the center between Desk TL and Desk TR
    const lampTilePos = this.iso(baseCol + deskW, baseRow + 0.4, dElev);
    this.drawGlowingDeskLamp(c, lampTilePos.x, lampTilePos.y - 4);

    // 3C. CRT Monitor on Ollama's desk
    const ollamaMonPos = this.iso(cTR + 0.4, rTR + 0.3, dElev);
    this.drawRetroCRTMonitorBack(c, ollamaMonPos.x, ollamaMonPos.y - 10);
    const ollamaKbPos = this.iso(cTR + 1.1, rTR + 0.7, dElev);
    this.drawKeyboardAndMouse(c, ollamaKbPos.x, ollamaKbPos.y, agOllama.status === 'working', agOllama.color);

    // ── STEP 4: BOTTOM DESKS (Bottom-Left and Bottom-Right) ─────────────────
    // Draw Desk BL
    this.drawIsometricBox(c, cBL, rBL, deskW, deskD, dElev, '#c38746', '#875321', '#9e642a');
    // Draw Desk BR
    this.drawIsometricBox(c, cBR, rBR, deskW, deskD, dElev, '#c68a49', '#8a5624', '#a46a30');

    // ── STEP 5: CRT MONITORS & KEYBOARDS ON BOTTOM DESKS ───────────────────
    // Both monitors face the front agents (so their screens are visible to us!)

    // 5A. Monitor on Desk BL for GPT-OSS: Green terminal code screen!
    const gptMonPos = this.iso(cBL + 0.8, rBL + 0.3, dElev);
    this.drawRetroCRTMonitor(c, gptMonPos.x, gptMonPos.y - 12, 'terminal', '#22c55e', agGemini.status === 'working' || agGPT_OSS_working(this));
    const gptKbPos = this.iso(cBL + 0.8, rBL + 0.9, dElev);
    this.drawKeyboard(c, gptKbPos.x, gptKbPos.y, true, '#22c55e');

    // 5B. Monitor on Desk BR for Claude Sonnet: Colorful browser/document layout screen!
    const claudeMonPos = this.iso(cBR + 0.8, rBL + 0.3, dElev);
    this.drawRetroCRTMonitor(c, claudeMonPos.x, claudeMonPos.y - 12, 'browser', '#38bdf8', true);
    const claudeKbPos = this.iso(cBR + 0.8, rBR + 0.9, dElev);
    this.drawKeyboard(c, claudeKbPos.x, claudeKbPos.y, true, '#a855f7');

    function agGPT_OSS_working(self) {
      return self.agents[1].status === 'working';
    }

    // ── STEP 6: FRONT CHARACTERS & CHAIRS (GPT-OSS & Claude Sonnet) ─────────
    // Drawn in front of the bottom desks, facing away from viewer (NW)

    // 6A. GPT-OSS (Purple shirt, dark skin, facing north-west typing on keyboard):
    const agGPT = this.agents[1];
    const gptChairPos = this.iso(cBL + 0.8, rBL + deskD + 0.5, 0);
    this.drawSwivelChair(c, gptChairPos.x, gptChairPos.y, 'front');
    const gptCharY = gptChairPos.y - 20;
    this.drawAgentFacingBack(c, agGPT, gptChairPos.x, gptCharY, agGPT.status === 'working');

    // Hit area for GPT-OSS
    this._hitAreas.push({
      x: gptChairPos.x - 20, y: gptCharY - 32, w: 40, h: 54, agent: agGPT
    });

    // 6B. Claude Sonnet (Purple suit woman with long auburn hair, typing):
    const agClaude = this.agents[2];
    const claudeChairPos = this.iso(cBR + 0.8, rBR + deskD + 0.5, 0);
    this.drawSwivelChair(c, claudeChairPos.x, claudeChairPos.y, 'front');
    const claudeCharY = claudeChairPos.y - 20;
    this.drawAgentFacingBack(c, agClaude, claudeChairPos.x, claudeCharY, agClaude.status === 'working');

    // Hit area for Claude
    this._hitAreas.push({
      x: claudeChairPos.x - 20, y: claudeCharY - 32, w: 40, h: 54, agent: agClaude
    });

    // ── STEP 7: SPEECH BUBBLES & ACTIVE INDICATORS ─────────────────────────

    // Speech bubble for Gemini (matches reference image: "Fetching web content")
    if (agGemini.status === 'working' || this.activeModel === agGemini.id) {
      const bubbleText = agGemini.task || 'Fetching web content';
      this.drawSpeechBubble(c, geminiBodyPos.x + 8, geminiBodyPos.y - 48, bubbleText);
    }

    // Speech bubble for other agents when active
    if (agGPT.status === 'working' && this.activeModel === agGPT.id) {
      this.drawSpeechBubble(c, gptChairPos.x, gptCharY - 48, agGPT.task || agGPT.bubble);
    }
    if (agClaude.status === 'working' && this.activeModel === agClaude.id) {
      this.drawSpeechBubble(c, claudeChairPos.x, claudeCharY - 48, agClaude.task || agClaude.bubble);
    }
    if (agOllama.status === 'working' && this.activeModel === agOllama.id) {
      this.drawSpeechBubble(c, ollamaBodyPos.x, ollamaBodyPos.y - 48, agOllama.task || agOllama.bubble);
    }

    // Floating Zzz for sleeping agents
    if (agOllama.status === 'sleeping') {
      this.drawZzz(c, ollamaBodyPos.x + 14, ollamaBodyPos.y - 25);
    }
    if (agGPT.status === 'sleeping' && this.activeModel !== agGPT.id) {
      this.drawZzz(c, gptChairPos.x + 14, gptCharY - 25);
    }
    if (agClaude.status === 'sleeping' && this.activeModel !== agClaude.id) {
      this.drawZzz(c, claudeChairPos.x + 14, claudeCharY - 25);
    }

    // Desk Pod Label
    const podLabelPos = this.iso(baseCol + deskW, baseRow + deskD * 2 + 0.8, 0);
    this.drawDeskLabel(c, podLabelPos.x, podLabelPos.y + 10, 'MEJA 1 (MIQA POD)', true);
  }

  // ══════════════════════════════════════════════════════════════════════════
  // ISOMETRIC 3D BOX (Used for wooden desks)
  // ══════════════════════════════════════════════════════════════════════════
  drawIsometricBox(c, col, row, w, d, h, topColor, leftColor, rightColor) {
    // 4 top corners at elevation h
    const p0 = this.iso(col,     row,     h);
    const p1 = this.iso(col + w, row,     h);
    const p2 = this.iso(col + w, row + d, h);
    const p3 = this.iso(col,     row + d, h);

    // 4 bottom corners at elevation 0
    const b0 = this.iso(col,     row,     0);
    const b1 = this.iso(col + w, row,     0);
    const b2 = this.iso(col + w, row + d, 0);
    const b3 = this.iso(col,     row + d, 0);

    // 1. Left front face
    c.fillStyle = leftColor;
    c.beginPath();
    c.moveTo(p0.x, p0.y);
    c.lineTo(p3.x, p3.y);
    c.lineTo(b3.x, b3.y);
    c.lineTo(b0.x, b0.y);
    c.closePath();
    c.fill();

    // 2. Right front face
    c.fillStyle = rightColor;
    c.beginPath();
    c.moveTo(p3.x, p3.y);
    c.lineTo(p2.x, p2.y);
    c.lineTo(b2.x, b2.y);
    c.lineTo(b3.x, b3.y);
    c.closePath();
    c.fill();

    // 3. Top surface face
    c.fillStyle = topColor;
    c.beginPath();
    c.moveTo(p0.x, p0.y);
    c.lineTo(p1.x, p1.y);
    c.lineTo(p2.x, p2.y);
    c.lineTo(p3.x, p3.y);
    c.closePath();
    c.fill();

    // Top surface perimeter highlights
    c.strokeStyle = '#e2aa66';
    c.lineWidth = 1.2;
    c.beginPath();
    c.moveTo(p0.x, p0.y);
    c.lineTo(p1.x, p1.y);
    c.stroke();
    c.beginPath();
    c.moveTo(p0.x, p0.y);
    c.lineTo(p3.x, p3.y);
    c.stroke();

    // Seams and corners
    c.strokeStyle = '#683d16';
    c.lineWidth = 0.8;
    c.beginPath();
    c.moveTo(p3.x, p3.y);
    c.lineTo(b3.x, b3.y);
    c.lineTo(b2.x, b2.y);
    c.stroke();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // RETRO BEIGE CRT MONITORS
  // ══════════════════════════════════════════════════════════════════════════

  // Monitor facing front (we see the screen!)
  drawRetroCRTMonitor(c, x, y, type, accentColor, isAnimated) {
    c.save();
    const mw = 34, mh = 26;

    // Outer beige case with rounded corners
    c.fillStyle = '#dbcebe';
    c.beginPath();
    c.roundRect(x - mw / 2, y - mh / 2, mw, mh, [3]);
    c.fill();

    // Dark screen bezel
    c.fillStyle = '#262320';
    c.fillRect(x - mw / 2 + 3, y - mh / 2 + 3, mw - 6, mh - 7);

    // Screen area
    const sx = x - mw / 2 + 4.5;
    const sy = y - mh / 2 + 4.5;
    const sw = mw - 9;
    const sh = mh - 10;

    if (type === 'terminal') {
      // Dark green phosphor CRT screen (matching purple-shirt guy's monitor!)
      c.fillStyle = '#061a0b';
      c.fillRect(sx, sy, sw, sh);

      // Monospace code lines
      const codeLines = [
        { w: 14, col: '#22c55e' },
        { w: 20, col: '#4ade80' },
        { w: 16, col: '#22c55e' },
        { w: 10, col: '#86efac' }
      ];
      codeLines.forEach((line, li) => {
        const lw = isAnimated ? ((line.w + (this.tick * 0.5 + li * 4)) % (sw - 4)) : line.w;
        c.fillStyle = line.col;
        c.fillRect(sx + 2, sy + 2 + li * 3.5, Math.min(lw, sw - 4), 1.6);
      });

      // Blinking cursor
      if (this.tick % 24 < 12) {
        c.fillStyle = '#4ade80';
        c.fillRect(sx + 2, sy + sh - 3, 3, 2);
      }
    } else if (type === 'browser') {
      // Colorful web page / document layout (matching purple-suit woman's monitor!)
      c.fillStyle = '#f8fafc';
      c.fillRect(sx, sy, sw, sh);

      // Blue browser title bar
      c.fillStyle = '#2563eb';
      c.fillRect(sx, sy, sw, 3);

      // Mini image thumbnail (green / brown layout)
      c.fillStyle = '#16a34a';
      c.fillRect(sx + 2, sy + 4.5, 9, 7);
      c.fillStyle = '#b45309';
      c.fillRect(sx + 4, sy + 7.5, 5, 4);

      // Document text lines
      c.fillStyle = '#64748b';
      c.fillRect(sx + 13, sy + 5, 9, 1.5);
      c.fillRect(sx + 13, sy + 7.5, 7, 1.5);
      c.fillRect(sx + 2, sy + 12.5, sw - 4, 1.5);
    } else {
      // Idle screen
      c.fillStyle = '#0b0f19';
      c.fillRect(sx, sy, sw, sh);
      c.fillStyle = accentColor || '#10b981';
      c.fillRect(sx + 2, sy + 3, 6, 1.5);
    }

    // Power LED
    c.fillStyle = isAnimated ? '#22c55e' : '#f59e0b';
    c.fillRect(x + mw / 2 - 6, y + mh / 2 - 3, 3, 2);

    // 3D side depth (right side)
    c.fillStyle = '#b7a996';
    c.beginPath();
    c.moveTo(x + mw / 2, y - mh / 2);
    c.lineTo(x + mw / 2 + 8, y - mh / 2 + 5);
    c.lineTo(x + mw / 2 + 8, y + mh / 2 + 5);
    c.lineTo(x + mw / 2, y + mh / 2);
    c.closePath();
    c.fill();

    // 3D top depth
    c.fillStyle = '#ccbeac';
    c.beginPath();
    c.moveTo(x - mw / 2, y - mh / 2);
    c.lineTo(x + mw / 2, y - mh / 2);
    c.lineTo(x + mw / 2 + 8, y - mh / 2 + 5);
    c.lineTo(x - mw / 2 + 8, y - mh / 2 + 5);
    c.closePath();
    c.fill();

    // Heavy CRT pedestal base
    c.fillStyle = '#b7a996';
    c.fillRect(x - 5, y + mh / 2, 10, 4);
    c.fillRect(x - 8, y + mh / 2 + 4, 16, 3);
    c.restore();
  }

  // Monitor viewed from back/side (showing beige rear casing, vents, cables)
  drawRetroCRTMonitorBack(c, x, y) {
    c.save();
    const mw = 32, mh = 25;

    // Rear casing
    c.fillStyle = '#d5c7b5';
    c.beginPath();
    c.roundRect(x - mw / 2, y - mh / 2, mw, mh, [3]);
    c.fill();

    // Ventilation cooling slots
    c.fillStyle = '#a69784';
    for (let i = 0; i < 4; i++) {
      c.fillRect(x - mw / 2 + 6, y - mh / 2 + 5 + i * 3.5, mw - 12, 1.5);
    }

    // Rear power plug socket & model label
    c.fillStyle = '#443d35';
    c.fillRect(x - 4, y + 2, 8, 5);

    // Dark grey cable coming out and running onto desk
    c.strokeStyle = '#2d2822';
    c.lineWidth = 1.5;
    c.beginPath();
    c.moveTo(x, y + 6);
    c.quadraticCurveTo(x + 10, y + 14, x + 16, y + 18);
    c.stroke();

    // Base stand
    c.fillStyle = '#b5a694';
    c.fillRect(x - 6, y + mh / 2, 12, 4);
    c.fillRect(x - 9, y + mh / 2 + 4, 18, 3);
    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // GLOWING BRASS DESK LAMP (Round yellow bell shade with warm glow)
  // ══════════════════════════════════════════════════════════════════════════
  drawGlowingDeskLamp(c, x, y) {
    c.save();
    // Warm radial light glow circle covering desks
    const glow = c.createRadialGradient(x, y + 10, 2, x, y + 10, 75);
    glow.addColorStop(0, 'rgba(255, 235, 120, 0.42)');
    glow.addColorStop(0.5, 'rgba(254, 215, 60, 0.16)');
    glow.addColorStop(1, 'transparent');
    c.fillStyle = glow;
    c.beginPath();
    c.ellipse(x, y + 12, 75, 45, 0, 0, Math.PI * 2);
    c.fill();

    // Round brass base
    c.fillStyle = '#b45309';
    c.beginPath();
    c.ellipse(x, y + 14, 7, 3.5, 0, 0, Math.PI * 2);
    c.fill();

    // Brass stem pole
    c.fillStyle = '#d97706';
    c.fillRect(x - 1.5, y - 6, 3, 20);

    // Glowing bell-shaped yellow lampshade
    c.fillStyle = '#fde047';
    c.beginPath();
    c.moveTo(x, y - 14);
    c.lineTo(x + 10, y - 2);
    c.lineTo(x - 10, y - 2);
    c.closePath();
    c.fill();

    // Lampshade highlight & trim
    c.strokeStyle = '#b45309';
    c.lineWidth = 1;
    c.stroke();

    // Bright light source dot under shade
    c.fillStyle = '#ffffff';
    c.fillRect(x - 2.5, y - 2, 5, 3);
    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // CHARACTERS
  // ══════════════════════════════════════════════════════════════════════════

  // Agent facing forward/SE (Like Gemini in green sweater)
  drawAgentFacingFront(c, ag, x, y, isWork) {
    c.save();
    const bob = isWork ? (this.tick % 6 < 3 ? 1 : 0) : 0;
    const cy = y + bob;

    // Body (green knit sweater)
    c.fillStyle = ag.clothColor;
    c.beginPath();
    c.roundRect(x - 9, cy + 2, 18, 16, [3]);
    c.fill();

    // Collar
    c.fillStyle = this._shade(ag.clothColor, -25);
    c.fillRect(x - 4, cy + 2, 8, 3);

    // Head
    c.fillStyle = ag.skinColor;
    c.beginPath();
    c.roundRect(x - 7, cy - 11, 14, 12, [3]);
    c.fill();

    // Hair (sandy brown with side bangs)
    c.fillStyle = ag.hairColor;
    c.fillRect(x - 8, cy - 15, 16, 6);
    c.fillRect(x - 8, cy - 11, 3, 5); // left sideburn
    c.fillRect(x + 5, cy - 11, 3, 5); // right sideburn
    c.fillRect(x - 4, cy - 17, 8, 3);  // top tuft

    // Eyes looking towards viewer / down-right
    c.fillStyle = '#1c1917';
    c.fillRect(x - 4, cy - 6, 2.5, 2.5);
    c.fillRect(x + 2, cy - 6, 2.5, 2.5);

    // Smile / friendly expression
    c.fillStyle = '#b45309';
    c.fillRect(x - 2, cy - 2, 4, 1.5);

    // Arms resting forward on desk
    c.fillStyle = ag.clothColor;
    c.fillRect(x - 11, cy + 8, 4, 8);
    c.fillRect(x + 7, cy + 8, 4, 8);

    // Hands
    c.fillStyle = ag.skinColor;
    c.fillRect(x - 11, cy + 15, 4, 3);
    c.fillRect(x + 7, cy + 15, 4, 3);

    // Typing sparks when working
    if (isWork && this.tick % 6 === 0) {
      this.sparks.push({
        x: x + (Math.random() * 16 - 8),
        y: cy + 16,
        vx: (Math.random() - 0.5) * 2,
        vy: -Math.random() * 2 - 0.5,
        life: 14,
        color: ag.color
      });
    }

    c.restore();
  }

  // Agent facing away/NW towards screen (Like purple-shirt guy & purple-suit woman)
  drawAgentFacingBack(c, ag, x, y, isWork) {
    c.save();
    const typingBob = isWork ? (this.tick % 4 < 2 ? -1 : 1) : 0;
    const cy = y + typingBob;

    // Shoulders & Back
    c.fillStyle = ag.clothColor;
    c.beginPath();
    c.roundRect(x - 10, cy + 3, 20, 17, [4]);
    c.fill();

    // Crease lines on back of shirt/jacket
    c.strokeStyle = this._shade(ag.clothColor, -25);
    c.lineWidth = 1;
    c.beginPath();
    c.moveTo(x, cy + 5); c.lineTo(x, cy + 17);
    c.moveTo(x - 5, cy + 12); c.lineTo(x + 5, cy + 12);
    c.stroke();

    // Pants / skirt
    c.fillStyle = '#374151';
    c.fillRect(x - 8, cy + 20, 16, 6);

    // Head
    c.fillStyle = ag.skinColor;
    c.beginPath();
    c.roundRect(x - 7, cy - 10, 14, 12, [3]);
    c.fill();

    // Hair
    c.fillStyle = ag.hairColor;
    if (ag.id === 'claude-sonnet-4-6') {
      // Woman with long auburn hair cascading down back past shoulders
      c.fillRect(x - 8, cy - 14, 16, 9);
      // Long hair strands
      c.fillRect(x - 9, cy - 8, 18, 14);
      c.fillRect(x - 7, cy + 6, 14, 6);
    } else {
      // Short black fade hair
      c.fillRect(x - 8, cy - 14, 16, 8);
      c.fillRect(x - 8, cy - 8, 16, 5);
    }

    // Arms typing forward on the keyboard
    c.fillStyle = ag.clothColor;
    if (isWork) {
      const alt = this.tick % 6 < 3;
      c.fillRect(x - 13, cy + 7 + (alt ? 2 : 0), 5, 8);
      c.fillRect(x + 8, cy + 7 + (alt ? 0 : 2), 5, 8);
      c.fillStyle = ag.skinColor;
      c.fillRect(x - 13, cy + 15 + (alt ? 2 : 0), 5, 3);
      c.fillRect(x + 8, cy + 15 + (alt ? 0 : 2), 5, 3);

      // Typing sparks
      if (this.tick % 5 === 0) {
        this.sparks.push({
          x: x + (Math.random() * 16 - 8),
          y: cy + 10,
          vx: (Math.random() - 0.5) * 2,
          vy: -Math.random() * 2 - 0.5,
          life: 14,
          color: ag.color
        });
      }
    } else {
      c.fillRect(x - 12, cy + 8, 4, 8);
      c.fillRect(x + 8, cy + 8, 4, 8);
    }

    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // SWIVEL CHAIR WITH 5-STAR WHEELS
  // ══════════════════════════════════════════════════════════════════════════
  drawSwivelChair(c, x, y, view = 'front') {
    c.save();
    // 1. 5-Star Wheeled Base resting on the parquet floor tiles
    c.strokeStyle = '#18181b';
    c.lineWidth = 2.5;
    const spokes = [
      { dx: -9, dy: -3 },
      { dx: 9,  dy: -3 },
      { dx: -6, dy: 5 },
      { dx: 6,  dy: 5 },
      { dx: 0,  dy: 7 }
    ];
    spokes.forEach(sp => {
      c.beginPath();
      c.moveTo(x, y + 2);
      c.lineTo(x + sp.dx, y + 2 + sp.dy);
      c.stroke();
      // Tiny black caster wheel
      c.fillStyle = '#09090b';
      c.fillRect(x + sp.dx - 1.5, y + 2 + sp.dy - 1, 3, 3);
    });

    // Central hydraulic lift column
    c.fillStyle = '#52525b';
    c.fillRect(x - 2, y - 6, 4, 8);

    // Thick black seat cushion
    c.fillStyle = '#1c1917';
    c.beginPath();
    c.roundRect(x - 11, y - 11, 22, 7, [3]);
    c.fill();

    // Ergonomic backrest
    c.fillStyle = '#262626';
    c.beginPath();
    c.roundRect(x - 10, y - 25, 20, 14, [4]);
    c.fill();

    // Lumbar contour highlight
    c.strokeStyle = '#404040';
    c.lineWidth = 1;
    c.stroke();

    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // KEYBOARDS & MOUSE
  // ══════════════════════════════════════════════════════════════════════════
  drawKeyboard(c, x, y, isWork, glowColor) {
    c.save();
    c.fillStyle = '#dcd4c6';
    c.beginPath();
    c.roundRect(x - 12, y - 4, 24, 8, [1.5]);
    c.fill();

    // Keys
    for (let r = 0; r < 2; r++) {
      for (let k = 0; k < 6; k++) {
        c.fillStyle = isWork && (this.tick + k + r) % 5 < 2 ? (glowColor || '#22c55e') : '#52525b';
        c.fillRect(x - 10 + k * 3.4, y - 2.5 + r * 3.2, 2.4, 1.8);
      }
    }
    c.restore();
  }

  drawKeyboardAndMouse(c, x, y, isWork, glowColor) {
    this.drawKeyboard(c, x, y, isWork, glowColor);

    // Mouse with curly black wire running across desk
    c.save();
    c.fillStyle = '#ece5d8';
    c.beginPath();
    c.roundRect(x + 15, y - 3, 6, 7, [2]);
    c.fill();

    // Curly cable
    c.strokeStyle = '#262626';
    c.lineWidth = 1;
    c.beginPath();
    c.moveTo(x + 18, y - 3);
    c.bezierCurveTo(x + 22, y - 9, x + 10, y - 11, x + 14, y - 15);
    c.stroke();
    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // SPEECH BUBBLE — Crisp white bubble matching reference image
  // ══════════════════════════════════════════════════════════════════════════
  drawSpeechBubble(c, x, y, text) {
    if (!text) return;
    let displayText = String(text).trim();
    if (displayText.length > 34) {
      displayText = displayText.slice(0, 31) + '...';
    }

    c.save();
    const floatY = y + Math.sin(this.tick * 0.08) * 1.5;

    c.font = '600 9px "Outfit",-apple-system,sans-serif';
    c.textAlign = 'center';
    const textWidth = c.measureText(displayText).width;
    const bw = Math.max(textWidth + 24, 75);
    const bh = 24;
    const bx = x - bw / 2;

    // Drop shadow
    c.shadowColor = 'rgba(0, 0, 0, 0.35)';
    c.shadowBlur = 8;
    c.shadowOffsetY = 3;

    // White bubble container
    c.fillStyle = '#ffffff';
    c.strokeStyle = '#1e293b';
    c.lineWidth = 1.6;
    c.beginPath();
    c.roundRect(bx, floatY, bw, bh, [6]);
    c.fill();
    c.stroke();

    // Downward pointing pointer tail
    c.beginPath();
    c.moveTo(x - 5, floatY + bh - 0.5);
    c.lineTo(x - 12, floatY + bh + 8);
    c.lineTo(x + 3, floatY + bh - 0.5);
    c.closePath();
    c.fill();
    c.stroke();

    // Remove shadow for clean text
    c.shadowColor = 'transparent';
    c.fillStyle = '#0f172a';
    c.fillText(displayText, x, floatY + 15.5);

    c.restore();
  }

  // ══════════════════════════════════════════════════════════════════════════
  // AMBIENT EFFECTS — Floating dust motes & typing sparks
  // ══════════════════════════════════════════════════════════════════════════
  drawParticles(c) {
    c.save();
    // Dust particles in sunlight
    this.dust.forEach(m => {
      m.x += m.vx;
      m.y += m.vy;
      if (m.x > 620) m.x = 40;
      if (m.y > 480) m.y = 50;

      const alpha = m.a * (0.5 + Math.sin(this.tick * 0.05 + m.phase) * 0.5);
      c.fillStyle = `rgba(255, 238, 170, ${alpha})`;
      c.beginPath();
      c.arc(m.x, m.y, m.r, 0, Math.PI * 2);
      c.fill();
    });

    // Keyboard sparks
    for (let i = this.sparks.length - 1; i >= 0; i--) {
      const s = this.sparks[i];
      s.x += s.vx;
      s.y += s.vy;
      s.life--;
      if (s.life <= 0) {
        this.sparks.splice(i, 1);
        continue;
      }
      c.globalAlpha = s.life / 14;
      c.fillStyle = s.color;
      c.fillRect(s.x, s.y, 2.5, 2.5);
    }
    c.restore();
  }

  drawZzz(c, x, y) {
    c.save();
    for (let i = 0; i < 3; i++) {
      const offset = (this.tick * 0.6 + i * 20) % 55;
      c.globalAlpha = Math.max(0, 1 - offset / 55);
      c.font = `${5 + i * 2}px "Press Start 2P",monospace`;
      c.fillStyle = '#a78bfa';
      c.textAlign = 'left';
      c.fillText('Z', x + Math.sin((offset + i * 10) * 0.1) * 4 + i * 3, y - offset);
    }
    c.restore();
  }

  drawDeskLabel(c, x, y, label, occupied) {
    c.save();
    c.font = '7px "Press Start 2P",monospace';
    c.textAlign = 'center';
    const tw = c.measureText(label).width;
    const bw = tw + 18, bh = 15;
    c.fillStyle = occupied ? 'rgba(15, 23, 42, 0.90)' : 'rgba(15, 23, 42, 0.60)';
    c.strokeStyle = occupied ? '#f59e0b' : '#64748b';
    c.lineWidth = 1;
    c.beginPath();
    c.roundRect(x - bw / 2, y - bh / 2, bw, bh, [3]);
    c.fill();
    c.stroke();
    c.fillStyle = occupied ? '#fef08a' : '#94a3b8';
    c.fillText(label, x, y + 3.5);
    c.restore();
  }

  _shade(hex, amt) {
    let r = parseInt(hex.slice(1, 3), 16) + amt;
    let g = parseInt(hex.slice(3, 5), 16) + amt;
    let b = parseInt(hex.slice(5, 7), 16) + amt;
    return `rgb(${Math.max(0, Math.min(255, r))},${Math.max(0, Math.min(255, g))},${Math.max(0, Math.min(255, b))})`;
  }
}

window.addEventListener('DOMContentLoaded', () => {
  window.pixelOffice = new PixelOffice('pixel-office-canvas');
  window.pixelOffice.renderRosterCards();
});
