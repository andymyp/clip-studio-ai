import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { PaginatedResponse, RenderJob } from "@/lib/types";

export type RenderListParams = {
  page: number;
  pageSize: number;
  sort: "newest" | "oldest" | "progress";
  status?: string;
};

export type RenderedClipListParams = {
  page: number;
  pageSize: number;
  sort: "newest" | "oldest" | "score";
  search?: string;
};

export function useRenderJobs(params: RenderListParams) {
  return useQuery({
    queryKey: ["render-jobs", params],
    queryFn: async () =>
      (await api.get<PaginatedResponse<RenderJob>>("/renders", {
        params: {
          page: params.page,
          page_size: params.pageSize,
          sort: params.sort,
          status: params.status === "all" ? undefined : params.status,
        },
      })).data,
    placeholderData: keepPreviousData,
  });
}

export function useRenderedClips(params: RenderedClipListParams) {
  return useQuery({
    queryKey: ["rendered-clips", params],
    queryFn: async () =>
      (await api.get<PaginatedResponse<RenderJob>>("/clips/rendered", {
        params: {
          page: params.page,
          page_size: params.pageSize,
          sort: params.sort,
          search: params.search || undefined,
        },
      })).data,
    placeholderData: keepPreviousData,
  });
}
