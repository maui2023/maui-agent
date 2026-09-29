# miqa (AntiGravity AI Coding Agent via CLI, WebUI & Telegram)

```
███╗   ███╗██╗ ██████╗  █████╗ 
████╗ ████║██║██╔═══██╗██╔══██╗
██╔████╔██║██║██║   ██║███████║
██║╚██╔╝██║██║██║▄▄ ██║██╔══██║
██║ ╚═╝ ██║██║╚██████╔╝██║  ██║
╚═╝     ╚═╝╚═╝ ╚══▀▀═╝ ╚═╝  ╚═╝
```

**miqa** ialah sistem ejen CLI & WebUI pintar berasaskan **Go (Golang)** yang direka untuk memudahkan pembangun perisian menguruskan sesi pengekodan AI, memantau output siaran langsung `agy`, mengkonfigurasi bot Telegram dua hala, memilih model AI secara dinamik, dan memantau penggunaan sistem & token secara visual melalui **Animated Pixel Office**.

Sistem ini dilengkapi automasi permulaan dengan mod pintasan kebenaran lanjutan (`--dangerously-skip-permissions`) untuk mempercepatkan automasi tugas pembangunan tanpa gangguan prompt manual.

---

## 🚀 Ciri-Ciri Utama (Key Features)

* ⚡ **Enjin Pantas & Ringan:** Dibina sepenuhnya dalam bahasa Go dengan binari tunggal yang pantas dan mudah alih.
* 🛡️ **Pintasan Keselamatan Lanjutan:** Automasi pelaksanaan arahan melalui bendera `--dangerously-skip-permissions`.
* 🏢 **Animated Pixel Office (WebUI):**
  * Lantai pejabat pixel-art beranimasi untuk ejen-ejen AI.
  * **Model Bekerja:** Menaip pantas di meja dengan percikan kod dan lampu monitor bercahaya.
  * **Model Tidak Digunakan:** Bersantai minum kopi atau tidur lena dengan animasi `Zzz...` terapung.
  * **Model Habis Kredit/Kuota:** Keadaan letih/lapar terlentap di atas meja dengan belon fikiran makanan (🍕 Pizza, 🍜 Ramen) dan ikon bateri 0%!
* 📊 **Telemetri & Graf Sistem Langsung:**
  * Graf gelombang neon CPU Load Avg masa nyata.
  * Meter bulatan memori RAM (digunakan vs jumlah).
  * Bar penggunaan storan cakera (bebas vs jumlah).
  * Statistik sesi dan bilangan langkah (steps) Antigravity.
* 💻 **Terminal WebUI Langsung (Live Agent Output):**
  * Paparan output `agy` masa nyata dengan togol scanline CRT, auto-scroll, dan bar arahan pantas.
* 🤖 **Integrasi Telegram Bot Interaktif:**
  * Papan Pemuka interaktif (Inline Keyboards) dengan arahan `/menu`, `/model`, `/usage`, `/path`, `/status`, `/web`, `/cancel`.
  * Penulisan kod terus melalui sembang Telegram.
* 💻 **Papan Pemuka Terminal (TUI Dashboard):** Menggunakan `lipgloss` dan `survey` untuk terminal moden.
* 📜 **Sokongan Bantuan CLI Penuh:** Menyokong arahan `miqa --help` dan `miqa -h`.

---

## 🛠️ Aliran Kerja Pengguna (User Flow)

```
[ Pengguna Menjalankan CLI: miqa ]
       │
       ▼
[ Jalankan & Sahkan agy --dangerously-skip-permissions ]
       │
       ▼
[ Semak Status Log Masuk (Check Login Status) ]
       ├── [ Belum Log Masuk ] ──► [ Paparkan Arahan Log Masuk ]
       └── [ Sudah Log Masuk ] 
               │
               ▼
       [ Minta Input Telegram Bot Token / Chat ID ]
               │
               ▼
       [ Paparkan Menu Utama (Interactive Dashboard) ]
               ├── 1. Pilih Model AI (Select Model)
               ├── 2. Semak Penggunaan (Check Usage)
               ├── 3. Tetapkan Laluan Pengekodan (Select Path for Coding)
               ├── 4. [MULA/HENTI] Pelayan WebUI (Latar Belakang)
               ├── 5. [MULA/HENTI] Bot Telegram (Latar Belakang)
               ├── 6. Jalankan Arahan Kod Terus (Run Prompt in CLI)
               └── 7. Konfigurasi Telegram Bot (Setup Telegram)
```

