const SHELL_VERSION = 'v1';
const SHELL_CACHE = `asesor-shell-${SHELL_VERSION}`;
const CONTENT_CACHE = 'asesor-content';

// Versioned shell assets. Cache-first: these are immutable for a deployment.
const SHELL_ASSETS = [
  '/assets/main.css',
  '/assets/htmx.min.js',
  '/assets/icon-192.png',
  '/assets/icon-512.png',
  '/offline.html'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(SHELL_CACHE).then((cache) => cache.addAll(SHELL_ASSETS))
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(
        keys
          .filter((key) => key !== SHELL_CACHE && key !== CONTENT_CACHE)
          .map((key) => caches.delete(key))
      )
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  const { request } = event;

  // Never cache mutations or non-idempotent requests.
  if (request.method !== 'GET') {
    return;
  }

  const url = new URL(request.url);

  // Shell assets: cache-first. The response body is never rewritten.
  if (SHELL_ASSETS.includes(url.pathname)) {
    event.respondWith(
      caches.match(request).then((cached) => {
        if (cached) {
          return cached;
        }
        return fetch(request).then((response) => {
          const copy = response.clone();
          caches.open(SHELL_CACHE).then((cache) => cache.put(request, copy));
          return response;
        });
      })
    );
    return;
  }

  // Documents (HTML pages): stale-while-revalidate with offline fallback.
  // The cached HTML preserves server-rendered timestamps and freshness classes.
  if (request.mode === 'navigate' || request.headers.get('accept')?.includes('text/html')) {
    event.respondWith(
      caches.open(CONTENT_CACHE).then((cache) =>
        fetch(request)
          .then((response) => {
            if (response.ok) {
              const copy = response.clone();
              cache.put(request, copy);
            }
            return response;
          })
          .catch(() =>
            caches.match(request).then((cached) => {
              if (cached) {
                return cached;
              }
              return caches.match('/offline.html');
            })
          )
      )
    );
    return;
  }
});
