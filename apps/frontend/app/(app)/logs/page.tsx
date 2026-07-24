"use client";

import {
  CheckCircleIcon as CheckCircle,
  CircleNotchIcon as CircleNotch,
  ClockCounterClockwiseIcon as ClockCounterClockwise,
  ClockIcon as Clock,
  WarningCircleIcon as WarningCircle,
  XCircleIcon as XCircle,
} from "@phosphor-icons/react";

import { PageHeading } from "@/components/page-heading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useJobLogs } from "@/hooks/queries/use-job-logs";
import type { JobLog } from "@/lib/types";

const statusMeta: Record<
  JobLog["status"],
  { icon: typeof CheckCircle; className: string; label: string }
> = {
  queued: {
    icon: Clock,
    className: "border-amber-200 bg-amber-50 text-amber-700",
    label: "Queued",
  },
  pending: {
    icon: Clock,
    className: "border-amber-200 bg-amber-50 text-amber-700",
    label: "Pending",
  },
  processing: {
    icon: CircleNotch,
    className: "border-blue-200 bg-blue-50 text-blue-700",
    label: "Processing",
  },
  completed: {
    icon: CheckCircle,
    className: "border-emerald-200 bg-emerald-50 text-emerald-700",
    label: "Completed",
  },
  failed: {
    icon: WarningCircle,
    className: "border-rose-200 bg-rose-50 text-rose-700",
    label: "Failed",
  },
  cancelled: {
    icon: XCircle,
    className: "border-zinc-200 bg-zinc-50 text-zinc-600",
    label: "Cancelled",
  },
};

export default function LogsPage() {
  const logs = useJobLogs();

  return (
    <div className="space-y-8">
      <PageHeading
        eyebrow="Clip queue"
        title="Queue logs"
        description="Backend-backed status for your clip analysis jobs. Active jobs refresh automatically."
        action={
          <Button
            variant="outline"
            className="rounded-xl"
            disabled={logs.isFetching}
            onClick={() => void logs.refetch()}
          >
            <ClockCounterClockwise className={logs.isFetching ? "animate-spin" : ""} />
            {logs.isFetching ? "Refreshing" : "Refresh"}
          </Button>
        }
      />

      {logs.isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="h-24 animate-pulse rounded-2xl bg-white" />
          ))}
        </div>
      ) : logs.data?.length ? (
        <Card className="overflow-hidden border-black/8 bg-white shadow-none">
          <div className="divide-y divide-black/6">
            {logs.data.map((log) => {
              const meta = statusMeta[log.status];
              return (
                <div key={log.id} className="flex items-start gap-4 p-4 sm:items-center sm:px-5">
                  <div
                    className={`grid size-10 shrink-0 place-items-center rounded-xl ${meta.className}`}
                  >
                    <meta.icon
                      weight="fill"
                      className={`size-5 ${log.status === "processing" ? "animate-spin" : ""}`}
                    />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="max-w-xl truncate font-heading text-sm font-semibold">
                        {log.video_title || "Untitled video"}
                      </p>
                      <Badge className={meta.className}>{meta.label}</Badge>
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                      <span>{log.message || meta.label}</span>
                      <span>{Math.round(log.progress)}%</span>
                      <span className="font-mono">#{log.id.slice(0, 8)}</span>
                    </div>
                    <div className="mt-2 h-1.5 max-w-xl overflow-hidden rounded-full bg-zinc-100">
                      <div
                        className="h-full rounded-full bg-violet-600 transition-all"
                        style={{ width: `${Math.min(100, Math.max(0, log.progress))}%` }}
                      />
                    </div>
                  </div>
                  <time className="hidden shrink-0 text-xs text-muted-foreground sm:block">
                    {new Intl.DateTimeFormat("en", {
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    }).format(new Date(log.updated_at))}
                  </time>
                </div>
              );
            })}
          </div>
        </Card>
      ) : (
        <div className="rounded-2xl border border-dashed border-black/15 bg-white/60 py-20 text-center">
          <ClockCounterClockwise className="mx-auto size-9 text-violet-400" />
          <h2 className="mt-4 font-heading text-lg font-semibold">The queue is empty</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            Analyze a video and its backend queue status will appear here.
          </p>
        </div>
      )}
    </div>
  );
}
