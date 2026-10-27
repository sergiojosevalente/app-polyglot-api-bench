// Router minimal dengan parameter {name}. Di produksi umumnya memakai Express/Fastify;
// di sini memakai node:http murni agar 0 dependency (setara Go stdlib).
import type { IncomingMessage } from 'node:http';
import { HttpError } from './errors.ts';

export interface Context {
  req: IncomingMessage;
  url: URL;
  params: Record<string, string>;
}

export interface Reply {
  status: number;
  body: unknown;
}

export type Handler = (ctx: Context) => Reply | Promise<Reply>;

interface Route {
  method: string;
  regex: RegExp;
  keys: string[];
  handler: Handler;
}

const MAX_BODY = 1 << 20; // 1 MiB, sama dengan service lain

export class Router {
  readonly #routes: Route[] = [];

  add(method: string, pattern: string, handler: Handler): this {
    const keys: string[] = [];
    const source = pattern.replace(/\{(\w+)\}/g, (_, key: string) => {
      keys.push(key);
      return '([^/]+)';
    });
    this.#routes.push({ method, regex: new RegExp(`^${source}$`), keys, handler });
    return this;
  }

  async dispatch(req: IncomingMessage, url: URL): Promise<Reply> {
    let pathMatched = false;
    for (const route of this.#routes) {
      const m = route.regex.exec(url.pathname);
      if (!m) continue;
      pathMatched = true;
      if (route.method !== req.method) continue;
      const params = Object.fromEntries(route.keys.map((k, i) => [k, m[i + 1] ?? '']));
      return route.handler({ req, url, params });
    }
    return pathMatched
      ? { status: 405, body: { error: 'method not allowed' } }
      : { status: 404, body: { error: 'not found' } };
  }
}

// Body request datang sebagai stream chunk; dikumpulkan secara async (non-blocking).
export async function readJSON(req: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of req) {
    size += (chunk as Buffer).length;
    if (size > MAX_BODY) throw HttpError.badRequest('invalid JSON: body too large');
    chunks.push(chunk as Buffer);
  }
  try {
    return JSON.parse(Buffer.concat(chunks).toString('utf8'));
  } catch (err) {
    throw HttpError.badRequest(`invalid JSON: ${(err as Error).message}`);
  }
}

export function isObject(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v);
}
