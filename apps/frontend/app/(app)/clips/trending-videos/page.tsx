"use client";

import {
  ArrowLeftIcon,
  ArrowSquareOutIcon,
  ChartLineUpIcon,
  ChatCircleIcon,
  EyeIcon,
  FilmStripIcon,
  HeartIcon,
  CircleNotchIcon,
  PlayIcon,
  SparkleIcon,
} from "@phosphor-icons/react";
import Image from "next/image";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

import { PageHeading } from "@/components/page-heading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ResponsiveModal } from "@/components/ui/responsive-modal";
import { useCreateClipAnalysis } from "@/hooks/mutations/use-create-clip-analysis";
import { useVideoDiscovery } from "@/hooks/queries/use-video-discovery";
import { apiErrorMessage } from "@/lib/api";
import type { VideoSearchResult } from "@/lib/types";

const metricFormatter = new Intl.NumberFormat("en", { notation: "compact" });

export default function TrendingVideosPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const sourceURL = searchParams.get("url")?.trim() || undefined;
  const discovery = useVideoDiscovery({ url: sourceURL });
  const [preview, setPreview] = useState<VideoSearchResult | null>(null);
  const createAnalysis = useCreateClipAnalysis();

  function clipVideo(video: VideoSearchResult) {
    createAnalysis.mutate(video, {
      onSuccess: (job) => {
        setPreview(null);
        router.push(`/clips/review/${encodeURIComponent(job.external_id)}`);
      },
    });
  }

  return (
    <div className="space-y-7">
      <div>
        <Button asChild variant="ghost" className="-ml-2 mb-4">
          <Link href="/clips">
            <ArrowLeftIcon />
            Back to clips
          </Link>
        </Button>
        <PageHeading
          eyebrow={sourceURL ? "Video found" : "Discovery"}
          title={sourceURL ? "Ready to clip" : "Trending videos"}
          description={
            sourceURL
              ? "Preview the video before sending it to the clipping workflow."
              : "Viral and trending videos discovered across YouTube and Reddit."
          }
        />
      </div>

      {discovery.isLoading ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: sourceURL ? 1 : 6 }).map((_, index) => (
            <div key={index} className="overflow-hidden rounded-2xl border bg-white">
              <div className="aspect-video animate-pulse bg-zinc-200" />
              <div className="space-y-3 p-4">
                <div className="h-5 animate-pulse rounded bg-zinc-200" />
                <div className="h-5 w-2/3 animate-pulse rounded bg-zinc-200" />
              </div>
            </div>
          ))}
        </div>
      ) : discovery.isError ? (
        <StatusPanel
          title="Discovery unavailable"
          detail={apiErrorMessage(discovery.error)}
        />
      ) : discovery.data?.length ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {discovery.data.map((video) => (
            <DiscoveryCard
              key={`${video.platform}-${video.external_id}`}
              video={video}
              onPreview={() => setPreview(video)}
            />
          ))}
        </div>
      ) : (
        <StatusPanel
          title="No videos found"
          detail="Try another video link or search again later."
        />
      )}

      <ResponsiveModal
        open={Boolean(preview)}
        onOpenChange={(open) => !open && setPreview(null)}
        title="Video preview"
        description={preview ? `Discovered on ${capitalize(preview.platform)}` : undefined}
        className="overflow-y-auto sm:max-w-4xl"
      >
        {preview && (
          <VideoPreview
            video={preview}
            clipping={createAnalysis.isPending}
            onClip={() => clipVideo(preview)}
          />
        )}
      </ResponsiveModal>
    </div>
  );
}

function DiscoveryCard({
  video,
  onPreview,
}: {
  video: VideoSearchResult;
  onPreview: () => void;
}) {
  return (
    <Card className="group gap-0 overflow-hidden border-black/8 bg-white py-0 transition duration-300 hover:-translate-y-1 hover:shadow-[0_20px_50px_-24px_rgba(31,18,52,0.35)]">
      <div className="relative aspect-video overflow-hidden bg-violet-50">
        {video.thumbnail ? (
          <Image
            src={video.thumbnail}
            alt=""
            fill
            unoptimized
            sizes="(max-width: 768px) 100vw, (max-width: 1280px) 50vw, 33vw"
            className="object-cover transition duration-500 group-hover:scale-[1.04]"
          />
        ) : (
          <div className="absolute inset-0 grid place-items-center">
            <FilmStripIcon className="size-10 text-violet-300" />
          </div>
        )}
        <div className="absolute inset-0 bg-linear-to-t from-black/50 via-transparent to-transparent" />
        <Badge className="absolute left-3 top-3 border-white/20 bg-black/60 text-white">
          {video.platform}
        </Badge>
        {video.duration > 0 && (
          <span className="absolute bottom-3 right-3 rounded-md bg-black/70 px-2 py-1 text-xs font-medium text-white">
            {formatDuration(video.duration)}
          </span>
        )}
      </div>
      <div className="p-4">
        <h2 className="line-clamp-2 min-h-12 font-heading font-semibold leading-6">
          {video.title}
        </h2>
        {(video.youtube_username || video.channel_title) && (
          <p className="mt-1 truncate text-xs font-medium text-violet-700">
            {video.youtube_username || video.channel_title}
          </p>
        )}
        <VideoMetrics video={video} compact />
        <Button variant="outline" className="mt-4 w-full rounded-xl" onClick={onPreview}>
          <PlayIcon />
          Preview video
        </Button>
      </div>
    </Card>
  );
}

