"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

export type AuthUser = {
  id: string;
  email: string;
  created_at: string;
};

type AuthState = {
  user: AuthUser | null;
  accessToken: string | null;
  refreshToken: string | null;
  hydrated: boolean;
  setSession: (user: AuthUser, accessToken: string, refreshToken: string) => void;
  setTokens: (accessToken: string, refreshToken: string) => void;
  clearSession: () => void;
  setHydrated: (hydrated: boolean) => void;
};

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      hydrated: false,
      setSession: (user, accessToken, refreshToken) =>
        set({ user, accessToken, refreshToken }),
      setTokens: (accessToken, refreshToken) => set({ accessToken, refreshToken }),
      clearSession: () => set({ user: null, accessToken: null, refreshToken: null }),
      setHydrated: (hydrated) => set({ hydrated }),
    }),
    {
      name: "clipstudio-auth",
      partialize: ({ user, accessToken, refreshToken }) => ({
        user,
        accessToken,
        refreshToken,
      }),
      onRehydrateStorage: () => (state) => state?.setHydrated(true),
    },
  ),
);
