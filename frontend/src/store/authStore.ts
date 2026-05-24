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
  adminSelectedDepartmentId: string | null;
  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setTokens: (token: string, refreshToken: string) => void;
  setAdminSelectedDepartmentId: (id: string | null) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      adminSelectedDepartmentId: null,
      setAuth: (token, user, refreshToken) =>
        set({
          token,
          user,
          isAuthenticated: true,
          ...(refreshToken !== undefined ? { refreshToken } : {}),
        }),
      setTokens: (token, refreshToken) =>
        set({ token, refreshToken }),
      setAdminSelectedDepartmentId: (id) =>
        set({ adminSelectedDepartmentId: id }),
      logout: () =>
        set({ token: null, refreshToken: null, user: null, isAuthenticated: false, adminSelectedDepartmentId: null }),
    }),
    {
      name: 'shiftmaster-auth',
    }
  )
);
