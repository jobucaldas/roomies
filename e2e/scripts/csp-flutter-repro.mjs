#!/usr/bin/env node
/**
 * Reproduce Flutter web boot under the Caddy CSP from tip 83b200f.
 * Usage: node e2e/scripts/csp-flutter-repro.mjs [--cdn|--local]
 */
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { chromium } from '@playwright/test';

const mode = process.argv.includes('--cdn') ? 'cdn' : 'local';
const root = path.resolve('src/flutter/build/web');
// Fixed CSP (connect-src includes fonts.gstatic.com for Flutter Roboto fetch).
// Pass --legacy-csp to reproduce tip 83b200f breakage.
const legacyCsp =
  "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: blob:; font-src 'self' data: https://fonts.gstatic.com; connect-src 'self'; worker-src 'self' blob:; manifest-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'";
const fixedCsp =
  "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: blob:; font-src 'self' data: https://fonts.gstatic.com; connect-src 'self' https://fonts.gstatic.com; worker-src 'self' blob:; manifest-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'";
const csp = process.argv.includes('--legacy-csp') ? legacyCsp : fixedCsp;

const mime = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.wasm': 'application/wasm',
  '.json': 'application/json',
  '.png': 'image/png',
  '.css': 'text/css',
  '.ttf': 'font/ttf',
  '.otf': 'font/otf',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
};

function rewriteBootstrap(body) {
  if (mode === 'cdn') {
    return body.replace(/,"useLocalCanvasKit":true/, '').replace(
      /"useLocalCanvasKit":true,?/,
      '',
    );
  }
  if (!body.includes('"useLocalCanvasKit":true')) {
    return body.replace(
      /"_flutter\.buildConfig"\s*=\s*\{/,
      '',
    ).replace(
      /(_flutter\.buildConfig\s*=\s*\{)/,
      '$1"useLocalCanvasKit":true,',
    );
  }
  return body;
}

const server = http.createServer((req, res) => {
  const urlPath = decodeURIComponent((req.url || '/').split('?')[0]);
  let filePath = path.join(root, urlPath === '/' ? 'index.html' : urlPath);
  if (!filePath.startsWith(root) || !fs.existsSync(filePath) || fs.statSync(filePath).isDirectory()) {
    filePath = path.join(root, 'index.html');
  }
  let body = fs.readFileSync(filePath);
  const ext = path.extname(filePath);
  if (path.basename(filePath) === 'flutter_bootstrap.js') {
    body = Buffer.from(rewriteBootstrap(body.toString('utf8')), 'utf8');
  }
  res.writeHead(200, {
    'Content-Type': mime[ext] || 'application/octet-stream',
    'Content-Security-Policy': csp,
    'Referrer-Policy': 'no-referrer',
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
  });
  res.end(body);
});

await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
const { port } = server.address();
const base = `http://127.0.0.1:${port}`;

const browser = await chromium.launch();
const page = await browser.newPage();
const consoleErrors = [];
const pageErrors = [];
const failed = [];
const responses = [];
page.on('console', (m) => {
  if (m.type() === 'error') consoleErrors.push(m.text());
});
page.on('pageerror', (e) => pageErrors.push(String(e)));
page.on('requestfailed', (r) =>
  failed.push({ url: r.url(), error: r.failure()?.errorText }),
);
page.on('response', (r) => responses.push({ url: r.url(), status: r.status() }));

await page.goto(base, { waitUntil: 'domcontentloaded', timeout: 60_000 });
let loginVisible = false;
try {
  await page.getByRole('heading', { name: 'Login' }).waitFor({
    state: 'visible',
    timeout: 45_000,
  });
  loginVisible = true;
} catch {
  loginVisible = false;
}

const cspViolations = consoleErrors.filter((t) =>
  /Content Security Policy|CSP|Refused to/i.test(t),
);

const report = {
  mode,
  loginVisible,
  consoleErrorCount: consoleErrors.length,
  pageErrorCount: pageErrors.length,
  failedRequestCount: failed.length,
  consoleErrors,
  pageErrors,
  failed,
  cspViolations,
  externalUrls: [...new Set(responses.map((r) => r.url).filter((u) => !u.startsWith(base)))],
  sameOriginJsWasm: responses
    .filter((r) => r.url.startsWith(base) && /\.(js|wasm)(\?|$)/.test(r.url))
    .map((r) => r.url.replace(base, '')),
};

fs.mkdirSync('/opt/cursor/logs', { recursive: true });
fs.appendFileSync(
  '/opt/cursor/logs/debug.log',
  JSON.stringify({
    location: 'e2e/scripts/csp-flutter-repro.mjs',
    message: 'csp flutter boot repro',
    data: report,
    timestamp: Date.now(),
    hypothesisId: mode === 'cdn' ? 'A' : 'A-fix',
  }) + '\n',
);

console.log(JSON.stringify(report, null, 2));
await browser.close();
server.close();
process.exit(loginVisible ? 0 : 1);
