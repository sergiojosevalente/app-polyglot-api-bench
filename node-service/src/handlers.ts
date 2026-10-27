// Lapisan HTTP: validasi, panggil business logic, bentuk response.
// Padanannya: handler.go, handlers.rs, Handler.php
import { fib, MAX_FIB, summarize } from './compute.ts';
import { HttpError } from './http/errors.ts';
import { isObject, readJSON, type Context, type Reply } from './http/router.ts';
import { readProc, type Metrics } from './metrics.ts';
import type { ItemStore } from './store.ts';

const LANG = 'node';
const DEFAULT_FIB_N = 25;

const ok = (body: unknown): Reply => ({ status: 200, body });

// Catatan type stripping: "parameter properties" (constructor(private x)) tidak
// didukung karena butuh transformasi kode; field dideklarasikan eksplisit.
export class Handlers {
  readonly #store: ItemStore;
  readonly #metrics: Metrics;

  constructor(store: ItemStore, metrics: Metrics) {
    this.#store = store;
    this.#metrics = metrics;
  }

  // Arrow function sebagai field: `this` terikat otomatis saat dioper ke router.
  health = (): Reply => ok({ status: 'ok', lang: LANG });

  hello = ({ url }: Context): Reply => {
    const name = url.searchParams.get('name') || 'World';
    return ok({ message: `Hello, ${name}!`, lang: LANG });
  };

  cpu = ({ url }: Context): Reply => {
    let n = DEFAULT_FIB_N;
    const raw = url.searchParams.get('n');
    if (raw !== null && raw !== '') {
      if (!/^\d{1,10}$/.test(raw)) throw HttpError.badRequest('n must be a non-negative integer');
      n = Number(raw);
    }
    if (n > MAX_FIB) throw HttpError.badRequest(`n must be <= ${MAX_FIB}`);
    // PENTING: kode sinkron ini memblokir event loop. Selama fib berjalan, Node
    // tidak bisa melayani request lain SAMA SEKALI (lihat event_loop_delay di /metrics).
    // Solusi untuk kerja berat: worker_threads atau memindahkannya ke service lain.
    return ok({ n, result: fib(n), lang: LANG });
  };

  jsonStats = async ({ req }: Context): Promise<Reply> => {
    const body = await readJSON(req);
    const numbers = isObject(body) ? (body.numbers ?? []) : undefined;
    // Tipe TypeScript hilang saat runtime: data dari luar wajib divalidasi manual
    // (atau pakai library schema seperti zod).
    if (!Array.isArray(numbers) || !numbers.every((x) => typeof x === 'number')) {
      throw HttpError.badRequest('invalid JSON: numbers must be an array of numbers');
    }
    const stats = summarize(numbers);
    if (!stats) throw HttpError.badRequest('numbers must not be empty');
    return ok({ ...stats, lang: LANG });
  };

  getItem = ({ params }: Context): Reply => {
    const id = parseId(params.id);
    const item = this.#store.get(id);
    if (!item) throw HttpError.notFound(`item ${id} not found`);
    return ok(item);
  };

  putItem = async ({ req, params }: Context): Promise<Reply> => {
    const id = parseId(params.id);
    const body = await readJSON(req);
    const { name, price } = isObject(body) ? body : {};
    if (typeof name !== 'string' || name.trim() === '') throw HttpError.badRequest('name must not be empty');
    if (price === undefined || price === null) throw HttpError.badRequest('price is required');
    if (typeof price !== 'number') throw HttpError.badRequest('invalid JSON: price must be a number');
    if (price < 0) throw HttpError.badRequest('price must be >= 0');
    return ok(this.#store.upsert({ id, name, price }));
  };

  metrics = (): Reply => {
    const ps = readProc();
    const loop = this.#metrics.loopDelay;
    const runtime = {
      heap_used_bytes: process.memoryUsage().heapUsed,
      event_loop_delay_mean_ms: loop.mean / 1e6,
      event_loop_delay_p99_ms: loop.percentile(99) / 1e6,
      event_loop_delay_max_ms: loop.max / 1e6,
      uv_threadpool_size: Number(process.env.UV_THREADPOOL_SIZE ?? 4),
    };
    loop.reset(); // jendela per scrape (monitor scrape tiap 1 detik)
    return ok({
      lang: LANG,
      version: `Node ${process.version}`,
      uptime_sec: this.#metrics.uptimeSec,
      requests_total: this.#metrics.requests,
      rss_bytes: ps.rssBytes,
      cpu_seconds: ps.cpuSeconds,
      threads: ps.threads,
      items: this.#store.count(),
      runtime,
    });
  };
}

function parseId(raw: string | undefined): number {
  if (raw === undefined || !/^\d{1,15}$/.test(raw)) {
    throw HttpError.badRequest('id must be a non-negative integer');
  }
  return Number(raw);
}
