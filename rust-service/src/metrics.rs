//! Counter request + statistik proses dari /proc (Linux/Docker).
//! Padanannya: go-service/internal/metrics/metrics.go

use std::{
    sync::atomic::{AtomicU64, Ordering},
    time::Instant,
};

pub struct Metrics {
    started: Instant,
    requests: AtomicU64,
}

impl Metrics {
    pub fn new() -> Self {
        Self { started: Instant::now(), requests: AtomicU64::new(0) }
    }

    /// `&self` (bukan `&mut self`) cukup karena atomic punya interior mutability.
    pub fn inc(&self) {
        self.requests.fetch_add(1, Ordering::Relaxed);
    }

    pub fn requests(&self) -> u64 {
        self.requests.load(Ordering::Relaxed)
    }

    pub fn uptime_secs(&self) -> f64 {
        self.started.elapsed().as_secs_f64()
    }
}

#[derive(Debug, Default)]
pub struct ProcStats {
    pub rss_bytes: u64,
    pub cpu_seconds: f64,
    pub threads: u64,
}

/// USER_HZ, hampir selalu 100 di Linux.
const CLOCK_TICKS: f64 = 100.0;
/// Ukuran page x86_64/arm64 umum (Go memakai os.Getpagesize()).
const PAGE_SIZE: u64 = 4096;

/// Membaca /proc/self/stat. Di luar Linux hasilnya nol (ProcStats::default()).
pub fn read_proc() -> ProcStats {
    let Ok(s) = std::fs::read_to_string("/proc/self/stat") else {
        return ProcStats::default();
    };
    // Nama proses bisa berisi spasi, jadi mulai setelah ')' terakhir.
    let Some(pos) = s.rfind(')') else {
        return ProcStats::default();
    };
    // f[0] = field 3 (state), jadi field k = f[k-3]
    let f: Vec<&str> = s[pos + 2..].split_whitespace().collect();
    if f.len() < 22 {
        return ProcStats::default();
    }
    let num = |i: usize| f[i].parse::<f64>().unwrap_or(0.0);
    ProcStats {
        cpu_seconds: (num(11) + num(12)) / CLOCK_TICKS, // utime + stime
        threads: f[17].parse().unwrap_or(0),
        rss_bytes: f[21].parse::<u64>().unwrap_or(0) * PAGE_SIZE,
    }
}
