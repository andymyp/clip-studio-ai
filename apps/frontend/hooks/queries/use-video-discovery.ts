import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { VideoSearchResult } from "@/lib/types";

type DiscoveryParams = {
  keywords?: string;
  language?: string;
  url?: string;
};

export function useVideoDiscovery(params: DiscoveryParams) {
  return useQuery({
    queryKey: ["video-discovery", params],
    queryFn: async () => {
      const { data } = await api.get<VideoSearchResult[]>("/videos/search", {
        params,
      });
      return data;
    },
  });
}
