import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { AnalysisJob, Video } from "@/lib/types";

export function useVideoSearch(keyword: string, enabled = true) {
  return useQuery({
    queryKey: ["videos", "search", keyword],
    queryFn: async () => {
      const { data } = await api.get<Video[]>("/api/videos/search", {
        params: { keyword },
      });
      return data;
    },
    enabled: enabled && keyword.trim().length >= 2,
  });
}

export function useAnalyzeVideo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (videoID: string) => {
      const { data } = await api.post<AnalysisJob>(`/api/videos/${videoID}/analyze`);
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["jobs"] });
    },
  });
}
