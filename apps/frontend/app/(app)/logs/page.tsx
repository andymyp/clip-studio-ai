"use client";

import {
  ArrowClockwiseIcon,
  CircleNotchIcon,
  ClockCounterClockwiseIcon,
} from "@phosphor-icons/react";
import { useState } from "react";
import { toast } from "sonner";

import { PageHeading } from "@/components/page-heading";
import { DataPagination } from "@/components/data-pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useRetryRender } from "@/hooks/mutations/use-retry-render";
import { useRenderJobs } from "@/hooks/queries/use-render-jobs";
import { useRenderJobEvents } from "@/hooks/sse/use-render-job-events";
import { apiErrorMessage } from "@/lib/api";

const statusClasses: Record<string, string> = {
  queued: "border-amber-200 bg-amber-50 text-amber-700",
  pending: "border-amber-200 bg-amber-50 text-amber-700",
  processing: "border-blue-200 bg-blue-50 text-blue-700",
  completed: "border-emerald-200 bg-emerald-50 text-emerald-700",
  failed: "border-rose-200 bg-rose-50 text-rose-700",
};
const jobsPerPage = 10;

export default function LogsPage() {
  useRenderJobEvents();
  const retry = useRetryRender();
  const [status, setStatus] = useState("all");
  const [sort, setSort] = useState<"newest" | "oldest" | "progress">("newest");
  const [page, setPage] = useState(1);
  const jobs = useRenderJobs({ page, pageSize: jobsPerPage, sort, status });
  const rows = jobs.data?.items ?? [];

  async function retryJob(id: string) {
    try {
      await retry.mutateAsync(id);
      toast.success("Render queued again.");
    } catch (error) {
      toast.error(apiErrorMessage(error));
    }
  }

  return (
    <div className="space-y-7">
      <PageHeading
        eyebrow="Rendering"
        title="Render jobs"
        description="Track every selected clip from subtitle generation through final export."
        action={
          <Button variant="outline" onClick={() => void jobs.refetch()} disabled={jobs.isFetching}>
            <ClockCounterClockwiseIcon className={jobs.isFetching ? "animate-spin" : ""} />
            Refresh
          </Button>
        }
      />
      <div className="flex flex-col gap-3 sm:flex-row sm:justify-end">
        <Select value={status} onValueChange={(value) => { setStatus(value); setPage(1); }}>
          <SelectTrigger className="bg-white sm:w-44"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="queued">Queued</SelectItem>
            <SelectItem value="processing">Processing</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>
        <Select value={sort} onValueChange={(value: "newest" | "oldest" | "progress") => { setSort(value); setPage(1); }}>
          <SelectTrigger className="bg-white sm:w-44"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="newest">Newest first</SelectItem>
            <SelectItem value="oldest">Oldest first</SelectItem>
            <SelectItem value="progress">Highest progress</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <Card className="overflow-x-auto border-black/8 bg-white p-0 shadow-none">
        <table className="w-full min-w-[760px] text-left text-sm">
          <thead className="border-b bg-zinc-50 text-xs uppercase text-muted-foreground">
            <tr><th className="p-4">Clip</th><th>Status</th><th>Profile / QC</th><th>Progress</th><th>Updated</th><th className="pr-4 text-right">Action</th></tr>
          </thead>
          <tbody className="divide-y">
            {rows.map((job) => (
              <tr key={job.id}>
                <td className="max-w-xs p-4">
                  <p className="truncate font-semibold">{job.video_title}</p>
                  <p className="truncate text-xs text-muted-foreground">{job.message || job.error}</p>
                </td>
                <td><Badge className={statusClasses[job.status]}>{job.status}</Badge></td>
                <td>
                  <div className="flex flex-col items-start gap-1">
                    <Badge className="border border-zinc-200 bg-white text-zinc-700">
                      {job.platform_profile === "smart" ? "Smart AI crop" : job.platform_profile}
                    </Badge>
                    {job.status === "completed" && (
                      <span className={job.quality_passed ? "text-xs text-emerald-700" : "text-xs text-rose-700"}>
                        {job.quality_passed ? `QC passed · ${job.output_width}×${job.output_height}` : "QC not verified"}
                      </span>
                    )}
                  </div>
                </td>
                <td className="pr-5 font-medium tabular-nums">
                  {Math.round(job.progress)}%
                </td>
                <td className="text-xs text-muted-foreground">{new Date(job.updated_at).toLocaleString()}</td>
                <td className="pr-4 text-right">
                  {job.status === "failed" && (
                    <Button size="sm" variant="outline" disabled={retry.isPending} onClick={() => void retryJob(job.id)}>
                      {retry.isPending ? <CircleNotchIcon className="animate-spin" /> : <ArrowClockwiseIcon />} Retry
                    </Button>
                  )}
                </td>
              </tr>
            ))}
            {!jobs.isLoading && rows.length === 0 && (
              <tr><td colSpan={6} className="p-16 text-center text-muted-foreground">No render jobs found.</td></tr>
            )}
          </tbody>
        </table>
      </Card>
      {!jobs.isLoading && rows.length > 0 && (
        <DataPagination
          page={page}
          pageSize={jobsPerPage}
          totalItems={jobs.data?.total_items ?? 0}
          onPageChange={setPage}
          itemLabel="jobs"
        />
      )}
    </div>
  );
}
