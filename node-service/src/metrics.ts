// Counter request + statistik proses. Padanannya: metrics.go, metrics.rs
import { readFileSync } from 'node:fs';
import { monitorEventLoopDelay } from 'node:perf_hooks';

export class Metrics {
  readonly #started = Date.now();
  #requests = 0; // number biasa cukup: single thread, tidak perlu atomic
  // Event loop delay = seberapa lama callback harus antre. Inilah "tail latency"
  // khas Node: satu kerja CPU berat menahan SEMUA request lain.
  readonly loopDelay = monitorEventLoopDelay({ resolution: 10 });

  constructor() {
    this.loopDelay.enable();
  }

  countRequest(): void {
    this.#requests++;
  }

  get requests(): number {
    return this.#requests;
  }

  get uptimeSec(): number {
    return (Date.now() - this.#started) / 1000;
  }
}

export interface ProcStats {
  rssBytes: number;
  cpuSeconds: number;
  threads: number;
}

// Node menyediakan API lintas-OS (getrusage di baliknya), jadi tidak perlu parsing
// /proc kecuali untuk jumlah thread (V8 + libuv threadpool).
export function readProc(): ProcStats {
  const cpu = process.cpuUsage();
  let threads = 0;
  try {
    const m = /^Threads:\s+(\d+)/m.exec(readFileSync('/proc/self/status', 'utf8'));
    threads = m ? Number(m[1]) : 0;
  } catch {
    // bukan Linux
  }
  return {
    rssBytes: process.memoryUsage.rss(),
    cpuSeconds: (cpu.user + cpu.system) / 1e6,
    threads,
  };
}
