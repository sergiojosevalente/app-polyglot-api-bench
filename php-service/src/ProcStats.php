<?php

declare(strict_types=1);

namespace App;

/**
 * Statistik resource SELURUH container (nginx + FPM master + semua worker).
 * Go/Rust cukup membaca /proc/self karena satu proses; PHP-FPM adalah banyak
 * proses, jadi dijumlahkan dari /proc/<pid> (container punya PID namespace sendiri).
 *
 * Memori memakai PSS (Proportional Set Size), bukan RSS: memori bersama
 * (opcache, kode PHP) dibagi rata antar proses, sehingga tidak dihitung berkali-kali.
 */
final readonly class ProcStats
{
    private const CLOCK_TICKS = 100.0;
    private const PAGE_SIZE = 4096;

    public function __construct(
        public int $memoryBytes,
        public string $memorySource,
        public float $cpuSeconds,
        public int $threads,
        public int $processes,
        public int $fpmWorkers,
        public float $uptimeSec,
    ) {
    }

    public static function read(): self
    {
        $mem = 0;
        $cpu = 0.0;
        $threads = 0;
        $procs = 0;
        $workers = 0;
        $source = 'pss';

        foreach (glob('/proc/[0-9]*', GLOB_ONLYDIR) ?: [] as $dir) {
            $f = self::statFields($dir);
            if ($f === null) {
                continue; // proses sudah selesai di antara glob() dan read
            }
            $procs++;
            // utime + stime + cutime + cstime: termasuk anak proses yang sudah selesai
            $cpu += ((float) $f[11] + (float) $f[12] + (float) $f[13] + (float) $f[14]) / self::CLOCK_TICKS;
            $threads += (int) $f[17];

            $rollup = @file_get_contents("{$dir}/smaps_rollup");
            if (is_string($rollup) && preg_match('/^Pss:\s+(\d+) kB/m', $rollup, $m) === 1) {
                $mem += (int) $m[1] * 1024;
            } else {
                $mem += (int) $f[21] * self::PAGE_SIZE;
                $source = 'rss';
            }

            $cmd = @file_get_contents("{$dir}/cmdline");
            if (is_string($cmd) && str_starts_with($cmd, 'php-fpm: pool')) {
                $workers++;
            }
        }

        return new self($mem, $source, $cpu, $threads, $procs, $workers, self::uptime());
    }

    /** @return list<string>|null field /proc/<pid>/stat mulai dari field 3 (field k = index k-3) */
    private static function statFields(string $dir): ?array
    {
        $s = @file_get_contents("{$dir}/stat");
        if (!is_string($s)) {
            return null;
        }
        $pos = strrpos($s, ')');
        if ($pos === false) {
            return null;
        }
        $f = preg_split('/\s+/', trim(substr($s, $pos + 2)));
        return is_array($f) && count($f) >= 22 ? $f : null;
    }

    /** Uptime container = uptime sistem - waktu start PID 1. */
    private static function uptime(): float
    {
        $sys = @file_get_contents('/proc/uptime');
        $f = self::statFields('/proc/1');
        if (!is_string($sys) || $f === null) {
            return 0.0;
        }
        return max(0.0, (float) $sys - (float) $f[19] / self::CLOCK_TICKS); // field 22 = starttime
    }
}
