self.addEventListener('install', function(event) {
  self.skipWaiting();
});

self.addEventListener('activate', function(event) {
  event.waitUntil(clients.claim());
});

self.addEventListener('push', function(event) {
  let data = {};
  if (event.data) {
    try {
      data = event.data.json();
    } catch (e) {
      data = { title: "New Notification", body: event.data.text() };
    }
  }

  const options = {
    body: data.body || 'You have a new update.',
    data: {
      url: data.url || '/'
    },
    vibrate: [200, 100, 200]
  };

  const notificationPromise = self.registration.showNotification(data.title || 'ShiftMaster', options);

  // Still send message to client so they hear the ding and see the toast
  const messagePromise = clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function(windowClients) {
    for (let i = 0; i < windowClients.length; i++) {
      windowClients[i].postMessage({
        type: 'PUSH_NOTIFICATION',
        payload: data
      });
    }
  });

  event.waitUntil(Promise.all([notificationPromise, messagePromise]));
});

self.addEventListener('notificationclick', function(event) {
  event.notification.close();
  
  const targetUrl = event.notification.data.url || '/';

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function(windowClients) {
      // If a window is already open, focus it and navigate
      for (let i = 0; i < windowClients.length; i++) {
        const client = windowClients[i];
        if (client.url.includes(self.location.origin) && 'focus' in client) {
          client.navigate(targetUrl);
          return client.focus();
        }
      }
      // Otherwise open a new window
      if (clients.openWindow) {
        return clients.openWindow(targetUrl);
      }
    })
  );
});
