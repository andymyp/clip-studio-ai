"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

type RecommendationSearchState = {
  language: string;
  topic: string;
  contentStyle: "auto" | "talking_head" | "gameplay" | "comedy" | "emotional" | "livestream" | "cinematic";
  hydrated: boolean;
  setSearch: (language: string, topic: string, contentStyle: RecommendationSearchState["contentStyle"]) => void;
  setHydrated: (hydrated: boolean) => void;
};

export const useRecommendationSearchStore = create<RecommendationSearchState>()(
  persist(
    (set) => ({
      language: "en",
      topic: "trending",
      contentStyle: "auto",
      hydrated: false,
      setSearch: (language, topic, contentStyle) => set({ language, topic, contentStyle }),
      setHydrated: (hydrated) => set({ hydrated }),
    }),
    {
      name: "clipstudio-recommendation-search",
      partialize: ({ language, topic, contentStyle }) => ({ language, topic, contentStyle }),
      onRehydrateStorage: () => (state) => state?.setHydrated(true),
    },
  ),
);
