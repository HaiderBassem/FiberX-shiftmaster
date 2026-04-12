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
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (token: string, user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      isAuthenticated: false,
      setAuth: (token, user) => set({ token, user, isAuthenticated: true }),
      logout: () => set({ token: null, user: null, isAuthenticated: false }),
    }),
    {
      name: 'shiftmaster-auth',
    }
  )
);
