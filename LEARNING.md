# Belajar Rust dari sudut pandang Go

Panduan ini memetakan konsep yang sudah Anda kuasai di Go ke padanannya di Rust, dengan contoh **dari kode di repo ini**. Buka file Go dan Rust yang berpasangan berdampingan saat membaca.

| Go | Rust |
|---|---|
| [cmd/server/main.go](go-service/cmd/server/main.go) | [src/main.rs](rust-service/src/main.rs) |
| [internal/handler/handler.go](go-service/internal/handler/handler.go) | [src/handlers.rs](rust-service/src/handlers.rs) + [src/error.rs](rust-service/src/error.rs) |
| [internal/compute/compute.go](go-service/internal/compute/compute.go) | [src/compute.rs](rust-service/src/compute.rs) |
| [internal/store/store.go](go-service/internal/store/store.go) | [src/store.rs](rust-service/src/store.rs) |
| [internal/metrics/metrics.go](go-service/internal/metrics/metrics.go) | [src/metrics.rs](rust-service/src/metrics.rs) |

---

## 1. Struktur project

| | Go | Rust |
|---|---|---|
| Unit modul | package = folder | module = file (`mod store;` di `main.rs`) |
| Visibility | Huruf besar = exported | `pub` eksplisit; default private |
| Manifest | `go.mod` (+ `go.sum`) | `Cargo.toml` (+ `Cargo.lock`) |
| Layout umum | `cmd/<app>/main.go` + `internal/...` | `src/main.rs` (binary) / `src/lib.rs` (library) + modul |
| Private ke project | folder `internal/` | `pub(crate)` atau tanpa `pub` |
| Dependency | stdlib sangat lengkap: **0 dependency** di go-service | std minimal: HTTP, async runtime, JSON dari crate (`axum`, `tokio`, `serde`) |

Polanya sama di kedua bahasa dan merupakan *best practice* yang bisa dipakai di mana saja:
`main` hanya *wiring* → lapisan HTTP (parse/validasi/response) → business logic murni yang tidak tahu HTTP → storage di belakang API kecil.

## 2. Error handling

**Go**: error adalah nilai, dicek manual:
```go
item, err := h.store.Get(id)
if errors.Is(err, store.ErrNotFound) { writeError(w, 404, ...); return }
```

**Rust**: `Result<T, E>` / `Option<T>` dengan operator `?`:
```rust
let Path(id) = path?;   // kalau gagal: otomatis return Err(AppError) lewat From<PathRejection>
state.store.get(id).map(Json).ok_or_else(|| AppError::NotFound(...))
```

- Di Rust, **compiler memaksa** Anda menangani error. Tidak bisa lupa `if err != nil`, dan tidak ada nil pointer.
- `?` ≈ `if err != nil { return err }`, tapi konversi tipenya otomatis lewat trait `From` (lihat `error.rs`).
- `IntoResponse` memusatkan pemetaan error → HTTP. Di Go setiap handler memanggil `writeError` sendiri.
- Go `ErrNotFound` (sentinel) ≈ Rust `Option::None` atau varian `enum`.

## 3. JSON & struct

| Go | Rust (serde) |
|---|---|
| struct tag `` `json:"name"` `` | `#[derive(Serialize, Deserialize)]`, nama field langsung dipakai |
| reflection saat **runtime** | kode di-generate saat **compile** (biasanya lebih cepat, lihat skenario `json`) |
| embedding `compute.Stats` di struct | `#[serde(flatten)] stats: Stats` |
| field hilang → zero value diam-diam | field hilang → **error**, kecuali bertipe `Option<T>` |

Contoh nyata di repo ini: agar Go menolak `{"name":"x"}` tanpa `price` (sama seperti Rust), Go harus memakai `Price *float64` lalu cek `nil`. Rust cukup `price: f64`.

## 4. Shared state & concurrency safety

