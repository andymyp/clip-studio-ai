import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { RenderJob } from "@/lib/types";

export function useRenderJobs() {
  return useQuery({
    queryKey: ["render-jobs"],
    queryFn: async () => (await api.get<RenderJob[]>("/renders")).data,
  });
}

export function useRenderedClips() {
  return useQuery({
    queryKey: ["rendered-clips"],
    queryFn: async () => (await api.get<RenderJob[]>("/clips/rendered")).data,
  });
}
