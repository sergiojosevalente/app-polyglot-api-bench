//! Lapisan HTTP: router, extractor, validasi, response.
//! Padanannya: go-service/internal/handler/handler.go

use std::sync::Arc;

use axum::{
    extract::{
        rejection::{JsonRejection, PathRejection, QueryRejection},
        DefaultBodyLimit, Path, Query, Request, State,
    },
    middleware::{self, Next},
    response::Response,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};

use crate::{
    compute::{self, Stats, MAX_FIB},
    error::AppError,
    metrics::{self, Metrics},
    store::{Item, Store},
};

const LANG: &str = "rust";
const DEFAULT_FIB_N: u32 = 25;
const MAX_BODY_SIZE: usize = 1 << 20; // 1 MiB, sama dengan maxBodySize di Go

/// State bersama. `Clone` murah karena isinya hanya `Arc` (naikkan ref-count).
/// Padanan struct Handler{store, metrics} di Go.
#[derive(Clone)]
pub struct AppState {
    pub store: Arc<Store>,
    pub metrics: Arc<Metrics>,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/health", get(health))
        .route("/api/hello", get(hello))
        .route("/api/json", post(json_stats))
        .route("/api/cpu", get(cpu))
        .route("/api/items/{id}", get(get_item).put(put_item))
        .route("/metrics", get(metrics_handler))
        .layer(middleware::from_fn_with_state(state.clone(), count_requests))
        .layer(DefaultBodyLimit::max(MAX_BODY_SIZE))
        .with_state(state)
}

/// Middleware: padanan Metrics.Middleware di Go.
async fn count_requests(State(state): State<AppState>, req: Request, next: Next) -> Response {
    state.metrics.inc();
    next.run(req).await
}

// ---- request / response types: derive macro menghasilkan kode (de)serialisasi saat compile ----

#[derive(Serialize)]
struct HealthResponse {
    status: &'static str,
    lang: &'static str,
}

#[derive(Deserialize)]
struct HelloQuery {
    name: Option<String>,
}

#[derive(Serialize)]
struct HelloResponse {
    message: String,
    lang: &'static str,
}

#[derive(Deserialize)]
struct CpuQuery {
    n: Option<u32>,
}

#[derive(Serialize)]
struct CpuResponse {
    n: u32,
    result: u64,
    lang: &'static str,
}

#[derive(Deserialize)]
struct StatsRequest {
    numbers: Vec<f64>,
}

/// `#[serde(flatten)]` = padanan struct embedding di Go.
#[derive(Serialize)]
struct StatsResponse {
    #[serde(flatten)]
    stats: Stats,
    lang: &'static str,
}

/// Field tanpa `Option` WAJIB ada di JSON; kalau hilang serde menolak request.
/// (Go perlu *float64 untuk perilaku yang sama.)
#[derive(Deserialize)]
struct UpsertRequest {
    name: String,
    price: f64,
}

#[derive(Serialize)]
struct MetricsResponse {
    lang: &'static str,
    version: &'static str,
    uptime_sec: f64,
    requests_total: u64,
    rss_bytes: u64,
    cpu_seconds: f64,
    threads: u64,
    items: usize,
    runtime: RustRuntime,
}

#[derive(Serialize)]
struct RustRuntime {
    tokio_workers: usize,
    alive_tasks: usize,
}

// ---- handlers: argumen = "extractor", diisi otomatis oleh axum dari request ----

async fn health() -> Json<HealthResponse> {
    Json(HealthResponse { status: "ok", lang: LANG })
}

async fn hello(Query(q): Query<HelloQuery>) -> Json<HelloResponse> {
    let name = q.name.filter(|n| !n.is_empty()).unwrap_or_else(|| "World".to_string());
    Json(HelloResponse { message: format!("Hello, {name}!"), lang: LANG })
}

async fn cpu(query: Result<Query<CpuQuery>, QueryRejection>) -> Result<Json<CpuResponse>, AppError> {
    let Query(q) = query?;
    let n = q.n.unwrap_or(DEFAULT_FIB_N);
    if n > MAX_FIB {
        return Err(AppError::BadRequest(format!("n must be <= {MAX_FIB}")));
    }
    // PENTING: tokio bersifat cooperative — task hanya melepas thread di `.await`.
    // Kerja CPU di sini memblokir worker thread sampai selesai. Untuk fib(25) (<1 ms)
    // masih wajar; untuk kerja berat gunakan tokio::task::spawn_blocking atau rayon.
    // Go tidak punya masalah ini karena goroutine di-preempt oleh runtime.
    Ok(Json(CpuResponse { n, result: compute::fib(n), lang: LANG }))
}

async fn json_stats(
    payload: Result<Json<StatsRequest>, JsonRejection>,
) -> Result<Json<StatsResponse>, AppError> {
    let Json(req) = payload?; // `?` -> From<JsonRejection> for AppError
    let stats = compute::summarize(&req.numbers)
        .ok_or_else(|| AppError::BadRequest("numbers must not be empty".into()))?;
    Ok(Json(StatsResponse { stats, lang: LANG }))
}

async fn get_item(
    State(state): State<AppState>,
    path: Result<Path<u64>, PathRejection>,
) -> Result<Json<Item>, AppError> {
    let Path(id) = path?;
    state
        .store
        .get(id)
        .map(Json)
        .ok_or_else(|| AppError::NotFound(format!("item {id} not found")))
}

// Extractor yang membaca body (Json) harus menjadi argumen terakhir.
async fn put_item(
    State(state): State<AppState>,
    path: Result<Path<u64>, PathRejection>,
    payload: Result<Json<UpsertRequest>, JsonRejection>,
) -> Result<Json<Item>, AppError> {
    let Path(id) = path?;
    let Json(req) = payload?;
    if req.name.trim().is_empty() {
        return Err(AppError::BadRequest("name must not be empty".into()));
    }
    if req.price < 0.0 {
        return Err(AppError::BadRequest("price must be >= 0".into()));
    }
    // `req.name` di-move (bukan disalin) ke Item: ownership berpindah, tanpa alokasi baru.
    let item = state.store.upsert(Item { id, name: req.name, price: req.price });
    Ok(Json(item))
}

async fn metrics_handler(State(state): State<AppState>) -> Json<MetricsResponse> {
    let ps = metrics::read_proc();
    let rt = tokio::runtime::Handle::current().metrics();
    Json(MetricsResponse {
        lang: LANG,
        // Diisi saat build di Dockerfile (RUSTC_VERSION=$(rustc --version)).
        version: option_env!("RUSTC_VERSION").unwrap_or("rustc (unknown)"),
        uptime_sec: state.metrics.uptime_secs(),
        requests_total: state.metrics.requests(),
        rss_bytes: ps.rss_bytes,
        cpu_seconds: ps.cpu_seconds,
        threads: ps.threads,
        items: state.store.count(),
        runtime: RustRuntime { tokio_workers: rt.num_workers(), alive_tasks: rt.num_alive_tasks() },
    })
}
