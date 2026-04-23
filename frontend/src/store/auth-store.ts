import { create } from "zustand";
import { persist } from "zustand/middleware";
import { UserProfile } from "@/types/auth";

interface AuthState {
  hydrated: boolean;
  token: string | null;
  refreshToken: string | null;
  user: UserProfile | null;
  setAuth: (token: string, user: UserProfile, refreshToken?: string | null) => void;
  clearAuth: () => void;
  markHydrated: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      hydrated: false,
      token: null,
      refreshToken: null,
      user: null,
      setAuth: (token, user, refreshToken) =>
        set((state) => ({
          token,
          user,
          refreshToken:
            refreshToken === undefined ? state.refreshToken : refreshToken
        })),
      clearAuth: () => set({ token: null, refreshToken: null, user: null }),
      markHydrated: () => set({ hydrated: true })
    }),
    {
      name: "snowpanel-auth",
      onRehydrateStorage: () => (state) => {
        state?.markHydrated();
      }
    }
  )
);
