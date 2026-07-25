"use client";

import {
  ArrowLeftIcon,
  CheckSquareIcon,
  CircleNotchIcon,
  FilmStripIcon,
} from "@phosphor-icons/react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";

import { PageHeading } from "@/components/page-heading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { useClipAnalysisEvents } from "@/hooks/sse/use-clip-analysis-events";
import { apiErrorMessage } from "@/lib/api";
import type { GeneratedClip } from "@/lib/types";

const workerURL = process.env.NEXT_PUBLIC_WORKER_URL ?? "http://localhost:3002";

export default function ReviewClipsPage() {
  const { jobId: externalID } = useParams<{ jobId: string }>();
  const job = useClipAnalysisEvents(externalID);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  function toggle(clipID: string, checked: boolean) {
    setSelected((current) => {
      const next = new Set(current);
      if (checked) next.add(clipID);
      else next.delete(clipID);
      return next;
    });
  }

  if (job.isLoading) return <ProgressState progress={0} message="Loading analysis job" />;
  if (job.isError) {
    return <EmptyState title="Couldn’t load clips" detail={apiErrorMessage(job.error)} />;
  }
  if (!job.data || job.data.status === "queued" || job.data.status === "processing") {
    return (
      <ProgressState
        progress={job.data?.progress ?? 0}
        message={job.data?.message ?? "Waiting for the worker"}
      />
    );
  }
  if (job.data.status === "failed") {
    return (
      <EmptyState
        title="Clip analysis failed"
        detail={job.data.error ?? job.data.message}
      />
    );
  }

  return (
    <div className="space-y-7">
      <div>
        <Button asChild variant="ghost" className="-ml-2 mb-4">
          <Link href="/clips/trending-videos">
            <ArrowLeftIcon />
            Back to discovery
          </Link>
        </Button>
        <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <PageHeading
            eyebrow="AI clip candidates"
            title="Review generated clips"
            description={`${job.data.clips.length} moments found in “${job.data.title}”. Preview and select the clips you want to render.`}
          />
          <Button
            className="h-11 rounded-xl px-4"
            disabled={selected.size === 0}
            onClick={() =>
              toast.info("Rendering is not implemented yet", {
                description: `${selected.size} clips are selected and ready for the render step.`,
              })
            }
          >
            <CheckSquareIcon />
            Render selected ({selected.size})
          </Button>
        </div>
      </div>

      <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        {job.data.clips.map((clip) => (
          <CandidateCard
            key={clip.id}
            clip={clip}
            selected={selected.has(clip.id)}
            onSelectedChange={(checked) => toggle(clip.id, checked)}
          />
        ))}
      </div>
    </div>
  );
}

function CandidateCard({
  clip,
  selected,
  onSelectedChange,
}: {
  clip: GeneratedClip;
  selected: boolean;
  onSelectedChange: (checked: boolean) => void;
}) {
  const mediaURL = new URL(clip.media_url, workerURL).toString();
  return (
    <Card
      className={`overflow-hidden border-black/8 bg-white p-0 transition ${
        selected ? "border-primary ring-2 ring-primary/20" : ""
      }`}
    >
      <video
        src={mediaURL}
        controls
        playsInline
        preload="metadata"
        className="aspect-9/16 max-h-[32rem] w-full bg-black object-contain"
      />
      <div className="space-y-4 p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <Badge className="border-violet-200 bg-violet-50 text-violet-700">
              {Math.round(clip.score)} score
            </Badge>
            <p className="mt-2 text-sm leading-6">{clip.reason}</p>
          </div>
          <Checkbox
            checked={selected}
            onCheckedChange={(checked) => onSelectedChange(checked === true)}
            aria-label="Select clip for rendering"
          />
        </div>
        <p className="text-xs text-muted-foreground">
          {formatTime(clip.start)}–{formatTime(clip.end)} ·{" "}
          {Math.round(clip.end - clip.start)} seconds
        </p>
      </div>
    </Card>
  );
}

function ProgressState({ progress, message }: { progress: number; message: string }) {
  return (
    <div className="mx-auto max-w-xl py-24 text-center">
      <div className="mx-auto grid size-14 place-items-center rounded-2xl bg-violet-100 text-violet-700">
        <CircleNotchIcon className="size-7 animate-spin" />
      </div>
      <p className="mt-5 text-xs font-bold uppercase tracking-[0.16em] text-violet-700">
        Creating clips
      </p>
      <h1 className="mt-2 font-heading text-3xl font-bold">{Math.round(progress)}%</h1>
      <div className="mx-auto mt-5 h-2 max-w-sm overflow-hidden rounded-full bg-violet-100">
        <div
          className="h-full rounded-full bg-primary transition-all"
          style={{ width: `${Math.max(2, progress)}%` }}
        />
      </div>
      <p className="mt-4 text-sm text-muted-foreground">{message}</p>
    </div>
  );
}

function EmptyState({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-black/15 bg-white/60 py-20 text-center">
      <FilmStripIcon className="mx-auto size-9 text-violet-400" />
      <h1 className="mt-4 font-heading text-xl font-semibold">{title}</h1>
      <p className="mx-auto mt-2 max-w-lg text-sm text-muted-foreground">{detail}</p>
      <Button asChild variant="outline" className="mt-5">
        <Link href="/clips/trending-videos">Return to discovery</Link>
      </Button>
    </div>
  );
}

function formatTime(seconds: number) {
  const minutes = Math.floor(seconds / 60);
  const remaining = Math.floor(seconds % 60);
  return `${minutes}:${String(remaining).padStart(2, "0")}`;
}
