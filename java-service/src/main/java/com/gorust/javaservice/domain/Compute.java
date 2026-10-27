package com.gorust.javaservice.domain;

import java.util.List;
import java.util.Optional;

/**
 * Business logic murni tanpa pengetahuan HTTP.
 * Padanannya: compute.go, compute.rs, Compute.php, compute.ts
 */
public final class Compute {

    public static final int MAX_FIB = 35;

    private Compute() {
    }

    /** Rekursif naif: benchmark CPU. JIT C2 HotSpot mengoptimasi setelah "warm-up". */
    public static long fib(int n) {
        return n < 2 ? n : fib(n - 1) + fib(n - 2);
    }

    /** Optional = padanan Option di Rust (tetapi null tetap ada di Java). */
    public static Optional<Stats> summarize(List<Double> nums) {
        if (nums == null || nums.isEmpty()) {
            return Optional.empty();
        }
        double sum = 0;
        double min = Double.POSITIVE_INFINITY;
        double max = Double.NEGATIVE_INFINITY;
        for (double n : nums) {
            sum += n;
            min = Math.min(min, n);
            max = Math.max(max, n);
        }
        return Optional.of(new Stats(nums.size(), sum, min, max, sum / nums.size()));
    }
}