---

## 📦 Pemasangan & Kompilasi (Installation)

### Prasyarat
* Go 1.22+ (atau terbaharu)
* Antigravity CLI (`agy`) dipasang pada sistem

### Bina & Pasang

```bash
# Klon repositori
git clone https://github.com/maui2023/maui-agent.git
cd maui-agent

# Bina binari
make build

# Pasang ke ~/.local/bin/miqa
make install
```

---

## 💻 Penggunaan CLI (CLI Usage)

### Arahan Bantuan
```bash
miqa --help
# atau
miqa -h
```

### Jalankan Papan Pemuka Interaktif Penuh
```bash
miqa
```

### Mulakan WebUI (Animated Pixel Office & Live Terminal)
```bash
miqa web
# atau tentukan port pilihan:
miqa web --port 8080
```
Buka penyemak imbas anda di: `http://localhost:8080`

### Mulakan Perkhidmatan Bot Telegram Secara Terus
```bash
miqa bot
```

### Semak Senarai Model AI yang Tersedia
```bash
miqa models
```

### Semak Statistik Penggunaan & Sumber Sistem
```bash
miqa usage
```

### Laksanakan Arahan Pengekodan Terus Melalui CLI
```bash
miqa run "Buat fungsi fibonacci dalam Go dan simpan ke file fib.go"
```

---

## 🌐 WebUI: Animated Pixel Office & Terminal

WebUI membolehkan anda melihat pejabat ejen AI beraksi secara visual:
1. **Lantai Pejabat Pixel Art:**
   * Setiap model AI mempunyai meja kerja, monitor berkembar, dan kerusi berputar.
   * **Sedang Bekerja:** Tangan menaip pantas, skrin kod matrix berkedip, percikan api papan kekunci, dan mug kopi berasap.
   * **Santai / Tidur:** Bersantai mengayun kerusi atau tidur lena dengan huruf `Zzz` terapung.
   * **Habis Kredit (Letih/Lapar):** Rebah muka di atas meja, peluh mengalir, dan belon fikiran makanan berpusing (🍕 Pizza / 🍜 Ramen / ⚡ 0% Bateri).
   * **Butang Simulasi:** Terdapat butang pantas `⚡ Kerja`, `🍕 Habis Kredit`, `💤 Tidur`, dan `🔄 Asal` untuk menguji animasi secara langsung.
2. **Graf Sistem:**
   * Gelombang beban CPU, tolok RAM, dan storan bebas dikemaskini setiap 2 saat melalui SSE (*Server-Sent Events*).
3. **Terminal Langsung:**
   * Memaparkan output perlaksanaan `agy` serta merta. Anda boleh menghantar arahan terus dari kotak teks atau butang cadangan prompt.

---

## 📱 Menu & Arahan Telegram Bot

| Arahan Telegram | Penerangan |
| :--- | :--- |
| `/start` atau `/menu` | Membuka Papan Pemuka Utama dengan butang inline interaktif |
| `/model` | Memaparkan senarai model AI dan menukar model aktif secara langsung |
| `/usage` | Memaparkan ringkasan sesi Antigravity, bilangan langkah, RAM, CPU & storan |
| `/path` | Melihat laluan folder kerja pengekodan semasa |
| `/path <folder>` | Menukar folder kerja pengekodan ke laluan baharu |
| `/status` | Melihat status kesihatan bot, model aktif, laluan kerja & uptime |
| `/web` | Mendapatkan pautan WebUI Pixel Office |
| `/cancel` | Menghentikan arahan pengekodan agy yang sedang berlangsung |
| `/help` | Menampilkan panduan arahan lengkap |

---

## ⚙️ Konfigurasi (Configuration)

Konfigurasi disimpan secara automatik dalam fail JSON:
`~/.config/miqa/config.json`

Contoh:
```json
{
  "telegram_token": "123456789:ABCDefGhIJKlmNoPQRsTUVwxyZ",
  "telegram_chat_id": 987654321,
  "model": "gemini-3.8-flash-high",
  "coding_path": "/home/maui/github/maui-agent",
  "dangerously_skip_permissions": true
}
```

---

## 🧪 Menjalankan Ujian Unit (Testing)

```bash
make test
# atau
go test -v ./...
```
