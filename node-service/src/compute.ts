// Business logic murni tanpa pengetahuan HTTP.
// Padanannya: compute.go, compute.rs, Compute.php

export const MAX_FIB = 35;

// Sengaja rekursif naif: benchmark CPU murni. V8 men-JIT fungsi "panas" ini
// (Ignition interpreter -> Maglev/TurboFan), jadi detik-detik awal lebih lambat.
export function fib(n: number): number {
  return n < 2 ? n : fib(n - 1) + fib(n - 2);
}

export interface Stats {
  count: number;
  sum: number;
  min: number;
  max: number;
  mean: number;
}

// `Stats | undefined` = padanan Option<Stats>; dengan strict TypeScript,
// caller dipaksa menangani undefined (tapi hanya saat type-check, bukan runtime).
export function summarize(nums: readonly number[]): Stats | undefined {
  if (nums.length === 0) return undefined;
  let sum = 0;
  let min = Infinity;
  let max = -Infinity;
  for (const n of nums) {
    sum += n;
    min = Math.min(min, n);
    max = Math.max(max, n);
  }
  return { count: nums.length, sum, min, max, mean: sum / nums.length };
}
