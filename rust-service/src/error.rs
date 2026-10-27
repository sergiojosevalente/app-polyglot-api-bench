//! Error aplikasi terpusat. Di Go setiap handler memanggil writeError(...);
//! di Rust handler cukup `return Err(AppError::...)` atau pakai operator `?`,
//! lalu trait `IntoResponse` mengubahnya menjadi HTTP response.

use axum::{
    extract::rejection::{JsonRejection, PathRejection, QueryRejection},
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;

#[derive(Debug)]
pub enum AppError {
    BadRequest(String),
    NotFound(String),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, msg) = match self {
            AppError::BadRequest(m) => (StatusCode::BAD_REQUEST, m),
            AppError::NotFound(m) => (StatusCode::NOT_FOUND, m),
        };
        (status, Json(json!({ "error": msg }))).into_response()
    }
}

// Implementasi `From` inilah yang membuat `?` bisa mengonversi error extractor
// axum menjadi AppError secara otomatis (default axum: 400/422 dengan body teks).
impl From<JsonRejection> for AppError {
    fn from(r: JsonRejection) -> Self {
        AppError::BadRequest(format!("invalid JSON: {}", r.body_text()))
    }
}

impl From<QueryRejection> for AppError {
    fn from(_: QueryRejection) -> Self {
        AppError::BadRequest("n must be a non-negative integer".into())
    }
}

impl From<PathRejection> for AppError {
    fn from(_: PathRejection) -> Self {
        AppError::BadRequest("id must be a non-negative integer".into())
    }
}
