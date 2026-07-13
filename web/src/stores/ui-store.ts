import { create } from "zustand";

/**
 * UI Store — hanya untuk state UI klien murni lintas-komponen.
 * JANGAN simpan data server (appointments, wallet, dll) di sini.
 * Data server dikelola oleh TanStack Query.
 */

interface UIState {
  /** Whether the mobile sidebar is open */
  sidebarOpen: boolean;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: false,
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
}));
