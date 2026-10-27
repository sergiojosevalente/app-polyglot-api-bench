package com.gorust.javaservice.metrics;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

/** Statistik proses dari /proc/self/stat (Linux/Docker), sama seperti metrics.go. */
public record ProcStats(long rssBytes, double cpuSeconds, int threads) {

    private static final double CLOCK_TICKS = 100.0;
    private static final long PAGE_SIZE = 4096;

    public static ProcStats read() {
        try {
            String s = Files.readString(Path.of("/proc/self/stat"));
            // Nama proses bisa berisi spasi, jadi mulai setelah ')' terakhir; field k = f[k-3].
            String[] f = s.substring(s.lastIndexOf(')') + 2).trim().split("\\s+");
            if (f.length < 22) {
                return new ProcStats(0, 0, 0);
            }
            double cpu = (Double.parseDouble(f[11]) + Double.parseDouble(f[12])) / CLOCK_TICKS;
            return new ProcStats(Long.parseLong(f[21]) * PAGE_SIZE, cpu, Integer.parseInt(f[17]));
        } catch (IOException | RuntimeException e) {
            return new ProcStats(0, 0, 0); // bukan Linux
        }
    }
}
