"use client";

import {
  CircleNotchIcon as CircleNotch,
  MagnifyingGlassIcon as MagnifyingGlass,
  VideoCameraIcon as VideoCamera,
} from "@phosphor-icons/react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { PageHeading } from "@/components/page-heading";
import { VideoCard } from "@/components/video-card";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { apiErrorMessage } from "@/lib/api";
import { useAnalyzeVideo } from "@/hooks/mutations/use-analyze-video";
import { useVideoSearch } from "@/hooks/queries/use-video-search";
import type { Video } from "@/lib/types";
import { videoSearchSchema, type VideoSearchValues } from "@/lib/validations";

export default function VideosPage() {
  const [keyword, setKeyword] = useState("podcast");
  const [analyzingID, setAnalyzingID] = useState<string>();
  const videos = useVideoSearch(keyword);
  const analyze = useAnalyzeVideo();
  const form = useForm<VideoSearchValues>({
    resolver: zodResolver(videoSearchSchema),
    defaultValues: { keyword: "podcast" },
  });

  function analyzeVideo(video: Video) {
    if (!video.id) return;
    setAnalyzingID(video.id);
    analyze.mutate(video.id, {
      onSettled: () => setAnalyzingID(undefined),
    });
  }

  async function search(values: VideoSearchValues) {
    setKeyword(values.keyword.trim());
  }

  const submitting = form.formState.isSubmitting || videos.isFetching;

  return (
    <div className="space-y-8">
      <PageHeading
        eyebrow="Source library"
        title="Find your next great clip"
        description="Search your connected video library and send promising conversations into analysis."
      />

      <Form {...form}>
        <form
          className="max-w-2xl"
          onSubmit={form.handleSubmit(search)}
          noValidate
        >
          <fieldset
            disabled={submitting}
            className="flex gap-2 rounded-2xl border border-black/8 bg-white p-2 shadow-sm"
          >
            <FormField
              control={form.control}
              name="keyword"
              render={({ field }) => (
                <FormItem className="min-w-0 flex-1 space-y-1">
                  <div className="relative">
                    <MagnifyingGlass className="absolute left-3.5 top-3.5 size-4 text-muted-foreground" />
                    <FormControl>
                      <Input
                        className="border-0 pl-10 shadow-none focus-visible:ring-0"
                        placeholder="Search podcasts, interviews, topics…"
                        {...field}
                      />
                    </FormControl>
                  </div>
                  <FormMessage className="px-3 pb-1" />
                </FormItem>
              )}
            />
            <Button className="h-11 rounded-xl px-5" disabled={submitting}>
              {submitting && <CircleNotch className="animate-spin" />}
              {submitting ? "Searching" : "Search"}
            </Button>
          </fieldset>
        </form>
      </Form>

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
