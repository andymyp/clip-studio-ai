"use client";

import {
  ArrowRightIcon,
  CalendarBlankIcon,
  FilmStripIcon,
  LinkSimpleIcon,
  MagnifyingGlassIcon,
  PlayIcon,
  SparkleIcon,
} from "@phosphor-icons/react";
import Image from "next/image";
import { useMemo, useState } from "react";

import { PageHeading } from "@/components/page-heading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 6;

type VideoStatus = "ready" | "processing" | "failed";

type LibraryClip = {
  id: string;
  title: string;
  thumbnail: string;
  platform: "YouTube" | "TikTok" | "Instagram";
  status: VideoStatus;
  score: number;
  source: string;
  duration: string;
  createdAt: string;
};

const clips: LibraryClip[] = [
  { id: "1", title: "Creativity becomes inevitable when you build this habit", thumbnail: "https://images.unsplash.com/photo-1478737270239-2f02b77fc618?auto=format&fit=crop&w=1200&q=80", platform: "YouTube", status: "ready", score: 96, source: "The Creative Process Podcast", duration: "00:42", createdAt: "Jul 24, 2026" },
  { id: "2", title: "Stop chasing trends—build an audience that stays", thumbnail: "https://images.unsplash.com/photo-1589903308904-1010c2294adc?auto=format&fit=crop&w=1200&q=80", platform: "YouTube", status: "processing", score: 91, source: "Creator Science", duration: "01:08", createdAt: "Jul 23, 2026" },
  { id: "3", title: "Three storytelling rules every creator should know", thumbnail: "https://images.unsplash.com/photo-1590602847861-f357a9332bbc?auto=format&fit=crop&w=1200&q=80", platform: "TikTok", status: "ready", score: 89, source: "Story Lab", duration: "00:37", createdAt: "Jul 22, 2026" },
  { id: "4", title: "Why consistency wins when talent runs out", thumbnail: "https://images.unsplash.com/photo-1573496359142-b8d87734a5a2?auto=format&fit=crop&w=1200&q=80", platform: "Instagram", status: "ready", score: 87, source: "Growth Conversations", duration: "00:54", createdAt: "Jul 21, 2026" },
  { id: "5", title: "The decision framework every founder needs", thumbnail: "https://images.unsplash.com/photo-1551836022-d5d88e9218df?auto=format&fit=crop&w=1200&q=80", platform: "YouTube", status: "failed", score: 84, source: "Founder Notes", duration: "01:12", createdAt: "Jul 20, 2026" },
  { id: "6", title: "The hidden psychology behind shareable content", thumbnail: "https://images.unsplash.com/photo-1531058020387-3be344556be6?auto=format&fit=crop&w=1200&q=80", platform: "YouTube", status: "ready", score: 82, source: "The Marketing Room", duration: "00:48", createdAt: "Jul 19, 2026" },
  { id: "7", title: "Build systems that let your creativity scale", thumbnail: "https://images.unsplash.com/photo-1556761175-b413da4baf72?auto=format&fit=crop&w=1200&q=80", platform: "TikTok", status: "processing", score: 80, source: "Creator Ops", duration: "00:45", createdAt: "Jul 18, 2026" },
  { id: "8", title: "Turn a blank page into a compelling narrative", thumbnail: "https://images.unsplash.com/photo-1505373877841-8d25f7d46678?auto=format&fit=crop&w=1200&q=80", platform: "Instagram", status: "ready", score: 78, source: "Writing Out Loud", duration: "01:03", createdAt: "Jul 17, 2026" },
];

const statusStyles: Record<VideoStatus, string> = {
  ready: "border-emerald-200 bg-emerald-50 text-emerald-700",
  processing: "border-amber-200 bg-amber-50 text-amber-700",
  failed: "border-rose-200 bg-rose-50 text-rose-700",
};

