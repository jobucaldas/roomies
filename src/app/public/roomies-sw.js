self.addEventListener('push', (event) => {
  let payload = {};
  try { payload = event.data ? event.data.json() : {}; } catch (_) {}
  const id = String(payload.id || payload.notification_id || 'roomies');
  event.waitUntil(self.registration.showNotification('Roomies', {
    body: 'You have a Roomies notification.',
    tag: `roomies-${id}`,
    renotify: false,
    data: { houseId: payload.house_id || '' }
  }));
});
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const path = event.notification.data && event.notification.data.houseId ? `/houses/${encodeURIComponent(event.notification.data.houseId)}` : '/';
  event.waitUntil(clients.matchAll({type: 'window', includeUncontrolled: true}).then((windows) => {
    const existing = windows.find((client) => new URL(client.url).pathname === path);
    return existing ? existing.focus() : clients.openWindow(path);
  }));
});
