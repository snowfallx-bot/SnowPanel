import { create } from "zustand";
import { persist } from "zustand/middleware";

interface HostState {
  selectedHostId: number | null;
  setSelectedHostId: (hostId: number | null) => void;
  clearSelectedHost: () => void;
}

export function hostScopeKey(selectedHostId: number | null) {
  return selectedHostId ?? "default";
}

export const useHostStore = create<HostState>()(
  persist(
    (set) => ({
      selectedHostId: null,
      setSelectedHostId: (selectedHostId) => set({ selectedHostId }),
      clearSelectedHost: () => set({ selectedHostId: null })
    }),
    {
      name: "snowpanel-host"
    }
  )
);