**Go** ([store.go](go-service/internal/store/store.go)): mutex *bersebelahan* dengan data. Lupa lock tetap compile, lalu muncul data race yang hanya ketahuan lewat `go test -race`.
```go
type Store struct { mu sync.RWMutex; items map[uint64]Item }
```

**Rust** ([store.rs](rust-service/src/store.rs)): data *di dalam* lock. Mustahil menyentuh map tanpa lock, dan ini dicek compiler.
```rust
pub struct Store { items: RwLock<HashMap<u64, Item>> }
```

- `Arc<T>` = pointer reference-counted yang aman dibagi antar thread (di Go: pointer biasa + GC).
- Trait `Send`/`Sync` membuat compiler menolak program yang berpotensi data race. Inilah "fearless concurrency".
- `AtomicU64` di Rust ≈ `atomic.Uint64` di Go.

## 5. Ownership & memory (konsep paling baru bagi programmer Go)

| | Go | Rust |
|---|---|---|
| Pembersihan memori | Garbage collector (concurrent, berhenti sebentar) | Ownership: memori dilepas tepat saat pemilik keluar scope |
| Menyalin vs memindahkan | value disalin, pointer dibagi bebas | **move** secara default; `&` = pinjam (borrow); `.clone()` eksplisit |
| Biaya | ringan untuk programmer, ada overhead CPU/RAM untuk GC | nol overhead runtime, tapi kurva belajar borrow checker |

Contoh di `put_item`: `Item { id, name: req.name, ... }` **memindahkan** `String` tanpa menyalin. Setelah itu `req.name` tidak bisa dipakai lagi, dan compiler menjamin itu.

Di dashboard, efeknya terlihat di **Memory RSS**: Go menyisakan ruang untuk GC (diatur `GOGC`, default heap boleh tumbuh 2×), sedangkan Rust biasanya lebih rendah dan stabil.

## 6. Concurrency model: goroutine vs async/await

| | Go | Rust (tokio) |
|---|---|---|
| Unit | goroutine (`go f()`) | task (`tokio::spawn`, `async fn`) |
| Runtime | bawaan bahasa, selalu ada | library; diaktifkan dengan `#[tokio::main]` |
| Scheduling | **preemptive**: runtime boleh menyela goroutine kapan saja | **cooperative**: task hanya melepas thread di `.await` |
| Blocking I/O / CPU berat | aman, runtime menambah thread | harus `spawn_blocking`, kalau tidak worker thread "beku" |
| "Function coloring" | tidak ada | fungsi `async` vs biasa itu berbeda |

Coba skenario **cpu** dengan n=30 dan VU tinggi, lalu amati grafik **waktu respons /metrics**. Di Rust, `/metrics` harus antre di belakang task fibonacci karena tokio tidak menyela. Komentar di `handlers.rs::cpu` menjelaskan solusinya.

Graceful shutdown juga mirip: `signal.NotifyContext` + `srv.Shutdown` di Go ≈ `with_graceful_shutdown(shutdown_signal())` + `tokio::select!` di Rust.

## 7. Routing & handler

```go
mux.HandleFunc("GET /api/items/{id}", h.getItem)       // Go 1.22+
id := r.PathValue("id")                                  // parse manual
```
```rust
.route("/api/items/{id}", get(get_item).put(put_item))   // axum 0.8
async fn get_item(State(s): State<AppState>, Path(id): Path<u64>) // diekstrak & di-parse otomatis
```
Di Rust, **tipe argumen handler menentukan apa yang diambil dari request** (extractor). Kalau parse gagal, handler tidak pernah dipanggil.

## 8. Testing

| Go | Rust |
|---|---|
| file `_test.go` terpisah, package sama | `#[cfg(test)] mod tests` di file yang sama |
| `t.Errorf`, table-driven test | `assert_eq!`, loop atas array tuple |
| `go test -race ./...` | `cargo test` (race dicegah compiler, bukan dideteksi) |