export default function ClipsPage() {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<"all" | VideoStatus>("all");
  const [page, setPage] = useState(1);

  const filteredClips = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return clips.filter(
      (clip) =>
        (status === "all" || clip.status === status) &&
        (!keyword ||
          clip.title.toLowerCase().includes(keyword) ||
          clip.source.toLowerCase().includes(keyword) ||
          clip.platform.toLowerCase().includes(keyword)),
    );
  }, [search, status]);

  const pageCount = Math.max(1, Math.ceil(filteredClips.length / PAGE_SIZE));
  const currentPage = Math.min(page, pageCount);
  const visibleClips = filteredClips.slice(
    (currentPage - 1) * PAGE_SIZE,
    currentPage * PAGE_SIZE,
  );

  function updateSearch(value: string) {
    setSearch(value);
    setPage(1);
  }

  function updateStatus(value: "all" | VideoStatus) {
    setStatus(value);
    setPage(1);
  }

  return (
    <div className="space-y-7">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <PageHeading
          eyebrow="Content library"
          title="Your clips"
          description="Review, manage, and publish the best moments found across your source videos."
        />
        <div className="flex flex-col gap-2 sm:flex-row">
          <Button variant="outline" className="h-11 rounded-xl bg-white px-4">
            <SparkleIcon />
            Search Trending
          </Button>
          <Button className="h-11 rounded-xl px-4">
            <LinkSimpleIcon />
            Clip By Link
          </Button>
        </div>
      </div>

      <Card className="gap-3 border-black/8 bg-white p-4 shadow-sm flex flex-col sm:flex-row sm:items-center">
        <div className="relative min-w-0 flex-1">
          <MagnifyingGlassIcon className="absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(event) => updateSearch(event.target.value)}
            className="h-11 rounded-xl pl-10"
            placeholder="Search clips or source videos..."
            aria-label="Search clips"
          />
        </div>
        <Select
          value={status}
          onValueChange={(value) => updateStatus(value as "all" | VideoStatus)}
        >
          <SelectTrigger className="sm:w-48" aria-label="Filter by status">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="ready">Ready</SelectItem>
            <SelectItem value="processing">Processing</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>
      </Card>

      {visibleClips.length ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {visibleClips.map((clip) => (
            <ClipLibraryCard key={clip.id} clip={clip} />
          ))}
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-black/15 bg-white/60 py-16 text-center">
          <FilmStripIcon className="mx-auto size-8 text-violet-400" />
          <h2 className="mt-3 font-heading text-lg font-semibold">
            No clips found
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">
            Try another search or status filter.
          </p>
        </div>
      )}

      {filteredClips.length > 0 && (
        <div className="flex flex-col gap-3 border-t border-black/8 pt-5 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-center text-xs text-muted-foreground sm:text-left">
            Page {currentPage} of {pageCount}
          </p>
          <Pagination className="mx-0 w-auto">
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  disabled={currentPage === 1}
                  onClick={() => setPage((value) => Math.max(1, value - 1))}
                />
              </PaginationItem>
              {Array.from({ length: pageCount }, (_, index) => index + 1).map(
                (pageNumber) => (
                  <PaginationItem key={pageNumber}>
                    <PaginationLink
                      isActive={pageNumber === currentPage}
                      aria-label={`Go to page ${pageNumber}`}
                      onClick={() => setPage(pageNumber)}
                    >
                      {pageNumber}
                    </PaginationLink>
                  </PaginationItem>
                ),
              )}
              <PaginationItem>
                <PaginationNext
                  disabled={currentPage === pageCount}
                  onClick={() =>
                    setPage((value) => Math.min(pageCount, value + 1))
                  }
                />
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      )}
    </div>
  );
}

function ClipLibraryCard({ clip }: { clip: LibraryClip }) {
  return (
    <Card className="group gap-0 overflow-hidden border-black/8 bg-white py-0 transition duration-300 hover:-translate-y-1 hover:shadow-[0_20px_50px_-24px_rgba(31,18,52,0.35)]">
      <div className="relative aspect-video overflow-hidden bg-violet-50">
        <Image
          src={clip.thumbnail}
          alt=""
          fill
          unoptimized
          sizes="(max-width: 768px) 100vw, (max-width: 1280px) 50vw, 33vw"
          className="object-cover transition duration-500 group-hover:scale-[1.04]"
        />
        <div className="absolute inset-0 bg-linear-to-t from-black/50 via-transparent to-transparent" />
        <span className="absolute bottom-3 right-3 rounded-md bg-black/65 px-2 py-1 text-xs font-medium text-white">
          {clip.duration}
        </span>
      </div>
      <div className="p-4">
        <div className="flex items-start justify-between gap-3">
          <h3 className="line-clamp-2 min-h-12 font-heading text-base font-semibold leading-6 tracking-[-0.02em]">
            {clip.title}
          </h3>
          <Badge className="border-white/20 bg-black/55 text-white backdrop-blur-md">
            {clip.platform}
          </Badge>
        </div>
        <p className="mt-1 truncate text-xs text-muted-foreground">
          From <span className="text-primary">{clip.source}</span>
        </p>
        <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <SparkleIcon className="size-4 text-violet-600" />
            {clip.score}% highlight score
          </span>
          <span className="flex items-center gap-1.5">
            <CalendarBlankIcon className="size-4" />
            {clip.createdAt}
          </span>
        </div>
        <Button variant="outline" className="mt-4 w-full rounded-xl">
          <PlayIcon />
          Preview clip
          <ArrowRightIcon className="ml-auto" />
        </Button>
      </div>
    </Card>
  );
}
