// ShiftMaster service worker.
//
// This worker exists only to deliver push notifications. It deliberately does no
// caching and installs no fetch handler, so it can never serve a stale build.

self.addEventListener('install', function () {
  self.skipWaiting();
});

self.addEventListener('activate', function (event) {
  event.waitUntil(clients.claim());
});

self.addEventListener('push', function (event) {
  let data = {};
  if (event.data) {
    try {
      data = event.data.json();
    } catch (e) {
      data = { title: 'New Notification', body: event.data.text() };
    }
  }

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (windowClients) {
      // Tell every open tab that something arrived. The page treats this purely
      // as a signal to refetch; it does not render this payload.
      for (var i = 0; i < windowClients.length; i++) {
        windowClients[i].postMessage({ type: 'PUSH_NOTIFICATION', payload: data });
      }

      var appIsVisible = windowClients.some(function (client) {
        return client.visibilityState === 'visible';
      });

      // When the user is already looking at the app, the in-app toast is the
      // notification. Raising an operating-system notification as well would
      // announce the same event twice, with two different sounds.
      if (appIsVisible) {
        return undefined;
      }

      return self.registration.showNotification(data.title || 'ShiftMaster', {
        body: data.body || 'You have a new update.',
        icon: data.icon || '/favicon.svg',
        badge: '/favicon.svg',
        // Collapses repeats in the notification centre rather than stacking them.
        tag: data.id || 'shiftmaster-notification',
        renotify: false,
        data: { url: data.url || '/' },
      }).catch(function (err) {
        console.error('Notification error:', err);
      });
    })
  );
});

self.addEventListener('notificationclick', function (event) {
  event.notification.close();

  var targetUrl = (event.notification.data && event.notification.data.url) || '/';

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (windowClients) {
      for (var i = 0; i < windowClients.length; i++) {
        var client = windowClients[i];
        if (client.url.indexOf(self.location.origin) === 0 && 'focus' in client) {
          if ('navigate' in client) {
            client.navigate(targetUrl);
          }
          return client.focus();
        }
      }
      if (clients.openWindow) {
        return clients.openWindow(targetUrl);
      }
      return undefined;
    })
  );
});
