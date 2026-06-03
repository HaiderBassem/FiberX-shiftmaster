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
    icon: data.icon || '/icon-192x192.png',
    badge: '/icon-192x192.png',
    data: {
      url: data.url || '/'
    },
    vibrate: [200, 100, 200]
  };

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function(windowClients) {
      let isFocused = false;
      for (let i = 0; i < windowClients.length; i++) {
        if (windowClients[i].focused) {
          isFocused = true;
          // Send message to the open window to show in-app toast
          windowClients[i].postMessage({
            type: 'PUSH_NOTIFICATION',
            payload: data
          });
          break;
        }
      }

      // If no window is focused, show native notification
      if (!isFocused) {
        return self.registration.showNotification(data.title || 'ShiftMaster', options);
      }
    })
  );
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
