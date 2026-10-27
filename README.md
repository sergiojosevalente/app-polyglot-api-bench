# Lab Perbandingan Bahasa: Go · Rust · PHP · Node.js · Java · Python

Satu REST API kecil yang ditulis **enam kali dengan perilaku identik**, ditambah **monitor** berisi load generator, stress test, contract test, dan dashboard live untuk membandingkan semuanya secara berdampingan.

> Materi belajar konsep (struktur, error handling, model concurrency, memori, serta kelebihan dan kekurangan tiap bahasa) ada di **[LEARNING.md](LEARNING.md)**.

| Service | Stack | Model eksekusi | Port |
|---|---|---|---|
| `go-service` | Go 1.25, **stdlib saja** | goroutine M:N preemptive + GC | 8081 |
| `rust-service` | Rust, axum + tokio + serde | async task cooperative, tanpa GC | 8082 |
| `php-service` | PHP 8.4, nginx + PHP-FPM, opcache JIT, APCu | **proses per request**, shared-nothing | 8083 |
| `node-service` | Node.js 24, TypeScript, `node:http` (0 dependency) | **satu thread** event loop + GC | 8084 |
| `java-service` | Java 21, Spring Boot 3.5, Tomcat | **virtual threads** + JIT + GC | 8085 |
| `python-service` | Python 3.13, FastAPI + Pydantic, uvicorn (uvloop) | asyncio event loop + threadpool, **GIL**, interpreter | 8086 |
| `monitor` | Go | load generator + dashboard | **8080** |

```
GO-RUST/
├── docker-compose.yml        # 7 container, batas CPU/RAM identik (x-limits)
├── .env.example              # CPU_LIMIT, MEM_LIMIT, PHP_WORKERS, PHP_JIT, port
├── go-service/               # cmd/server + internal/{handler,compute,store,metrics}
├── rust-service/src/         # main, handlers, error, compute, store, metrics
├── php-service/              # public/index.php (front controller) + src/ (PSR-4) + docker/ (nginx, fpm, ini)
├── node-service/src/         # main, app, handlers, http/{router,errors}, compute, store, metrics
├── java-service/src/main/... # api/ (controller, DTO, exception handler), domain/, store/, metrics/
├── python-service/app/       # main (app factory + error handler), api, schemas (Pydantic), compute, store, metrics
└── monitor/                  # loadtest/, collector/, verify/, hub/ (SSE), web/index.html
```

Lapisannya sama di semua bahasa: **entry point (wiring) → lapisan HTTP → business logic murni (`compute`) → storage (`store`)**, ditambah `metrics`. Buka file berpasangan secara berdampingan untuk membandingkan idiomnya.

## Rekomendasi: Docker atau install?

**Rekomendasi: jalankan semuanya dengan Docker** (sudah disiapkan). Di server tidak perlu menginstal Rust, PHP, Node, Java, atau Python; cukup Docker.

| | Docker (direkomendasikan untuk menjalankan & benchmark) | Install toolchain lokal (opsional, untuk menulis kode) |
|---|---|---|
| Keamanan & kebersihan server | Tidak ada yang terpasang di host; hapus dengan `docker compose down --rmi all` | Banyak toolchain dengan versi yang bisa saling bentrok |
| Keadilan benchmark | **Batas CPU & RAM identik**, OS sama (Linux) | Proses bersaing dengan aplikasi lain |
| Reproducible | Versi dikunci di Dockerfile + lockfile | Bergantung pada versi yang terpasang |
| Feedback saat coding | Lambat (rebuild image) | Cepat, autocomplete IDE |

- Go dan Rust menghasilkan **binary native**: image akhir hanya berisi binary (8 MB / 25 MB). PHP, Node, Java, dan Python membutuhkan **runtime** (interpreter/VM) di dalam image, sehingga image-nya jauh lebih besar (100–230 MB).
- Semua container berjalan sebagai user **non-root**.
- Untuk belajar menulis kode, install toolchain di **laptop dev**, bukan server: rustup (Rust), Node 24 (TypeScript langsung jalan), JDK 21 + IntelliJ (Java), PHP 8.4 + Composer, Python 3.13 + virtualenv (atau `uv`).
- Docker Desktop di Windows berjalan di VM WSL2. **Yang bermakna adalah perbandingan relatif**, bukan angka absolut.

## Quick start

```bash
docker compose up -d --build
```

Build pertama memakan waktu beberapa menit (kompilasi Rust dan download dependency Maven paling lama). Waktu build ini juga merupakan salah satu poin perbandingan. Setelah selesai, buka **<http://localhost:8080>**.

Langkah pertama: klik **"✓ Verifikasi kesamaan fungsi"**. Hasilnya harus `18/18 kasus identik` di keenam service, yang membuktikan semua implementasi setara sebelum dibandingkan.

```bash
docker compose logs -f php-api java-api
```

```bash
docker compose down
```

## Endpoint (identik di semua service)

| Method | Path | Fungsi | Yang diuji |
|---|---|---|---|
| GET | `/health` | status | – |
| GET | `/api/hello?name=X` | `{"message":"Hello, X!"}` | overhead HTTP stack |
| POST | `/api/json` | body `{"numbers":[...]}` → count/sum/min/max/mean | JSON parse + serialize |
| GET | `/api/cpu?n=25` | fibonacci rekursif (n ≤ 35) | kecepatan CPU murni (AOT vs JIT) |
| GET | `/api/items/{id}` | ambil item (1–1000 sudah di-seed) | shared state |
| PUT | `/api/items/{id}` | body `{"name":"..","price":1.5}` → upsert | penulisan concurrent |
| GET | `/metrics` | memori, CPU-detik, threads, info runtime | dipakai monitor |

