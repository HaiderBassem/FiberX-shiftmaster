import axios from 'axios';
import { useAuthStore } from '../store/authStore';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  headers: {
    'Content-Type': 'application/json',
  },
  // Uploaded files are authorised by a cookie scoped to /api/uploads, which the
  // server sets on login and refresh. Sending credentials keeps that cookie
  // flowing when the API is on a different origin from the SPA.
  withCredentials: true,
});

/**
 * Ends the session.
 *
 * Clearing local state is not enough on its own: the upload cookie is HttpOnly,
 * so only the server can remove it, and leaving it behind would let the next
 * person at the same browser fetch files from the previous session. The request
 * is best-effort — an expired or offline session must still be able to sign out
 * locally.
 */
export async function signOut(): Promise<void> {
  try {
    await api.post('/auth/logout');
  } catch {
    // Ignore: local state is cleared regardless.
  } finally {
    useAuthStore.getState().logout();
  }
}

api.interceptors.request.use(
  (config) => {
    const store = useAuthStore.getState();
    const token = store.token;
    
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    // Send the selected department header so the backend can scope responses.
    if (store.user?.role === 'admin' && store.adminSelectedDepartmentId) {
      config.headers['X-Department-ID'] = store.adminSelectedDepartmentId;
    } else if (store.user?.role === 'manager' && store.managerSelectedDepartmentId) {
      config.headers['X-Department-ID'] = store.managerSelectedDepartmentId;
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// Flag to prevent multiple simultaneous refresh attempts
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    // If we get a 401 and it's not a login or refresh request, try to refresh the token
    const isLoginRequest = originalRequest?.url?.includes('/auth/login');
    const isRefreshRequest = originalRequest?.url?.includes('/auth/refresh');

    if (error.response?.status === 401 && !isLoginRequest && !isRefreshRequest && !originalRequest._retry) {
      const refreshToken = useAuthStore.getState().refreshToken;

      // If no refresh token, logout immediately
      if (!refreshToken) {
        void signOut().finally(() => window.location.replace('/login'));
        return Promise.reject(error);
      }

      if (isRefreshing) {
        // Queue the request while refresh is in progress
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        })
          .then((token) => {
            originalRequest.headers.Authorization = `Bearer ${token}`;
            return api(originalRequest);
          })
          .catch((err) => Promise.reject(err));
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const { data } = await axios.post(
          `${api.defaults.baseURL}/auth/refresh`,
          { refresh_token: refreshToken },
          { headers: { 'Content-Type': 'application/json' }, withCredentials: true }
        );

        if (data.success) {
          const newAccessToken = data.data.access_token;
          const newRefreshToken = data.data.refresh_token;
          const employee = data.data.employee;

          // Update store with new tokens
          useAuthStore.getState().setAuth(newAccessToken, employee, newRefreshToken);

          // Retry the original request
          originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
          processQueue(null, newAccessToken);

          return api(originalRequest);
        } else {
          // Refresh failed, logout
          processQueue(error, null);
          void signOut().finally(() => window.location.replace('/login'));
          return Promise.reject(error);
        }
      } catch (refreshError) {
        // Refresh request failed, logout
        processQueue(refreshError, null);
        void signOut().finally(() => window.location.replace('/login'));
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default api;