## 9. Build & deploy

| | Go | Rust |
|---|---|---|
| Waktu compile | sangat cepat (detik) | lambat (menit pertama kali; LTO menambah) |
| Binary | static murni (`CGO_ENABLED=0`) → image `distroless/static` | link ke glibc → `distroless/cc` (atau target musl untuk static) |
| Cross-compile | `GOOS=linux GOARCH=arm64 go build` | `rustup target add ...` + linker; lebih repot |
| Cache build di Docker | `go mod download` | trik `main.rs` kosong agar layer dependency ter-cache |

---

## Kelebihan & kekurangan: ringkasan jujur

### Go unggul di
- **Produktivitas & kesederhanaan**: bahasa kecil, satu cara melakukan sesuatu, mudah dibaca tim.
- **Compile sangat cepat**, sehingga feedback loop cepat.
- **Stdlib lengkap**: server HTTP produksi tanpa dependency.
- **Goroutine preemptive**: kode blocking pun aman, tidak perlu memikirkan `async`.
- Onboarding programmer baru dalam hitungan hari.

### Go lebih lemah di
- **GC**: memori lebih besar dan jeda kecil yang terlihat di tail latency (p99/max) saat alokasi tinggi.
- **Nil & zero value**: error saat runtime yang di Rust tertangkap saat compile.
- Data race hanya terdeteksi saat runtime (`-race`).
- Kontrol low-level terbatas (layout memori, SIMD, tanpa runtime).

### Rust unggul di
- **Performa & efisiensi**: tanpa GC, optimasi LLVM, zero-cost abstraction. Biasanya memori lebih kecil dan p99 lebih stabil.
- **Keamanan saat compile**: tanpa null, tanpa data race, error wajib ditangani.
- Sistem tipe ekspresif (`enum` dengan data, `match`, trait, generics).
- Cocok untuk area yang menuntut efisiensi maksimal: proxy, database, embedded, WASM, hot path.

### Rust lebih lemah di
- **Kurva belajar curam**: ownership, lifetime, borrow checker, async + `Send`/`Sync`.
- **Compile lambat**, terasa saat iterasi.
- Ekosistem async terfragmentasi (wajib pilih runtime) dan ada jebakan seperti CPU blocking di tokio.
- Lebih banyak keputusan yang harus diambil (crate apa, `Arc` vs `Rc`, `String` vs `&str`).

### Kapan memilih yang mana?
- **Go**: microservice / API bisnis, tooling DevOps & CLI, tim besar, kecepatan delivery lebih penting dari efisiensi ekstrem.
- **Rust**: komponen performance-critical atau hemat resource (biaya cloud, edge, embedded), latency p99 harus stabil, keamanan memori kritis.
- Banyak perusahaan memakai **keduanya**: Go untuk sebagian besar service, Rust untuk hot path tertentu.

---

# Bagian 2: PHP, Node.js, Java, dan Python dibanding Go & Rust

| Lapisan | PHP | Node.js (TypeScript) | Java (Spring Boot) | Python (FastAPI) |
|---|---|---|---|---|
| Entry point | [public/index.php](php-service/public/index.php) → [src/App.php](php-service/src/App.php) | [src/main.ts](node-service/src/main.ts) + [src/app.ts](node-service/src/app.ts) | [JavaServiceApplication.java](java-service/src/main/java/com/gorust/javaservice/JavaServiceApplication.java) | [app/main.py](python-service/app/main.py) |
| HTTP | [src/Handler.php](php-service/src/Handler.php), [src/Http/](php-service/src/Http) | [src/handlers.ts](node-service/src/handlers.ts), [src/http/](node-service/src/http) | [api/ApiController.java](java-service/src/main/java/com/gorust/javaservice/api/ApiController.java), [api/ApiExceptionHandler.java](java-service/src/main/java/com/gorust/javaservice/api/ApiExceptionHandler.java) | [app/api.py](python-service/app/api.py), [app/schemas.py](python-service/app/schemas.py) |
| Logic | [src/Compute.php](php-service/src/Compute.php) | [src/compute.ts](node-service/src/compute.ts) | [domain/Compute.java](java-service/src/main/java/com/gorust/javaservice/domain/Compute.java) | [app/compute.py](python-service/app/compute.py) |
| Store | [src/ItemStore.php](php-service/src/ItemStore.php) (APCu) | [src/store.ts](node-service/src/store.ts) (Map) | [store/ItemStore.java](java-service/src/main/java/com/gorust/javaservice/store/ItemStore.java) (ConcurrentHashMap) | [app/store.py](python-service/app/store.py) (dict) |