Semua error memakai format `{"error": "..."}` dengan status code yang sama.

## Cara memakai dashboard

1. **Pilih target**: default keenam bahasa tercentang; hilangkan centang untuk membandingkan sebagian saja.
2. **Skenario**: `hello`, `json`, `cpu`, `store`, `mixed`. Penjelasan setiap skenario muncul di bawah dropdown.
3. **Mode**: `load` (VU konstan), `stress` (naik 10% → 100% untuk mencari titik jenuh), atau `spike` (lonjakan mendadak).
4. **Start**. Grafik live diperbarui setiap detik: throughput, latency p50/p99, error rate, CPU, memori, thread, dan waktu respons `/metrics`.
5. Setelah selesai, muncul **tabel perbandingan** (★ = terbaik per metrik) dan **kesimpulan otomatis** ("X terbaik di throughput: ×1,7 vs Y, ×4,2 vs Z").

Kartu tiap bahasa menampilkan info runtime khasnya: goroutine dan GC (Go), tokio worker (Rust), jumlah FPM worker dan status JIT (PHP), **event loop delay** (Node), heap dan jenis GC (Java), serta status **GIL**, thread, dan asyncio task (Python).

## Eksperimen yang disarankan

| # | Setting | Pertanyaan yang dijawab |
|---|---|---|
| 1 | Semua target, `hello`, load, 50 VU | Seberapa besar overhead dasar tiap HTTP stack? |
| 2 | Java saja, `hello`, load 60 s | **JIT warm-up**: throughput Java naik di detik-detik awal |
| 3 | PHP vs Node, `cpu`, n=27 | JIT PHP vs V8. Ulangi dengan `PHP_JIT=disable` di `.env` |
| 4 | Node, `cpu`, n=30, 20 VU | **Event loop terblokir**: lihat event loop delay & p99 |
| 5 | Semua, `store`, stress, 300 VU | Lock (Go/Rust) vs ConcurrentHashMap (Java) vs single-thread (Node) vs APCu (PHP) |
| 6 | PHP, `mixed`, stress, 400 VU | Apa yang terjadi saat VU > `PHP_WORKERS`? (antrean FPM) |
| 7 | Diamkan 1 menit tanpa test | Memori idle: binary native vs VM/runtime |
| 8 | `.env`: `MEM_LIMIT=96m`, lalu stress | Siapa yang kena OOM-kill lebih dulu? |
| 9 | `.env`: `CPU_LIMIT=2`, Node vs Go | Node tetap ~100% CPU (1 thread); Go/Java memakai 2 core |
| 10 | Python vs Node, `cpu`, n=25, 20 VU | Interpreter (CPython) vs JIT (V8). Bandingkan dengan `hello`: selisihnya mengecil saat beban bukan CPU |
| 11 | `.env`: `CPU_LIMIT=2`, Python saja, `cpu` | **GIL**: walau `def` berjalan di threadpool, CPU Python tetap ~100% |

Bandingkan juga ukuran image dan waktu build:

```bash
docker images gorust/*
```

## Menjalankan unit test (semua di Docker, tanpa install)

Go (sudah terinstal di host):

```bash
cd go-service && go test ./...
```

Rust:

```bash
docker run --rm -v "${PWD}/rust-service:/app" -w /app rust:1-slim-bookworm cargo test
```

Node.js (type-check TypeScript + `node:test`):

```bash
docker run --rm -v "${PWD}/node-service:/src:ro" -w /work node:24-alpine sh -c "cp -r /src/. . && npm install && npm run typecheck && npm test"
```

Java (JUnit 5):

```bash
docker run --rm -v "${PWD}/java-service:/src:ro" -w /work maven:3.9-eclipse-temurin-21 sh -c "cp -r /src/. . && mvn -B test"
```

PHP (PHPUnit):

```bash
docker run --rm -v "${PWD}/php-service:/src:ro" -w /work composer:2 sh -c "cp -r /src/. . && composer install --ignore-platform-req=ext-apcu && vendor/bin/phpunit"
```

Python (pytest + TestClient FastAPI):

```bash
docker run --rm -v "${PWD}/python-service:/src:ro" -w /work python:3.13-slim sh -c "cp -r /src/. . && pip install -q -r requirements-dev.txt && python -m pytest -q"
```

## Batasan yang perlu diingat

- Semua service dan load generator berbagi CPU host yang sama. Menguji 6 target sekaligus tetap valid untuk perbandingan relatif, tetapi untuk angka yang presisi uji 2–3 target per test (dashboard memberi peringatan).
- Memori PHP adalah total PSS **seluruh proses** di container (nginx + master + worker), karena PHP-FPM memang multi-proses. Bahasa lain diukur per satu proses.
- `/metrics` PHP dilayani pool FPM terpisah (`[ops]`) supaya tetap responsif saat worker utama penuh. Ini memang praktik yang baik di produksi.
- Resolusi CPU dari `/proc` adalah 10 ms, sehingga grafik CPU per detik sedikit bergerigi.
- Ini microbenchmark. Di aplikasi nyata, database dan jaringan biasanya lebih dominan daripada bahasa.
