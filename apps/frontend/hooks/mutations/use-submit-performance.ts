import { useProgress } from "@bprogress/next";
import { useMutation } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { ClipPerformance } from "@/lib/types";
import type { PerformanceFeedbackValues } from "@/lib/validations";

export function useSubmitPerformance(renderID: string) {
  const progress = useProgress();
  return useMutation({
    mutationFn: async (payload: PerformanceFeedbackValues) => {
      progress.start();
      try {
        return (await api.post<ClipPerformance>(`/renders/${renderID}/feedback`, payload)).data;
      } finally {
        progress.stop();
      }
    },
  });
}
