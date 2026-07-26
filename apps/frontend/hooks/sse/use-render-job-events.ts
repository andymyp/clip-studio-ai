"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import { useAuthStore } from "@/stores/auth-store";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001";

export function useRenderJobEvents() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!accessToken) return;
    const controller = new AbortController();

    async function connect() {
      let attempt = 0;
      while (!controller.signal.aborted) {
        try {
          const response = await fetch(`${apiURL}/renders/events`, {
            headers: {
              Accept: "text/event-stream",
              Authorization: `Bearer ${accessToken}`,
            },
            cache: "no-store",
            signal: controller.signal,
          });
          if (!response.ok || !response.body) {
            throw new Error(`Render event stream returned ${response.status}`);
          }
          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let buffer = "";
          attempt = 0;
          while (!controller.signal.aborted) {
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
                const jobs = JSON.parse(payload) as unknown;
                if (Array.isArray(jobs)) {
                  await Promise.all([
                    queryClient.invalidateQueries({ queryKey: ["render-jobs"] }),
                    queryClient.invalidateQueries({ queryKey: ["rendered-clips"] }),
                  ]);
                }
              }
              boundary = buffer.indexOf("\n\n");
            }
          }
        } catch {
          if (controller.signal.aborted) return;
        }
        attempt += 1;
        await new Promise((resolve) =>
          setTimeout(resolve, Math.min(attempt, 5) * 1_000),
        );
      }
    }

    void connect();
    return () => controller.abort();
  }, [accessToken, queryClient]);
}
