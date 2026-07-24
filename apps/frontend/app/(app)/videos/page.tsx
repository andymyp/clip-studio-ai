"use client";

import { MagnifyingGlass, VideoCamera } from "@phosphor-icons/react";
import { useState } from "react";

import { PageHeading } from "@/components/page-heading";
import { VideoCard } from "@/components/video-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiErrorMessage } from "@/lib/api";
import { useAnalyzeVideo, useVideoSearch } from "@/lib/queries";
import type { Video } from "@/lib/types";
import { useActivityStore } from "@/stores/activity-store";

export default function VideosPage() {
  const [input, setInput] = useState("podcast");
  const [keyword, setKeyword] = useState("podcast");
  const [analyzingID, setAnalyzingID] = useState<string>();
  const videos = useVideoSearch(keyword);
  const analyze = useAnalyzeVideo();
  const addLog = useActivityStore((state) => state.addLog);

  function analyzeVideo(video: Video) {
    if (!video.id) return;
    setAnalyzingID(video.id);
    analyze.mutate(video.id, {
      onSuccess: (job) =>
        addLog({
          action: "Analysis queued",
          detail: `${video.title} · Job ${job.id.slice(0, 8)}`,
          status: "pending",
        }),
      onError: (error) =>
        addLog({
          action: "Analysis failed",
          detail: apiErrorMessage(error),
          status: "error",
        }),
      onSettled: () => setAnalyzingID(undefined),
    });
  }

  return (
    <div className="space-y-8">
      <PageHeading
        eyebrow="Source library"
        title="Find your next great clip"
        description="Search your connected video library and send promising conversations into analysis."
      />

      <form
        className="flex max-w-2xl gap-2 rounded-2xl border border-black/8 bg-white p-2 shadow-sm"
        onSubmit={(event) => {
          event.preventDefault();
          if (input.trim().length >= 2) setKeyword(input.trim());
        }}
      >
        <div className="relative flex-1">
          <MagnifyingGlass className="absolute left-3.5 top-3.5 size-4 text-muted-foreground" />
          <Input
            className="border-0 pl-10 shadow-none focus-visible:ring-0"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="Search podcasts, interviews, topics…"
            minLength={2}
          />
        </div>
        <Button className="h-11 rounded-xl px-5">Search</Button>
      </form>

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {videos.data ? `${videos.data.length} results for “${keyword}”` : `Searching “${keyword}”`}
        </p>
      </div>

      {videos.isLoading ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }).map((_, index) => (
            <div key={index} className="aspect-[1.2] animate-pulse rounded-2xl bg-white" />
          ))}
        </div>
      ) : videos.isError ? (
        <StatusPanel title="Video search unavailable" detail={apiErrorMessage(videos.error)} />
      ) : videos.data?.length ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {videos.data.map((video) => (
            <VideoCard
              key={video.id ?? video.url}
              video={video}
              analyzing={analyzingID === video.id}
              onAnalyze={analyzeVideo}
            />
          ))}
        </div>
      ) : (
        <StatusPanel
          title="No matching videos"
          detail="Try a broader title or topic, or add videos to this account first."
        />
      )}
    </div>
  );
}

function StatusPanel({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-black/15 bg-white/60 py-16 text-center">
      <VideoCamera className="mx-auto size-8 text-violet-400" />
      <h2 className="mt-3 font-heading text-lg font-semibold">{title}</h2>
      <p className="mx-auto mt-2 max-w-md text-sm text-muted-foreground">{detail}</p>
    </div>
  );
}
