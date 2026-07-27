"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

type RecommendationSearchState = {
  language: string;
  topic: string;
  hydrated: boolean;
  setSearch: (language: string, topic: string) => void;
  setHydrated: (hydrated: boolean) => void;
};

export const useRecommendationSearchStore = create<RecommendationSearchState>()(
  persist(
    (set) => ({
      language: "en",
      topic: "trending",
      hydrated: false,
      setSearch: (language, topic) => set({ language, topic }),
      setHydrated: (hydrated) => set({ hydrated }),
    }),
    {
      name: "clipstudio-recommendation-search",
      partialize: ({ language, topic }) => ({ language, topic }),
      onRehydrateStorage: () => (state) => state?.setHydrated(true),
    },
  ),
);
