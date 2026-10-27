//! Business logic murni tanpa pengetahuan HTTP.
//! Padanannya: go-service/internal/compute/compute.go

use serde::Serialize;

/// Batas n supaya satu request tidak memonopoli CPU berdetik-detik.
pub const MAX_FIB: u32 = 35;

/// Sengaja rekursif naif (eksponensial): benchmark CPU murni.
pub fn fib(n: u32) -> u64 {
    if n < 2 {
        n as u64
    } else {
        fib(n - 1) + fib(n - 2)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Serialize)]
pub struct Stats {
    pub count: usize,
    pub sum: f64,
    pub min: f64,
    pub max: f64,
    pub mean: f64,
}

/// Rust memakai `Option` untuk "mungkin tidak ada hasil" — tidak ada nil,
/// compiler memaksa caller menangani kasus `None`.
pub fn summarize(nums: &[f64]) -> Option<Stats> {
    if nums.is_empty() {
        return None;
    }
    let sum: f64 = nums.iter().sum();
    let min = nums.iter().copied().fold(f64::INFINITY, f64::min);
    let max = nums.iter().copied().fold(f64::NEG_INFINITY, f64::max);
    Some(Stats {
        count: nums.len(),
        sum,
        min,
        max,
        mean: sum / nums.len() as f64,
    })
}

// Unit test tinggal di file yang sama, hanya dikompilasi saat `cargo test`.
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fib_values() {
        for (n, want) in [(0, 0), (1, 1), (2, 1), (10, 55), (25, 75025)] {
            assert_eq!(fib(n), want, "fib({n})");
        }
    }

    #[test]
    fn summarize_values() {
        let s = summarize(&[1.0, 2.0, 3.0, 4.5, -2.0]).unwrap();
        assert_eq!(
            s,
            Stats { count: 5, sum: 8.5, min: -2.0, max: 4.5, mean: 1.7 }
        );
    }

    #[test]
    fn summarize_empty() {
        assert!(summarize(&[]).is_none());
    }
}
