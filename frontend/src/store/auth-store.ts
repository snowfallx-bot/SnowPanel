import { create } from "zustand";
import { persist } from "zustand/middleware";
import { UserProfile } from "@/types/auth";

interface AuthState {
  hydrated: boolean;
  token: string | null;
  user: UserProfile | null;
  setAuth: (token: string, user: UserProfile) => void;
  clearAuth: () => void;
  markHydrated: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      hydrated: false,
      token: null,
      user: null,
      setAuth: (token, user) => set({ token, user }),
      clearAuth: () => set({ token: null, user: null }),
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