## Model eksekusi: perbedaan paling penting dari keenam bahasa

| | Go | Rust | PHP-FPM | Node.js | Java 21 | Python (uvicorn) |
|---|---|---|---|---|---|---|
| Unit konkurensi | goroutine | tokio task | **proses** worker | callback/Promise di 1 thread | virtual thread | coroutine `async` + threadpool untuk `def` |
| Request bersamaan per proses | ribuan | ribuan | **1** | ribuan (I/O), 1 (CPU) | ribuan | ribuan (I/O), 1 (CPU, karena GIL) |
| Memakai multi-core | ya | ya | ya (banyak proses) | **tidak** (perlu cluster/worker_threads) | ya | **tidak** (GIL; perlu multi-proses atau build 3.13t) |
| CPU berat memblokir yang lain? | tidak (preemptive) | ya, di worker itu | tidak (proses terpisah) | **ya, semuanya** | sebagian (carrier thread) | `async def`: ya semuanya; `def`: GIL bergantian tiap ~5 ms |
| State di memori antar request | ya | ya | **tidak** (shared-nothing) | ya | ya | ya (per proses worker) |
| Kompilasi | AOT native | AOT native (LLVM) | bytecode + opcache + JIT opsional | JIT (V8) | bytecode + JIT (HotSpot C1/C2) | bytecode, **interpreter** (JIT eksperimental 3.13) |
| Manajemen memori | GC | ownership | refcount + cycle GC, dibuang per request | GC (V8) | GC (Serial/G1/ZGC) | refcount + cycle GC |

### PHP: shared-nothing
Setiap request menjalankan `index.php` dari awal: autoload, membuat objek, router, dan seterusnya, lalu **semuanya dibuang**. Konsekuensinya:
- **Sangat tahan bug memori/state**: kebocoran memori atau state rusak hilang di akhir request. Ini salah satu alasan PHP stabil di hosting murah.
- State antar request harus disimpan di luar proses: **APCu** (shared memory, dipakai di sini), Redis, atau database. Lihat `ItemStore.php`. Counter request pun memakai `apcu_inc`.
- Konkurensi = jumlah worker FPM (`PHP_WORKERS`). Jika VU melebihi jumlah worker, request **antre** di socket. Bandingkan p99 PHP saat VU = 8 dengan VU = 200.
- Butuh web server terpisah (nginx ↔ FastCGI). Go/Rust/Node/Java melayani HTTP sendiri.
- Alternatif modern: FrankenPHP worker mode, Swoole, dan RoadRunner membuat PHP *long-running* seperti Go, dengan risiko yang sama soal state bocor.

### Node.js: satu thread, event loop
- Kode JS berjalan di **satu thread**, sehingga `Map` tidak butuh lock (lihat `store.ts`). Tidak ada data race di level JS.
- I/O (jaringan, file) tidak memblokir, jadi Node sangat efisien untuk API yang banyak menunggu database.
- Kerja CPU berat (skenario `cpu`) **membekukan seluruh server**. Lihat `event loop delay` di kartu Node dan grafik `/metrics`.
- Satu proses ≈ satu core. Dengan `CPU_LIMIT=2`, Node tetap sekitar 100% kecuali memakai `cluster` (multi proses, yang berarti state tidak lagi bersama) atau `worker_threads`.
- TypeScript memberi type safety **hanya saat compile**. Data JSON dari luar tetap harus divalidasi manual (lihat `handlers.ts`) atau dengan library seperti zod.

