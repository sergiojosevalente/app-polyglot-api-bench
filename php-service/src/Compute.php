<?php

declare(strict_types=1);

namespace App;

/**
 * Business logic murni tanpa pengetahuan HTTP.
 * Padanannya: go-service/internal/compute/compute.go, rust-service/src/compute.rs
 */
final class Compute
{
    public const MAX_FIB = 35;

    /** Sengaja rekursif naif (eksponensial): benchmark CPU murni. JIT sangat berpengaruh di sini. */
    public static function fib(int $n): int
    {
        return $n < 2 ? $n : self::fib($n - 1) + self::fib($n - 2);
    }

    /**
     * Nullable return (?Stats) = padanan Option<Stats> di Rust,
     * tapi tidak dipaksa compiler: caller bisa lupa cek null.
     *
     * @param list<int|float> $nums
     */
    public static function summarize(array $nums): ?Stats
    {
        if ($nums === []) {
            return null;
        }
        $sum = 0.0;
        $min = INF;
        $max = -INF;
        foreach ($nums as $n) {
            $n = (float) $n;
            $sum += $n;
            $min = min($min, $n);
            $max = max($max, $n);
        }
        $count = count($nums);
        return new Stats($count, $sum, $min, $max, $sum / $count);
    }
}
