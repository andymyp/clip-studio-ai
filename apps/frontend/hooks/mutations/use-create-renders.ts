import { useMutation } from "@tanstack/react-query";
import { useProgress } from "@bprogress/next";

import { api } from "@/lib/api";
import type { RenderJob } from "@/lib/types";

export function useCreateRenders() {
  const progress = useProgress();
  return useMutation({
    mutationFn: async (payload: {
      external_id: string;
      clip_ids: string[];
      watermark_text: string;
      content_style:
        | "auto"
        | "talking_head"
        | "gameplay"
        | "comedy"
        | "emotional"
        | "livestream"
        | "cinematic";
      rights_confirmed: boolean;
    }) => {
      progress.start();
      try {
        const { data } = await api.post<RenderJob[]>("/renders", payload);
        return data;
      } finally {
        progress.stop();
      }
    },
  });
}
