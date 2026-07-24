"use client";

import { ArrowSquareOut, MagicWand, Play } from "@phosphor-icons/react";
import Image from "next/image";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { Video } from "@/lib/types";

type VideoCardProps = {
  video: Video;
  analyzing?: boolean;
  onAnalyze?: (video: Video) => void;
};

const viewFormatter = new Intl.NumberFormat("en", { notation: "compact" });

export function VideoCard({ video, analyzing = false, onAnalyze }: VideoCardProps) {
  return (
    <Card className="group overflow-hidden border-black/8 bg-white transition duration-300 hover:-translate-y-1 hover:shadow-[0_20px_50px_-24px_rgba(31,18,52,0.35)]">
      <div className="relative aspect-video overflow-hidden bg-linear-to-br from-violet-100 via-fuchsia-50 to-amber-100">
        {video.thumbnail ? (
          <Image
            src={video.thumbnail}
            alt=""
            fill
            unoptimized
            sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
            className="object-cover transition duration-500 group-hover:scale-[1.04]"
          />
        ) : (
          <div className="absolute inset-0 grid place-items-center">
            <div className="grid size-14 place-items-center rounded-full bg-white/80 text-violet-700 shadow-lg backdrop-blur">
              <Play weight="fill" className="size-6 translate-x-0.5" />
            </div>
          </div>
        )}
        <div className="absolute inset-x-0 bottom-0 h-20 bg-linear-to-t from-black/50 to-transparent" />
        <Badge className="absolute left-3 top-3 border-white/20 bg-black/55 text-white backdrop-blur-md">
          {video.platform}
        </Badge>
        <span className="absolute bottom-3 left-3 text-xs font-medium text-white/90">
          {viewFormatter.format(video.views)} views
        </span>
      </div>

      <div className="p-4">
        <h3 className="line-clamp-2 min-h-12 font-heading text-base font-semibold leading-6 tracking-[-0.02em]">
          {video.title}
        </h3>
        <div className="mt-4 flex gap-2">
          <Button asChild variant="outline" className="flex-1 rounded-xl">
            <a href={video.url} target="_blank" rel="noreferrer">
              <ArrowSquareOut />
              View
            </a>
          </Button>
          <Button
            className="flex-1 rounded-xl"
            disabled={!video.id || analyzing}
            onClick={() => onAnalyze?.(video)}
          >
            <MagicWand className={analyzing ? "animate-pulse" : ""} />
            {analyzing ? "Queuing" : "Analyze"}
          </Button>
        </div>
      </div>
    </Card>
  );
}
