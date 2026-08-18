"use client";

import {
  ArrowLeftIcon,
  CheckSquareIcon,
  CircleNotchIcon,
  FilmStripIcon,
} from "@phosphor-icons/react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";

import { PageHeading } from "@/components/page-heading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { ResponsiveModal } from "@/components/ui/responsive-modal";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { useCreateRenders } from "@/hooks/mutations/use-create-renders";
import { useClipAnalysisEvents } from "@/hooks/sse/use-clip-analysis-events";
import { apiErrorMessage } from "@/lib/api";
import type { GeneratedClip } from "@/lib/types";
import { renderClipsSchema, type RenderClipsValues } from "@/lib/validations";

const workerURL = process.env.NEXT_PUBLIC_WORKER_URL ?? "http://localhost:3002";

export default function ReviewClipsPage() {
  const { jobId: externalID } = useParams<{ jobId: string }>();
  const router = useRouter();
  const job = useClipAnalysisEvents(externalID);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [playingClipID, setPlayingClipID] = useState<string | null>(null);
  const [renderOpen, setRenderOpen] = useState(false);
  const createRenders = useCreateRenders();
  const renderForm = useForm<RenderClipsValues>({
    resolver: zodResolver(renderClipsSchema),
    defaultValues: {
      content_style: "auto",
      watermark_text: "",
      rights_confirmed: false,
    },
  });

  function submitRender(values: RenderClipsValues) {
    createRenders.mutate(
      {
        external_id: externalID,
        clip_ids: [...selected],
        content_style: values.content_style,
        watermark_text: values.watermark_text,
        rights_confirmed: values.rights_confirmed,
      },
      {
        onSuccess: (jobs) => {
          toast.success(`${jobs.length} render jobs queued`);
          setRenderOpen(false);
          router.push("/logs");
        },
        onError: (error) => toast.error(apiErrorMessage(error)),
      },
    );
  }

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
          <Link href="/clips/recommendations">
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
            onClick={() => setRenderOpen(true)}
          >
            <CheckSquareIcon />
            Render selected ({selected.size})
          </Button>
        </div>
      </div>

      <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        {job.data.clips
          .toSorted((a, b) => b.score - a.score)
          .slice(0, 5)
          .map((clip, index) => (
          <CandidateCard
            key={clip.id}
            clip={clip}
            rank={index + 1}
            playing={playingClipID === clip.id}
            onPlay={() => setPlayingClipID(clip.id)}
            selected={selected.has(clip.id)}
            onSelectedChange={(checked) => toggle(clip.id, checked)}
          />
          ))}
      </div>

      <ResponsiveModal
        open={renderOpen}
        onOpenChange={setRenderOpen}
        title="Render selected clips"
        description={`${selected.size} clips will be rendered in 9:16 with animated subtitles.`}
      >
        <Form {...renderForm}>
          <form onSubmit={renderForm.handleSubmit(submitRender)} className="space-y-5">
            <fieldset disabled={createRenders.isPending}>
              <FormField
                control={renderForm.control}
                name="content_style"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Content style</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Choose a content style" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="auto">Auto detect / general</SelectItem>
                        <SelectItem value="talking_head">Podcast / talking head</SelectItem>
                        <SelectItem value="gameplay">Gameplay / screen content</SelectItem>
                        <SelectItem value="comedy">Comedy / funny</SelectItem>
                        <SelectItem value="emotional">Sadness / emotional story</SelectItem>
                        <SelectItem value="livestream">Livestream highlight</SelectItem>
                        <SelectItem value="cinematic">Cinematic / visual story</SelectItem>
                      </SelectContent>
                    </Select>
                    <p className="text-xs leading-5 text-muted-foreground">
                      Controls pacing, silence removal, camera framing, and visual beat length.
                    </p>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={renderForm.control}
                name="watermark_text"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Watermark text</FormLabel>
                    <FormControl>
                      <Input placeholder="@yourbrand" maxLength={100} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="mt-5 rounded-xl border border-violet-200 bg-violet-50 p-4">
                <p className="text-sm font-semibold text-violet-950">
                  Smart AI Auto Crop
                </p>
                <p className="mt-1 text-xs leading-5 text-violet-800">
                  Adapts framing and pacing to the selected content. Gameplay keeps
                  the full screen visible; speaker-focused styles use a stable face crop.
                </p>
              </div>
              <FormField
                control={renderForm.control}
                name="rights_confirmed"
                render={({ field }) => (
                  <FormItem className="mt-5 flex items-start gap-3 rounded-xl border p-4">
                    <FormControl>
                      <Checkbox
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        aria-label="Confirm content reuse rights"
                      />
                    </FormControl>
                    <div>
                      <FormLabel>I have permission to reuse this content</FormLabel>
                      <p className="mt-1 text-xs leading-5 text-muted-foreground">
                        Confirm the source license or creator permission before rendering.
                      </p>
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />
            </fieldset>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setRenderOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createRenders.isPending}>
                {createRenders.isPending && <CircleNotchIcon className="animate-spin" />}
                Render
              </Button>
            </div>
          </form>
        </Form>
      </ResponsiveModal>
    </div>
  );
}

function CandidateCard({
  clip,
  rank,
  playing,
  onPlay,
  selected,
  onSelectedChange,
}: {
  clip: GeneratedClip;
  rank: number;
  playing: boolean;
  onPlay: () => void;
  selected: boolean;
  onSelectedChange: (checked: boolean) => void;
}) {
  const mediaURL = new URL(clip.media_url, workerURL).toString();
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (!playing && videoRef.current && !videoRef.current.paused) {
      videoRef.current.pause();
    }
  }, [playing]);

  return (
    <Card
      className={`overflow-hidden border-black/8 bg-white p-0 transition ${
        selected ? "border-primary ring-2 ring-primary/20" : ""
      }`}
    >
      <video
        ref={videoRef}
        src={mediaURL}
        controls
        onPlay={onPlay}
        playsInline
        preload="metadata"
        className="aspect-video max-h-128 w-full bg-black object-contain"
      />
      <label>
        <div className="space-y-4 p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="flex flex-wrap gap-2">
                <Badge>#{rank} ranked</Badge>
                <Badge className="border-violet-200 bg-violet-50 text-violet-700">
                  {Math.round(clip.score)} score
                </Badge>
              </div>
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
      </label>
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
        <Link href="/clips/recommendations">Return to recommendations</Link>
      </Button>
    </div>
  );
}

function formatTime(seconds: number) {
  const minutes = Math.floor(seconds / 60);
  const remaining = Math.floor(seconds % 60);
  return `${minutes}:${String(remaining).padStart(2, "0")}`;
}
