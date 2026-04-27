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
    const token = useAuthStore.getState().token;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Flag to prevent multiple simultaneous 401 redirects
let isRedirecting = false;

api.interceptors.response.use(
  (response) => response,
  (error) => {
<<<<<<< HEAD
    // Only redirect on 401 if it's NOT the login endpoint itself.
    // A 401 on /auth/login means wrong credentials — let the Login
    // component handle it and show the error message.
    if (
      error.response?.status === 401 &&
      !error.config?.url?.includes('/auth/login')
    ) {
      useAuthStore.getState().logout();
      window.location.href = '/login';
=======
    if (error.response?.status === 401) {
      // Don't redirect if we're already on the login page or already redirecting
      const isLoginRequest = error.config?.url?.includes('/auth/login');
      if (!isLoginRequest && !isRedirecting) {
        isRedirecting = true;
        useAuthStore.getState().logout();
        window.location.replace('/login');
        // Reset after a short delay so future 401s can still trigger
        setTimeout(() => { isRedirecting = false; }, 2000);
      }
>>>>>>> f1859212aff21226df2bebcd3380dc27de2a6a4e
    }
    return Promise.reject(error);
  }
);

export default api;
