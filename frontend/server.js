// Custom dev server: proxies WebSocket upgrades for /api/assistant/live
// to the backend, so local dev matches the production Caddy setup.
// Only used in development (pnpm dev). Production uses Caddy.
const { createServer } = require('http');
const { parse } = require('url');
const next = require('next');
const httpProxy = require('http-proxy');

const port = parseInt(process.env.PORT || '3000', 10);
const app = next({ dev: true, hostname: 'localhost', port });
const handle = app.getRequestHandler();

const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080';
const proxy = httpProxy.createProxyServer({ target: backendUrl, ws: true });
proxy.on('error', (err, _req, res) => {
  console.error('[ws-proxy] error:', err.message);
  if (res && typeof res.end === 'function') res.end();
});

app.prepare().then(() => {
  const server = createServer((req, res) => {
    handle(req, res, parse(req.url, true));
  });

  server.on('upgrade', (req, socket, head) => {
    if (req.url.startsWith('/api/assistant/live')) {
      proxy.ws(req, socket, head);
    }
  });

  server.listen(port, 'localhost', () => {
    console.log(`> Ready on http://localhost:${port}`);
  });
});