### Java: virtual threads + JIT + Spring
- **Virtual threads** (Java 21) ≈ goroutine: tulis kode blocking biasa, runtime yang memultipleksnya. Diaktifkan dengan `spring.threads.virtual.enabled=true`.
- **JIT warm-up**: JVM mulai dari interpreter lalu mengompilasi kode panas (C1 → C2). Throughput beberapa detik pertama lebih rendah. Inilah alasan Java populer untuk server *long-running* dan kurang cocok untuk serverless/CLI.
- **Spring = konvensi + dependency injection**: `@RestController`, `@Repository`, dan constructor injection. Kode lebih sedikit, tapi lebih banyak "magic" dan startup lebih lambat (lihat log `Started ... in X seconds`).
- Error handling memakai exception + `@RestControllerAdvice` terpusat. Polanya mirip `IntoResponse` di Rust dan try/catch di `App.php`.
- JVM memerlukan memori dasar paling besar (metaspace, JIT code cache, heap). Dengan RAM < 1792 MB, JVM otomatis memilih **SerialGC** (lihat kartu Java).
- `ConcurrentHashMap` + `LongAdder` adalah struktur concurrent kelas industri tanpa lock global.

### Python: GIL, `async def` vs `def`, dan Pydantic
- **GIL (Global Interpreter Lock)**: hanya satu thread yang mengeksekusi bytecode Python pada satu waktu. Thread tetap berguna untuk I/O, tapi tidak untuk CPU. Python 3.13 punya build *free-threaded* (3.13t) tanpa GIL; kartu Python di dashboard menampilkan status GIL.
- FastAPI menjalankan **`async def`** di event loop (seperti Node), dan **`def` biasa di threadpool**. Lihat `api.py`: endpoint `cpu` sengaja memakai `def` supaya fib tidak membekukan event loop, tetapi tetap dibatasi GIL ke ~1 core.
- **Pydantic** = validasi deklaratif di model (`Field(ge=0)`, `min_length=1`, `field_validator`). Hasilnya mirip serde, tetapi dicek saat runtime. Default-nya *lax* ("12" diterima sebagai angka); di sini dibuat `strict=True`.
- **Type hints tidak dicek saat runtime**: sama seperti TypeScript, tipe hanya diperiksa oleh tool (mypy/pyright). Pydantic-lah yang memvalidasi data dari luar.
- CPython adalah interpreter tanpa JIT (default), sehingga skenario `cpu` menunjukkan selisih terbesar. Untuk beban I/O (database/HTTP), selisih ini jauh mengecil. Inilah alasan Python tetap sangat populer untuk API.
- Kejutan kecil: sejak Python 3.12, `sum()` untuk float memakai penjumlahan terkompensasi sehingga hasilnya bisa berbeda tipis dari bahasa lain. Lihat komentar di `compute.py`.

### Validasi data JSON: dari paling ketat ke paling longgar
1. **Rust** (serde): tipe salah atau field hilang langsung ditolak, semuanya diturunkan dari struct.
2. **Python** (Pydantic, strict) dan **Java** (Jackson): berbasis model, tetapi default-nya longgar ("12" diterima sebagai angka). Di sini keduanya diperketat (`strict=True`, `allow-coercion-of-scalars=false`).
3. **Go** (encoding/json): tipe salah ditolak, tapi field hilang menjadi zero value. Butuh pointer (`*float64`) untuk mendeteksinya.
4. **Node/PHP**: JSON menjadi objek/array dinamis, jadi **semua** validasi tipe ditulis manual (`typeof`, `is_int`).

## Kelebihan & kekurangan: PHP, Node.js, Java

