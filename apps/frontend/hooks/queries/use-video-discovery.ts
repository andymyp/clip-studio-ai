import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { VideoSearchResult } from "@/lib/types";

type DiscoveryParams = {
  keywords?: string;
  language?: string;
  content_style?: string;
  url?: string;
};

export function useVideoDiscovery(params: DiscoveryParams) {
  return useQuery({
    queryKey: ["video-discovery", params],
    queryFn: async () => {
      const endpoint = params.url ? "/videos/search" : "/videos/recommendations";
      const { data } = await api.get<VideoSearchResult[]>(endpoint, {
        params,
      });
      return data;
    },
  });
}
