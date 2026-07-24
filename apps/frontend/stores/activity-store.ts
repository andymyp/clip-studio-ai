"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ActivityLog = {
  id: string;
  action: string;
  detail: string;
  status: "success" | "pending" | "error";
  createdAt: string;
};

type ActivityState = {
  logs: ActivityLog[];
  addLog: (log: Omit<ActivityLog, "id" | "createdAt">) => void;
  clearLogs: () => void;
};

export const useActivityStore = create<ActivityState>()(
  persist(
    (set) => ({
      logs: [],
      addLog: (log) =>
        set((state) => ({
          logs: [
            {
              ...log,
              id: crypto.randomUUID(),
              createdAt: new Date().toISOString(),
            },
            ...state.logs,
          ].slice(0, 100),
        })),
      clearLogs: () => set({ logs: [] }),
    }),
    { name: "clipstudio-activity" },
  ),
);
