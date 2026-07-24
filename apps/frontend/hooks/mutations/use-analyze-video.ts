import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@/lib/api";
import type { AnalysisJob } from "@/lib/types";

export function useAnalyzeVideo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (videoID: string) => {
      const { data } = await api.post<AnalysisJob>(`/api/videos/${videoID}/analyze`);
      return data;
    },
    onSuccess: () => {
      toast.success("Analysis queued");
      void queryClient.invalidateQueries({ queryKey: ["jobs"] });
    },
  });
}
