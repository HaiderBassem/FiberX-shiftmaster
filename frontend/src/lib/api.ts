import axios from 'axios';
import { useAuthStore } from '../store/authStore';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use(
  (config) => {
    const store = useAuthStore.getState();
    const token = store.token;
    
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    if (store.user?.role === 'admin' && store.adminSelectedDepartmentId) {
      config.headers['X-Department-ID'] = store.adminSelectedDepartmentId;
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
        useAuthStore.getState().logout();
        window.location.replace('/login');
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
          { headers: { 'Content-Type': 'application/json' } }
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
          useAuthStore.getState().logout();
          window.location.replace('/login');
          return Promise.reject(error);
        }
      } catch (refreshError) {
        // Refresh request failed, logout
        processQueue(refreshError, null);
        useAuthStore.getState().logout();
        window.location.replace('/login');
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default api;
