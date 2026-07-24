"use client";

import {
  ArrowRightIcon as ArrowRight,
  FilmStripIcon as FilmStrip,
  LightningIcon as Lightning,
  MagicWandIcon as MagicWand,
  TrendUpIcon as TrendUp,
} from "@phosphor-icons/react";
import Link from "next/link";
import { useState } from "react";

import { PageHeading } from "@/components/page-heading";
import { VideoCard } from "@/components/video-card";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { apiErrorMessage } from "@/lib/api";
import { useAnalyzeVideo } from "@/hooks/mutations/use-analyze-video";
import { useVideoSearch } from "@/hooks/queries/use-video-search";
import type { Video } from "@/lib/types";
import { useAuthStore } from "@/stores/auth-store";

const stats = [
  {
    label: "Videos discovered",
    value: "—",
    detail: "Your searchable library",
    icon: FilmStrip,
    color: "bg-violet-100 text-violet-700",
  },
  {
    label: "Analysis jobs",
    value: "—",
    detail: "Queued and completed",
    icon: MagicWand,
    color: "bg-amber-100 text-amber-700",
  },
  {
    label: "Creator momentum",
    value: "Ready",
    detail: "Workspace is connected",
    icon: TrendUp,
    color: "bg-emerald-100 text-emerald-700",
  },
];

export default function DashboardPage() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const [analyzingID, setAnalyzingID] = useState<string>();
  const videos = useVideoSearch("podcast", Boolean(accessToken));
  const analyze = useAnalyzeVideo();

  function analyzeVideo(video: Video) {
    if (!video.id) return;
    setAnalyzingID(video.id);
    analyze.mutate(video.id, {
      onSettled: () => setAnalyzingID(undefined),
    });
  }

  const recentVideos = videos.data?.slice(0, 6) ?? [];

  return (
    <div className="space-y-10">
      <PageHeading
        eyebrow="Creator command center"
        title="Good to see you."
        description="Find high-potential moments, start analysis, and keep your clipping workflow moving."
        action={
          <Button asChild className="h-10 rounded-xl px-4">
            <Link href="/clips">
              Explore clips
              <ArrowRight />
            </Link>
          </Button>
        }
      />

      <section className="grid gap-4 md:grid-cols-3">
        {stats.map((stat) => (
          <Card key={stat.label} className="border-black/8 bg-white p-5 shadow-none">
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-sm font-medium text-muted-foreground">{stat.label}</p>
                <p className="mt-3 font-heading text-3xl font-bold tracking-tighter">
                  {stat.label === "Videos discovered" && videos.data
                    ? videos.data.length
                    : stat.value}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">{stat.detail}</p>
              </div>
              <div className={`grid size-10 place-items-center rounded-xl ${stat.color}`}>
                <stat.icon weight="fill" className="size-5" />
              </div>
            </div>
          </Card>
        ))}
      </section>

      <section>
        <div className="mb-5 flex items-end justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 text-violet-700">
              <Lightning weight="fill" className="size-4" />
              <p className="text-xs font-bold uppercase tracking-[0.16em]">Suggested sources</p>
            </div>
            <h2 className="mt-2 font-heading text-2xl font-bold tracking-[-0.035em]">
              Podcast videos
            </h2>
          </div>
          <Link href="/clips" className="text-sm font-semibold text-violet-700 hover:text-violet-900">
            View all
          </Link>
        </div>

        {videos.isLoading ? (
          <VideoSkeletons />
        ) : videos.isError ? (
          <EmptyPanel
            title="Couldn’t load videos"
            description={apiErrorMessage(videos.error)}
          />
        ) : recentVideos.length ? (
          <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
            {recentVideos.map((video) => (
              <VideoCard
                key={video.id ?? video.url}
                video={video}
                analyzing={analyzingID === video.id}
                onAnalyze={analyzeVideo}
              />
            ))}
          </div>
        ) : (
          <EmptyPanel
            title="Your video workspace is clear"
            description="Videos matching “podcast” will appear here once they are added to your account."
          />
        )}
      </section>
    </div>
  );
}

function VideoSkeletons() {
  return (
    <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 3 }).map((_, index) => (
        <div key={index} className="overflow-hidden rounded-2xl border bg-white">
          <div className="aspect-video animate-pulse bg-zinc-200" />
          <div className="space-y-3 p-4">
            <div className="h-4 w-4/5 animate-pulse rounded bg-zinc-200" />
            <div className="h-4 w-2/3 animate-pulse rounded bg-zinc-100" />
            <div className="h-9 animate-pulse rounded-xl bg-zinc-100" />
          </div>
        </div>
      ))}
    </div>
  );
}

function EmptyPanel({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-black/15 bg-white/60 px-6 py-14 text-center">
      <MagicWand className="mx-auto size-7 text-violet-400" />
      <h3 className="mt-3 font-heading font-semibold">{title}</h3>
      <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-muted-foreground">{description}</p>
    </div>
  );
}
