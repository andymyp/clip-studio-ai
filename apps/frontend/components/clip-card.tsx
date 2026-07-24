import { Clock, Sparkle } from "@phosphor-icons/react/dist/ssr";

import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import type { Clip } from "@/lib/types";

const statusStyles: Record<Clip["status"], string> = {
  candidate: "border-amber-200 bg-amber-50 text-amber-700",
  approved: "border-emerald-200 bg-emerald-50 text-emerald-700",
  rejected: "border-rose-200 bg-rose-50 text-rose-700",
  rendered: "border-violet-200 bg-violet-50 text-violet-700",
};

function formatTime(seconds: number) {
  const minutes = Math.floor(seconds / 60);
  return `${minutes}:${Math.floor(seconds % 60).toString().padStart(2, "0")}`;
}

export function ClipCard({ clip }: { clip: Clip }) {
  return (
    <Card className="flex items-center gap-4 border-black/8 p-4 shadow-none">
      <div className="grid size-14 shrink-0 place-items-center rounded-2xl bg-violet-100 text-violet-700">
        <Sparkle weight="fill" className="size-6" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-3">
          <h3 className="truncate font-heading font-semibold">{clip.title}</h3>
          <Badge className={statusStyles[clip.status]}>{clip.status}</Badge>
        </div>
        <div className="mt-2 flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <Clock className="size-3.5" />
            {formatTime(clip.startTime)}–{formatTime(clip.endTime)}
          </span>
          <span>{Math.round(clip.score)}% highlight score</span>
        </div>
      </div>
    </Card>
  );
}
