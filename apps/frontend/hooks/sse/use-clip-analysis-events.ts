"use client";

import { useEffect, useState } from "react";

import { useAuthStore } from "@/stores/auth-store";
import type { ClipAnalysisJob } from "@/lib/types";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001";

type AnalysisEventsState = {
  data?: ClipAnalysisJob;
  error?: Error;
  isLoading: boolean;
  isError: boolean;
};

export function useClipAnalysisEvents(externalID: string): AnalysisEventsState {
  const accessToken = useAuthStore((state) => state.accessToken);
  const [data, setData] = useState<ClipAnalysisJob>();
  const [error, setError] = useState<Error>();

  useEffect(() => {
    if (!externalID || !accessToken) return;

    const controller = new AbortController();

    async function connect() {
      let attempts = 0;
      while (!controller.signal.aborted) {
        let terminal = false;
        try {
        const response = await fetch(
          `${apiURL}/clips/reviews/${encodeURIComponent(externalID)}/events`,
          {
            headers: {
              Accept: "text/event-stream",
              Authorization: `Bearer ${accessToken}`,
            },
            cache: "no-store",
            signal: controller.signal,
          },
        );
        if (!response.ok) {
          const body = (await response.json().catch(() => null)) as {
            error?: string;
          } | null;
          throw new Error(body?.error ?? `Event stream returned ${response.status}`);
        }
        if (!response.body) throw new Error("Event stream is unavailable");

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, "\n");

          let boundary = buffer.indexOf("\n\n");
          while (boundary !== -1) {
            const frame = buffer.slice(0, boundary);
            buffer = buffer.slice(boundary + 2);
            const payload = frame
              .split("\n")
              .filter((line) => line.startsWith("data:"))
              .map((line) => line.slice(5).trimStart())
              .join("\n");
            if (payload) {
              const next = JSON.parse(payload) as ClipAnalysisJob;
              terminal = next.status === "completed" || next.status === "failed";
              setData(next);
              setError(undefined);
            }
            boundary = buffer.indexOf("\n\n");
          }
        }
          if (terminal || controller.signal.aborted) return;
          throw new Error("Clip event stream closed before the job finished");
        } catch (streamError) {
          if (controller.signal.aborted) return;
          attempts += 1;
          if (attempts < 11) {
            await new Promise((resolve) =>
              setTimeout(resolve, Math.min(attempts, 3) * 1_000),
            );
            continue;
          }
          setError(
            streamError instanceof Error
              ? streamError
              : new Error("Clip event stream failed"),
          );
          return;
        }
      }
    }

    void connect();
    return () => controller.abort();
  }, [accessToken, externalID]);

  return {
    data,
    error,
    isLoading: !data && !error,
    isError: Boolean(error),
  };
}