function VideoPreview({
  video,
  clipping,
  onClip,
}: {
  video: VideoSearchResult;
  clipping: boolean;
  onClip: () => void;
}) {
  return (
    <div className="grid gap-6 md:grid-cols-[minmax(0,1.2fr)_minmax(280px,0.8fr)]">
      <div className="overflow-hidden rounded-2xl bg-black">
        {video.platform === "youtube" && video.embed_url ? (
          <iframe
            src={`${video.embed_url}?autoplay=0&rel=0`}
            title={video.title}
            className="aspect-video w-full"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
            allowFullScreen
          />
        ) : video.media_url ? (
          <video
            src={video.media_url}
            poster={video.thumbnail}
            controls
            playsInline
            preload="metadata"
            className="aspect-video w-full object-contain"
          />
        ) : (
          <div className="relative aspect-video">
            {video.thumbnail && (
              <Image src={video.thumbnail} alt="" fill unoptimized className="object-cover" />
            )}
          </div>
        )}
      </div>
      <div className="min-w-0 space-y-5">
        <div>
          <div className="flex flex-wrap gap-2">
            <Badge className="border-violet-200 bg-violet-50 text-violet-700">
              {video.platform}
            </Badge>
            {video.reusable && (
              <Badge className="border-emerald-200 bg-emerald-50 text-emerald-700">
                Reusable · CC BY
              </Badge>
            )}
          </div>
          <h2 className="mt-3 font-heading text-xl font-semibold leading-7">
            {video.title}
          </h2>
          {(video.youtube_username || video.channel_title) && (
            <p className="mt-2 text-sm font-medium text-violet-700">
              {video.youtube_username || video.channel_title}
            </p>
          )}
        </div>
        <VideoMetrics video={video} />
        <div className="flex flex-col-reverse gap-2 sm:flex-row">
          <Button asChild variant="outline" className="flex-1">
            <a href={video.url} target="_blank" rel="noreferrer">
              Source
              <ArrowSquareOutIcon />
            </a>
          </Button>
          <Button
            className="flex-1"
            disabled={clipping || video.platform !== "youtube"}
            onClick={onClip}
          >
            {clipping ? <CircleNotchIcon className="animate-spin" /> : <SparkleIcon />}
            {clipping
              ? "Queuing"
              : video.platform === "youtube"
                ? "Clip"
                : "YouTube only"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function VideoMetrics({
  video,
  compact = false,
}: {
  video: VideoSearchResult;
  compact?: boolean;
}) {
  const metrics =
    video.platform === "reddit"
      ? [
          { icon: ChartLineUpIcon, value: video.score, label: "score" },
          { icon: ChatCircleIcon, value: video.comments, label: "comments" },
        ]
      : [
          { icon: EyeIcon, value: video.views, label: "views" },
          { icon: HeartIcon, value: video.likes, label: "likes" },
          { icon: ChatCircleIcon, value: video.comments, label: "comments" },
        ];
  return (
    <div className={`flex flex-wrap gap-3 text-muted-foreground ${compact ? "mt-3 text-xs" : "text-sm"}`}>
      {metrics.map((metric) => (
        <span key={metric.label} className="flex items-center gap-1.5">
          <metric.icon className="size-4" />
          {metricFormatter.format(metric.value)} {compact ? "" : metric.label}
        </span>
      ))}
    </div>
  );
}

function StatusPanel({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-black/15 bg-white/60 py-16 text-center">
      <FilmStripIcon className="mx-auto size-8 text-violet-400" />
      <h2 className="mt-3 font-heading text-lg font-semibold">{title}</h2>
      <p className="mx-auto mt-2 max-w-lg text-sm text-muted-foreground">{detail}</p>
    </div>
  );
}

function formatDuration(seconds: number) {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remaining = Math.floor(seconds % 60);
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(remaining).padStart(2, "0")}`
    : `${minutes}:${String(remaining).padStart(2, "0")}`;
}

function capitalize(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
