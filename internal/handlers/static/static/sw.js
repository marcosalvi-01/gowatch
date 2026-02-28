const CACHE_NAME = "gowatch-static-v1";
const OFFLINE_URL = "/static/offline.html";
const PRECACHE_URLS = [
	OFFLINE_URL,
	"/static/css/output.css",
	"/static/favicon.svg",
	"/static/icon-192.png",
	"/static/icon-512.png",
];

self.addEventListener("install", event => {
	event.waitUntil(
		caches
			.open(CACHE_NAME)
			.then(cache => cache.addAll(PRECACHE_URLS))
			.then(() => self.skipWaiting())
	);
});

self.addEventListener("activate", event => {
	event.waitUntil(
		caches.keys().then(cacheNames =>
			Promise.all(
				cacheNames
					.filter(cacheName => cacheName !== CACHE_NAME)
					.map(cacheName => caches.delete(cacheName))
			)
		)
			.then(() => self.clients.claim())
	);
});

self.addEventListener("fetch", event => {
	const { request } = event;

	if (request.method !== "GET") {
		return;
	}

	const requestURL = new URL(request.url);
	if (requestURL.origin !== self.location.origin) {
		return;
	}

	if (request.mode === "navigate") {
		event.respondWith(handleNavigationRequest(request));
		return;
	}

	if (requestURL.pathname.startsWith("/static/")) {
		event.respondWith(handleStaticRequest(request));
	}
});

async function handleNavigationRequest(request) {
	try {
		return await fetch(request);
	} catch (_error) {
		const cachedOfflinePage = await caches.match(OFFLINE_URL);
		if (cachedOfflinePage) {
			return cachedOfflinePage;
		}

		return new Response("Offline", {
			status: 503,
			statusText: "Service Unavailable",
			headers: {
				"Content-Type": "text/plain; charset=utf-8",
			},
		});
	}
}

async function handleStaticRequest(request) {
	const cachedResponse = await caches.match(request);
	if (cachedResponse) {
		return cachedResponse;
	}

	try {
		const networkResponse = await fetch(request);
		if (networkResponse && networkResponse.ok) {
			const cache = await caches.open(CACHE_NAME);
			await cache.put(request, networkResponse.clone());
		}

		return networkResponse;
	} catch (_error) {
		const cachedOfflinePage = await caches.match(OFFLINE_URL);
		if (request.destination === "document" && cachedOfflinePage) {
			return cachedOfflinePage;
		}

		return new Response("", {
			status: 504,
			statusText: "Gateway Timeout",
		});
	}
}
