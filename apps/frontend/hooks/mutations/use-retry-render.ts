import { useProgress } from "@bprogress/next";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { RenderJob } from "@/lib/types";

export function useRetryRender() {
  const queryClient = useQueryClient();
  const progress = useProgress();
  return useMutation({
    mutationFn: async (id: string) => {
      progress.start();
      try {
        const { data } = await api.post<RenderJob>(`/renders/${id}/retry`);
        return data;
      } finally {
        progress.stop();
      }
    },
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["render-jobs"] }),
  });
}
