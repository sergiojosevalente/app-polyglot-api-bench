// Entry point: wiring + graceful shutdown. Padanannya: main.go, main.rs
import { createServer } from 'node:http';
import { createApp } from './app.ts';
import { Metrics } from './metrics.ts';
import { ItemStore } from './store.ts';

const port = Number(process.env.PORT ?? 8080);

const store = new ItemStore();
store.seed(1000);
const metrics = new Metrics();

const server = createServer(createApp(store, metrics));
// Default Node 5 detik; disamakan dengan service lain agar koneksi keep-alive tidak sering putus.
server.keepAliveTimeout = 65_000;

server.listen(port, () => {
  console.log(`node-service listening on :${port}`);
});

// Node sebagai PID 1 di Docker TIDAK menangani SIGTERM kecuali didaftarkan eksplisit.
function shutdown(): void {
  console.log('shutting down...');
  server.close(() => {
    console.log('node-service stopped');
    process.exit(0);
  });
  server.closeIdleConnections();
  setTimeout(() => process.exit(1), 5_000).unref();
}
process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);
