import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { VideoSearchResult } from "@/lib/types";

export function useVideoSearch(keyword: string, enabled = true) {
  return useQuery({
    queryKey: ["videos", "search", keyword],
    queryFn: async () => {
      const { data } = await api.get<VideoSearchResult[]>("/videos/search", {
        params: { keyword },
      });
      return data;
    },
    enabled: enabled && keyword.trim().length >= 2,
  });
}
