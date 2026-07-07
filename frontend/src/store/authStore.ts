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
  can_create_tables?: boolean;
  can_manage_help_docs?: boolean;
  can_manage_services?: boolean;
  ui_preferences?: Record<string, any>;
  profile_image?: string;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  isAuthenticated: boolean;
  adminSelectedDepartmentId: string | null;
  managerSelectedDepartmentId: string | null;
  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setTokens: (token: string, refreshToken: string) => void;
  setAdminSelectedDepartmentId: (id: string | null) => void;
  setManagerSelectedDepartmentId: (id: string | null) => void;
  updateUserPreferences: (prefs: Record<string, any>) => void;
  updateProfileImage: (url: string) => void;
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
      managerSelectedDepartmentId: null,
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
      setManagerSelectedDepartmentId: (id) =>
        set({ managerSelectedDepartmentId: id }),
      updateUserPreferences: (prefs) =>
        set((state) => ({
          user: state.user ? { ...state.user, ui_preferences: { ...state.user.ui_preferences, ...prefs } } : null
        })),
      updateProfileImage: (url) =>
        set((state) => ({
          user: state.user ? { ...state.user, profile_image: url } : null
        })),
      logout: () =>
        set({
          token: null,
          refreshToken: null,
          user: null,
          isAuthenticated: false,
          adminSelectedDepartmentId: null,
          managerSelectedDepartmentId: null,
        }),
    }),
    {
      name: 'shiftmaster-auth',
    }
  )
);