### PHP
- **Kelebihan**: deploy & hosting termudah, shared-nothing membuatnya tahan bug state, ekosistem web matang (Laravel, Symfony, WordPress), dan PHP 8 modern sudah cukup bertipe (`readonly`, `enum`, typed properties).
- **Kekurangan**: overhead bootstrap per request, konkurensi dibatasi jumlah proses (memori ×N), butuh nginx + FPM, dan kurang cocok untuk WebSocket/long-running tanpa runtime khusus.
- **Cocok untuk**: web app/CMS/e-commerce, tim web, time-to-market cepat.

### Node.js
- **Kelebihan**: satu bahasa untuk frontend & backend, I/O non-blocking sangat efisien, ekosistem npm terbesar, startup cepat, dan TypeScript membuat kode besar tetap terkelola.
- **Kekurangan**: satu thread (CPU berat = bencana), type safety hilang saat runtime, ekosistem dependency rawan (supply chain), dan GC pause.
- **Cocok untuk**: API gateway/BFF, realtime (WebSocket), I/O-bound service, tim full-stack JS.

### Java
- **Kelebihan**: performa puncak setelah warm-up sangat tinggi, tooling & observability terbaik (JFR, profiler), ekosistem enterprise sangat matang, virtual threads membuat konkurensi sederhana, dan GC kelas dunia (ZGC dengan pause < 1 ms).
- **Kekurangan**: memori dasar besar, startup & warm-up lambat (bisa diatasi dengan GraalVM native-image atau CRaC), verbose, dan framework penuh "magic".
- **Cocok untuk**: sistem enterprise besar, layanan long-running dengan throughput tinggi, tim besar dengan kebutuhan tooling kuat.

### Python
- **Kelebihan**: paling mudah dibaca dan dipelajari, produktivitas tinggi, ekosistem data/AI/ML terbesar (NumPy, pandas, PyTorch), FastAPI + Pydantic memberi validasi dan dokumentasi OpenAPI otomatis, dan sangat cocok untuk prototyping.
- **Kekurangan**: interpreter lambat untuk kerja CPU, GIL membatasi multi-core dalam satu proses, memori per objek besar, dan type safety hanya lewat tool eksternal.
- **Cocok untuk**: API di sekitar data/ML, scripting & otomasi, prototipe cepat, dan service I/O-bound. Kerja berat biasanya dilempar ke library C/Rust (NumPy, Pydantic-core sendiri ditulis dengan Rust).

### Ringkasan memilih bahasa (aturan praktis)
| Kebutuhan utama | Pilihan kuat |
|---|---|
| Efisiensi resource maksimal, latency stabil | Rust |
| Microservice sederhana, cepat dikirim, hemat resource | Go |
| Web app/CMS dengan time-to-market cepat | PHP |
| Realtime/I/O-bound, satu bahasa dengan frontend | Node.js |
| Sistem enterprise besar, ekosistem & tooling matang | Java |
| API di sekitar data/AI/ML, prototipe cepat | Python |

## Urutan belajar Rust yang disarankan

1. Baca `compute.rs` dan `store.rs` (logika murni, tanpa async), lalu jalankan `cargo test`.
2. Pelajari *The Rust Book* bab 4 (Ownership), 6 (Enum & match), 9 (Error handling), 16 (Concurrency): <https://doc.rust-lang.org/book/>
3. Baca `error.rs` + `handlers.rs`: trait `From`, `IntoResponse`, operator `?`.
4. Latihan: tambahkan endpoint `DELETE /api/items/{id}` di **semua** service, lalu tambahkan kasusnya di `monitor/internal/verify/verify.go` dan pastikan verifikasi tetap 100%. Latihan ini menunjukkan "biaya" fitur yang sama di lima bahasa.
5. Latihan lanjut: ganti fibonacci Rust ke `tokio::task::spawn_blocking` dan ukur dampaknya pada grafik `/metrics` dan throughput.
