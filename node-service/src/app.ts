// Merakit router + handler menjadi satu request listener untuk node:http.
import type { IncomingMessage, RequestListener, ServerResponse } from 'node:http';
import { Handlers } from './handlers.ts';
import { HttpError } from './http/errors.ts';
import { Router } from './http/router.ts';
import type { Metrics } from './metrics.ts';
import type { ItemStore } from './store.ts';

export function createApp(store: ItemStore, metrics: Metrics): RequestListener {
  const h = new Handlers(store, metrics);
  const router = new Router()
    .add('GET', '/health', h.health)
    .add('GET', '/api/hello', h.hello)
    .add('POST', '/api/json', h.jsonStats)
    .add('GET', '/api/cpu', h.cpu)
    .add('GET', '/api/items/{id}', h.getItem)
    .add('PUT', '/api/items/{id}', h.putItem)
    .add('GET', '/metrics', h.metrics);

  return async (req: IncomingMessage, res: ServerResponse) => {
    metrics.countRequest();
    try {
      const url = new URL(req.url ?? '/', 'http://localhost');
      const reply = await router.dispatch(req, url);
      send(res, reply.status, reply.body);
    } catch (err) {
      // Error handling terpusat (try/catch + throw), mirip PHP & Java.
      if (err instanceof HttpError) {
        send(res, err.status, { error: err.message });
      } else {
        console.error(err);
        send(res, 500, { error: 'internal server error' });
      }
    }
  };
}

function send(res: ServerResponse, status: number, body: unknown): void {
  const payload = JSON.stringify(body) + '\n';
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(payload),
  });
  res.end(payload);
}
