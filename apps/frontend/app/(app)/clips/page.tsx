"use client";

import {
  CircleNotchIcon,
  DownloadSimpleIcon,
  FilmStripIcon,
  LinkSimpleIcon,
  MagnifyingGlassIcon,
  PlayIcon,
  SparkleIcon,
} from "@phosphor-icons/react";
import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

import { PageHeading } from "@/components/page-heading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { ResponsiveModal } from "@/components/ui/responsive-modal";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useRenderedClips } from "@/hooks/queries/use-render-jobs";
import { useSubmitPerformance } from "@/hooks/mutations/use-submit-performance";
import { apiErrorMessage } from "@/lib/api";
import type { RenderJob } from "@/lib/types";
import { clipByLinkSchema, performanceFeedbackSchema, type ClipByLinkValues, type PerformanceFeedbackValues } from "@/lib/validations";

const workerURL = process.env.NEXT_PUBLIC_WORKER_URL ?? "http://localhost:3002";

export default function ClipsPage() {
  const router = useRouter();
  const clips = useRenderedClips();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("newest");
  const [linkOpen, setLinkOpen] = useState(false);
  const [preview, setPreview] = useState<RenderJob | null>(null);
  const form = useForm<ClipByLinkValues>({ resolver: zodResolver(clipByLinkSchema), defaultValues: { url: "" } });
  const rows = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return (clips.data ?? [])
      .filter((clip) => !keyword || `${clip.title} ${clip.video_title} ${clip.description}`.toLowerCase().includes(keyword))
      .toSorted((a, b) => sort === "oldest"
        ? +new Date(a.created_at) - +new Date(b.created_at)
        : sort === "score" ? b.score - a.score : +new Date(b.created_at) - +new Date(a.created_at));
  }, [clips.data, search, sort]);

  function mediaURL(path: string) {
    return path.startsWith("http") ? path : `${workerURL}${path}`;
  }

  async function submitLink(values: ClipByLinkValues) {
    setLinkOpen(false);
    router.push(`/clips/trending-videos?url=${encodeURIComponent(values.url)}`);
  }

  return (
    <div className="space-y-7">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <PageHeading eyebrow="Content library" title="Your clips" description="Completed vertical clips, ready to preview and publish." />
        <div className="flex gap-2">
          <Button asChild variant="outline"><Link href="/clips/trending-videos"><SparkleIcon />Search Trending</Link></Button>
          <Button onClick={() => setLinkOpen(true)}><LinkSimpleIcon />Clip By Link</Button>
        </div>
      </div>
      <Card className="flex flex-col gap-3 border-black/8 bg-white p-4 sm:flex-row">
        <div className="relative flex-1">
          <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input className="pl-9" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search rendered clips..." />
        </div>
        <Select value={sort} onValueChange={setSort}>
          <SelectTrigger className="sm:w-44"><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="newest">Newest first</SelectItem><SelectItem value="oldest">Oldest first</SelectItem><SelectItem value="score">Highest score</SelectItem></SelectContent>
        </Select>
      </Card>
      {rows.length ? (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {rows.map((clip) => (
            <Card key={clip.id} className="gap-0 overflow-hidden border-black/8 bg-white p-0">
              <div className="aspect-video bg-black"><video src={mediaURL(clip.media_url)} className="size-full object-cover" preload="metadata" /></div>
              <div className="space-y-3 p-4">
                <div className="flex items-start justify-between gap-2"><h2 className="line-clamp-2 font-semibold">{clip.title || clip.video_title}</h2><Badge>{clip.score}%</Badge></div>
                <p className="line-clamp-2 text-xs text-muted-foreground">{clip.description}</p>
                <Button variant="outline" className="w-full" onClick={() => setPreview(clip)}><PlayIcon />Preview clip</Button>
              </div>
            </Card>
          ))}
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed py-20 text-center"><FilmStripIcon className="mx-auto size-9 text-violet-400" /><p className="mt-3 font-semibold">{clips.isLoading ? "Loading clips..." : "No rendered clips yet"}</p></div>
      )}
      <ResponsiveModal open={linkOpen} onOpenChange={setLinkOpen} title="Clip by link" description="Paste a public YouTube link.">
        <Form {...form}><form className="space-y-5" onSubmit={form.handleSubmit(submitLink)}>
          <fieldset disabled={form.formState.isSubmitting}><FormField control={form.control} name="url" render={({ field }) => <FormItem><FormLabel>Video link</FormLabel><FormControl><Input type="url" {...field} /></FormControl><FormMessage /></FormItem>} /></fieldset>
          <Button className="w-full" disabled={form.formState.isSubmitting}>{form.formState.isSubmitting ? <CircleNotchIcon className="animate-spin" /> : <SparkleIcon />}Clip</Button>
        </form></Form>
      </ResponsiveModal>
      <ResponsiveModal open={Boolean(preview)} onOpenChange={(open) => !open && setPreview(null)} title="Rendered clip" className="overflow-y-auto sm:max-w-4xl">
        {preview && <div className="grid gap-6 md:grid-cols-2">
          <video controls autoPlay src={mediaURL(preview.media_url)} className="mx-auto max-h-[70dvh] rounded-2xl bg-black" />
          <div className="space-y-4">
            <Badge>Best potential cut</Badge>
            <h2 className="text-xl font-semibold">{preview.title}</h2>
            {preview.hook && <div className="rounded-xl bg-violet-50 p-3 text-sm"><strong>Opening hook:</strong> {preview.hook}</div>}
            <p className="text-sm leading-6">{preview.description}</p>
            <div className="flex gap-2 text-xs text-muted-foreground"><span>{preview.removed_seconds}s dead air removed</span><span>·</span><span>{preview.pattern_interrupts} visual beats</span></div>
            <div className="flex flex-wrap gap-2">{preview.hashtags.map((tag) => <Badge key={tag}>#{tag.replace(/^#/, "")}</Badge>)}</div>
            <Button asChild><a href={mediaURL(preview.media_url)} download><DownloadSimpleIcon />Download MP4</a></Button>
            <PerformanceForm renderID={preview.id} />
          </div>
        </div>}
      </ResponsiveModal>
    </div>
  );
}

function PerformanceForm({ renderID }: { renderID: string }) {
  const submit = useSubmitPerformance(renderID);
  const form = useForm<PerformanceFeedbackValues>({
    resolver: zodResolver(performanceFeedbackSchema),
    defaultValues: {
      platform: "youtube", views: 0, likes: 0, comments: 0, shares: 0,
      average_watch_seconds: 0, completion_percentage: 0,
    },
  });
  async function onSubmit(values: PerformanceFeedbackValues) {
    try {
      const result = await submit.mutateAsync(values);
      toast.success(`Performance saved · viral score ${Math.round(result.viral_score)}`);
      form.reset();
    } catch (error) {
      toast.error(apiErrorMessage(error));
    }
  }
  return (
    <details className="border-t pt-4">
      <summary className="cursor-pointer text-sm font-semibold">Add published performance</summary>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="mt-4 space-y-3">
          <fieldset disabled={submit.isPending} className="grid grid-cols-2 gap-3">
            <FormField control={form.control} name="platform" render={({ field }) => <FormItem className="col-span-2"><FormLabel>Platform</FormLabel><Select value={field.value} onValueChange={field.onChange}><FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl><SelectContent><SelectItem value="youtube">YouTube</SelectItem><SelectItem value="tiktok">TikTok</SelectItem><SelectItem value="instagram">Instagram</SelectItem></SelectContent></Select><FormMessage /></FormItem>} />
            {(["views", "likes", "comments", "shares", "average_watch_seconds", "completion_percentage"] as const).map((name) => (
              <FormField key={name} control={form.control} name={name} render={({ field }) => <FormItem><FormLabel>{name.replaceAll("_", " ")}</FormLabel><FormControl><Input type="number" min={0} max={name === "completion_percentage" ? 100 : undefined} name={field.name} ref={field.ref} onBlur={field.onBlur} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} /></FormControl><FormMessage /></FormItem>} />
            ))}
          </fieldset>
          <Button type="submit" className="w-full" disabled={submit.isPending}>{submit.isPending && <CircleNotchIcon className="animate-spin" />}Save performance</Button>
        </form>
      </Form>
    </details>
  );
}
