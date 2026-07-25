"use client";

import {
  ArrowRightIcon,
  CalendarBlankIcon,
  CircleNotchIcon,
  FilmStripIcon,
  LinkSimpleIcon,
  MagnifyingGlassIcon,
  PlayIcon,
  SparkleIcon,
} from "@phosphor-icons/react";
import { zodResolver } from "@hookform/resolvers/zod";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

import { PageHeading } from "@/components/page-heading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
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
import { ResponsiveModal } from "@/components/ui/responsive-modal";
import {
  clipByLinkSchema,
  type ClipByLinkValues,
} from "@/lib/validations";

const PAGE_SIZE = 6;

type VideoStatus = "ready" | "processing" | "failed";
type ClipSort = "newest" | "oldest" | "highest-score" | "shortest" | "longest";

type LibraryClip = {
  id: string;
  title: string;
  thumbnail: string;
  platform: "YouTube" | "TikTok" | "Instagram";
  status: VideoStatus;
  score: number;
  source: string;
  description?: string;
  hashtags?: string[];
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

function durationInSeconds(duration: string) {
  const [minutes, seconds] = duration.split(":").map(Number);
  return minutes * 60 + seconds;
}

export default function ClipsPage() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<ClipSort>("newest");
  const [page, setPage] = useState(1);
  const [linkModalOpen, setLinkModalOpen] = useState(false);
  const [previewClip, setPreviewClip] = useState<LibraryClip | null>(null);
  const linkForm = useForm<ClipByLinkValues>({
    resolver: zodResolver(clipByLinkSchema),
    defaultValues: { url: "" },
  });

  const filteredClips = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    const matches = clips.filter(
      (clip) =>
        !keyword ||
          clip.title.toLowerCase().includes(keyword) ||
          clip.source.toLowerCase().includes(keyword) ||
          clip.platform.toLowerCase().includes(keyword),
    );

    return matches.toSorted((a, b) => {
      switch (sort) {
        case "oldest":
          return Number(b.id) - Number(a.id);
        case "highest-score":
          return b.score - a.score;
        case "shortest":
          return durationInSeconds(a.duration) - durationInSeconds(b.duration);
        case "longest":
          return durationInSeconds(b.duration) - durationInSeconds(a.duration);
        default:
          return Number(a.id) - Number(b.id);
      }
    });
  }, [search, sort]);

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

  function updateSort(value: ClipSort) {
    setSort(value);
    setPage(1);
  }

  async function submitLink(values: ClipByLinkValues) {
    linkForm.reset();
    setLinkModalOpen(false);
    router.push(`/clips/trending-videos?url=${encodeURIComponent(values.url)}`);
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
          <Button asChild variant="outline" className="h-11 rounded-xl bg-white px-4">
            <Link href="/clips/trending-videos">
              <SparkleIcon />
              Search Trending
            </Link>
          </Button>
          <Button
            className="h-11 rounded-xl px-4"
            onClick={() => setLinkModalOpen(true)}
          >
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
        <Select value={sort} onValueChange={(value) => updateSort(value as ClipSort)}>
          <SelectTrigger className="sm:w-48" aria-label="Sort clips">
            <SelectValue placeholder="Sort by" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="newest">Newest first</SelectItem>
            <SelectItem value="oldest">Oldest first</SelectItem>
            <SelectItem value="highest-score">Highest score</SelectItem>
            <SelectItem value="shortest">Shortest first</SelectItem>
            <SelectItem value="longest">Longest first</SelectItem>
          </SelectContent>
        </Select>
      </Card>

      {visibleClips.length ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {visibleClips.map((clip) => (
            <ClipLibraryCard
              key={clip.id}
              clip={clip}
              onPreview={() => setPreviewClip(clip)}
            />
          ))}
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-black/15 bg-white/60 py-16 text-center">
          <FilmStripIcon className="mx-auto size-8 text-violet-400" />
          <h2 className="mt-3 font-heading text-lg font-semibold">
            No clips found
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">
            Try another search.
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

      <ResponsiveModal
        open={linkModalOpen}
        onOpenChange={setLinkModalOpen}
        title="Clip by link"
        description="Paste a public video link and ClipStudio AI will find its best moments."
      >
        <Form {...linkForm}>
          <form
            onSubmit={linkForm.handleSubmit(submitLink)}
            className="space-y-5"
            noValidate
          >
            <fieldset disabled={linkForm.formState.isSubmitting}>
              <FormField
                control={linkForm.control}
                name="url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Video link</FormLabel>
                    <FormControl>
                      <Input
                        type="url"
                        placeholder="https://youtube.com/watch?v=... or reddit.com/..."
                        autoComplete="url"
                        autoFocus
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      YouTube and Reddit video links are supported.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </fieldset>
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={() => setLinkModalOpen(false)}
                disabled={linkForm.formState.isSubmitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={linkForm.formState.isSubmitting}>
                {linkForm.formState.isSubmitting ? (
                  <CircleNotchIcon className="animate-spin" />
                ) : (
                  <SparkleIcon />
                )}
                {linkForm.formState.isSubmitting ? "Clipping" : "Clip"}
              </Button>
            </div>
          </form>
        </Form>
      </ResponsiveModal>

      <ResponsiveModal
        open={Boolean(previewClip)}
        onOpenChange={(open) => !open && setPreviewClip(null)}
        title="Clip preview"
        className="overflow-y-auto sm:max-w-4xl"
      >
        {previewClip && <ClipPreview clip={previewClip} />}
      </ResponsiveModal>
    </div>
  );
}

function ClipLibraryCard({
  clip,
  onPreview,
}: {
  clip: LibraryClip;
  onPreview: () => void;
}) {
  return (
    <Card className="group gap-0 overflow-hidden border-black/8 bg-white py-0 transition duration-300 hover:-translate-y-1 hover:shadow-[0_20px_50px_-24px_rgba(31,18,52,0.35)]">
      <div className="relative aspect-video overflow-hidden bg-violet-50">
        <Image
          src={clip.thumbnail}
          alt=""
          fill
          unoptimized
          sizes="(max-width: 640px) 100vw, (max-width: 1280px) 50vw, 33vw"
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
        <Button
          variant="outline"
          className="mt-4 w-full rounded-xl"
          onClick={onPreview}
        >
          <PlayIcon />
          Preview clip
          <ArrowRightIcon className="ml-auto" />
        </Button>
      </div>
    </Card>
  );
}

function ClipPreview({ clip }: { clip: LibraryClip }) {
  const description =
    clip.description ??
    `A high-potential moment selected from ${clip.source}, optimized for a concise short-form story.`;
  const hashtags = clip.hashtags ?? [
    clip.platform.toLowerCase(),
    "shortvideo",
    "clipstudio",
  ];

  return (
    <div className="grid gap-6 md:grid-cols-[minmax(240px,0.8fr)_minmax(0,1.2fr)]">
      <div className="md:sticky md:top-0 md:self-start">
        <div className="relative mx-auto aspect-9/16 h-[min(62dvh,36rem)] w-auto max-w-full overflow-hidden rounded-2xl bg-black">
          <Image
            src={clip.thumbnail}
            alt=""
            fill
            unoptimized
            sizes="(max-width: 768px) 80vw, 320px"
            className="object-cover opacity-80"
          />
          <div className="absolute inset-0 grid place-items-center bg-black/15">
            <button
              type="button"
              className="grid size-16 place-items-center rounded-full bg-white text-primary shadow-xl transition hover:scale-105"
              aria-label="Play clip preview"
            >
              <PlayIcon weight="fill" className="size-7 translate-x-0.5" />
            </button>
          </div>
          <span className="absolute bottom-3 right-3 rounded-md bg-black/70 px-2 py-1 text-xs font-medium text-white">
            {clip.duration}
          </span>
        </div>
      </div>

      <div className="min-w-0 space-y-5">
        <div className="flex flex-wrap items-center gap-2">
          <Badge className={statusStyles[clip.status]}>{clip.status}</Badge>
          <Badge className="border-violet-200 bg-violet-50 text-violet-700">
            {clip.score}% highlight score
          </Badge>
        </div>
        <h3 className="mt-3 font-heading text-xl font-semibold leading-7">
          {clip.title}
        </h3>
        <div className="mt-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Source
          </p>
          <p className="mt-1 text-sm font-medium">{clip.source}</p>
        </div>
        <div className="border-t pt-4">
          <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Description
          </p>
          <p className="mt-2 text-sm leading-6 text-foreground/80">
            {description}
          </p>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Hashtags
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            {hashtags.map((hashtag) => (
              <Badge
                key={hashtag}
                className="border-violet-200 bg-violet-50 text-violet-700 normal-case tracking-normal"
              >
                #{hashtag}
              </Badge>
            ))}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3 border-y py-4 text-sm">
          <div>
            <p className="text-muted-foreground">Platform</p>
            <p className="mt-1 font-medium">{clip.platform}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Created</p>
            <p className="mt-1 font-medium">{clip.createdAt}</p>
          </div>
        </div>
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button variant="outline" onClick={() => setPreviewActionToast("download")}>
            Download
          </Button>
          <Button onClick={() => setPreviewActionToast("editor")}>
            Open in editor
            <ArrowRightIcon />
          </Button>
        </div>
      </div>
    </div>
  );
}

function setPreviewActionToast(action: "download" | "editor") {
  toast.info(
    action === "download"
      ? "Download will be available when rendering is connected."
      : "The clip editor will be connected in a later step.",
  );
}
