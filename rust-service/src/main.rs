//! Entry point: hanya wiring (config -> state -> router -> server).
//! Padanannya: go-service/cmd/server/main.go

// Di Rust, modul harus dideklarasikan eksplisit (Go: otomatis per folder/package).
mod compute;
mod error;
mod handlers;
mod metrics;
mod store;

use std::{net::SocketAddr, sync::Arc};

use tokio::signal;

use crate::{handlers::AppState, metrics::Metrics, store::Store};

// Macro ini membuat runtime tokio multi-thread lalu menjalankan `main` async di atasnya.
// Go tidak perlu ini: runtime goroutine selalu ada di setiap program.
#[tokio::main]
async fn main() {
    let port: u16 = std::env::var("PORT").ok().and_then(|p| p.parse().ok()).unwrap_or(8080);

    let store = Store::new();
    store.seed(1000);
    let state = AppState { store: Arc::new(store), metrics: Arc::new(Metrics::new()) };

    let addr = SocketAddr::from(([0, 0, 0, 0], port));
    let listener = tokio::net::TcpListener::bind(addr).await.expect("bind port");
    println!("rust-service listening on {addr}");

    axum::serve(listener, handlers::router(state))
        .with_graceful_shutdown(shutdown_signal())
        .await
        .expect("server error");
    println!("rust-service stopped");
}

/// Selesai saat Ctrl+C atau SIGTERM (docker stop). Padanan signal.NotifyContext di Go.
async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c().await.expect("install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("install SIGTERM handler")
            .recv()
            .await;
    };
    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
    println!("shutting down...");
}
