import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { JobLog } from "@/lib/types";

export function useJobLogs() {
  return useQuery({
    queryKey: ["jobs", "logs"],
    queryFn: async () => {
      const { data } = await api.get<JobLog[]>("/api/logs");
      return data;
    },
    refetchInterval: (query) => {
      const jobs = query.state.data;
      return jobs?.some((job) =>
        ["queued", "pending", "processing"].includes(job.status),
      )
        ? 3_000
        : false;
    },
  });
}
