import React, { useEffect, createContext, useContext } from 'react';
import { useAuthStore } from '@/store/authStore';
import api from '@/lib/api';
import { toast } from 'sonner';

interface NotificationContextType {
  requestPermission: () => Promise<void>;
  permission: NotificationPermission;
}

const NotificationContext = createContext<NotificationContextType>({
  requestPermission: async () => {},
  permission: 'default',
});

export const useNotification = () => useContext(NotificationContext);

// Base64 to Uint8Array helper for VAPID key
function urlBase64ToUint8Array(base64String: string) {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding)
    .replace(/\-/g, '+')
    .replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export const NotificationProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  const [permission, setPermission] = React.useState<NotificationPermission>(
    'Notification' in window ? Notification.permission : 'default'
  );

  // Play sound function using Web Audio API
  const playSound = () => {
    try {
      const AudioContext = window.AudioContext || (window as any).webkitAudioContext;
      if (!AudioContext) return;
      
      const ctx = new AudioContext();
      const osc = ctx.createOscillator();
      const gainNode = ctx.createGain();
      
      osc.type = 'sine';
      osc.frequency.setValueAtTime(880, ctx.currentTime); // A5 note
      osc.frequency.exponentialRampToValueAtTime(440, ctx.currentTime + 0.1);
      
      gainNode.gain.setValueAtTime(0.5, ctx.currentTime);
      gainNode.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.5);
      
      osc.connect(gainNode);
      gainNode.connect(ctx.destination);
      
      osc.start();
      osc.stop(ctx.currentTime + 0.5);
    } catch (err) {
      console.log('Audio play error:', err);
    }
  };

  const subscribeToPush = async () => {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) return;
    
    try {
      const registration = await navigator.serviceWorker.register('/sw.js');
      await navigator.serviceWorker.ready;

      const { data: keyData } = await api.get('/push/public-key');
      const applicationServerKey = urlBase64ToUint8Array(keyData.data.publicKey);

      let subscription = await registration.pushManager.getSubscription();

      if (subscription) {
        // Compare keys or just unsubscribe to be safe if we want to ensure keys match
        // But let's try subscribing directly, if it fails, we unsubscribe
      }

      try {
        subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey
        });
      } catch (subErr) {
        console.log('Subscribe failed, trying to unsubscribe existing...', subErr);
        if (subscription) {
          await subscription.unsubscribe();
        }
        // Try again
        subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey
        });
      }

      await api.post('/push/subscribe', subscription);
      console.log('Successfully subscribed to push notifications');
    } catch (err) {
      console.error('Failed to subscribe to push notifications:', err);
    }
  };

  const requestPermission = async () => {
    if (!('Notification' in window)) return;
    
    const result = await Notification.requestPermission();
    setPermission(result);
    
    if (result === 'granted' && isAuthenticated) {
      await subscribeToPush();
    }
  };

  useEffect(() => {
    if (isAuthenticated && permission === 'granted') {
      subscribeToPush();
    }
  }, [isAuthenticated, permission]);

  useEffect(() => {
    // Listen for messages from the service worker
    const handleMessage = (event: MessageEvent) => {
      if (event.data && event.data.type === 'PUSH_NOTIFICATION') {
        const { title, body } = event.data.payload;
        playSound();
        toast(title || 'New Notification', {
          description: body,
          action: {
            label: 'View',
            onClick: () => {
              if (event.data.payload.url) {
                window.location.href = event.data.payload.url;
              }
            }
          }
        });
      }
    };

    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.addEventListener('message', handleMessage);
    }

    return () => {
      if ('serviceWorker' in navigator) {
        navigator.serviceWorker.removeEventListener('message', handleMessage);
      }
    };
  }, []);

  // Set up SSE connection for guaranteed in-app notifications
  React.useEffect(() => {
    if (!isAuthenticated) return;

    const token = localStorage.getItem('token');
    if (!token) return;

    const eventSource = new EventSource(`/api/notifications/stream?token=${token}`);

    eventSource.onmessage = (event) => {
      if (event.data === 'connected') return;

      try {
        const data = JSON.parse(event.data);
        
        // Ensure we don't duplicate if Service Worker already fired it
        // (We can just trigger the ding and toast, sonner handles duplicates if ids are used, 
        // but for safety we just play it. Since SW postMessage might be delayed or blocked, this is the primary.)
        playSound();
        toast(data.title || 'ShiftMaster', {
          description: data.body || 'You have a new update.',
          action: {
            label: 'View',
            onClick: () => {
              if (data.url) {
                window.location.href = data.url;
              }
            },
          },
          duration: 6000,
        });
      } catch (err) {
        console.error('Failed to parse SSE notification:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.error('SSE connection error:', err);
      eventSource.close();
      // Optional: implement reconnect logic or let it stay closed
    };

    return () => {
      eventSource.close();
    };
  }, [isAuthenticated]);

  return (
    <NotificationContext.Provider value={{ requestPermission, permission }}>
      {children}
    </NotificationContext.Provider>
  );
};
