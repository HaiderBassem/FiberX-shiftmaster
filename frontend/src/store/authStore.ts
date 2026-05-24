import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface User {
  id: string;
  employee_code: string;
  first_name: string;
  last_name: string;
  email: string;
  role: 'employee' | 'team_leader' | 'manager' | 'admin';
  department_id: string | null;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setTokens: (token: string, refreshToken: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      setAuth: (token, user, refreshToken) =>
        set({
          token,
          user,
          isAuthenticated: true,
          ...(refreshToken !== undefined ? { refreshToken } : {}),
        }),
      setTokens: (token, refreshToken) =>
        set({ token, refreshToken }),
      logout: () =>
        set({ token: null, refreshToken: null, user: null, isAuthenticated: false }),
    }),
    {
      name: 'shiftmaster-auth',
    }
  )
);
