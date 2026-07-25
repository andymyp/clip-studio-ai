import { useMutation } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { ClipAnalysisJob, VideoSearchResult } from "@/lib/types";

export function useCreateClipAnalysis() {
  return useMutation({
    mutationFn: async (video: VideoSearchResult) => {
      const { data } = await api.post<ClipAnalysisJob>("/clips/analyze", {
        external_id: video.external_id,
        url: video.url,
        title: video.title,
        platform: video.platform,
        thumbnail: video.thumbnail,
      });
      return data;
    },
  });
}
