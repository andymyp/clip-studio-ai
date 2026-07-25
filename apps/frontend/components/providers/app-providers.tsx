"use client";

import { useProgress } from "@bprogress/next";
import { ProgressProvider } from "@bprogress/next/app";
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import type { ReactNode } from "react";
import { useState } from "react";
import { toast } from "sonner";

import { Toaster } from "@/components/ui/sonner";
import { apiErrorMessage } from "@/lib/api";

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <ProgressProvider
      height="4px"
      color="#7c3aed"
      options={{ showSpinner: false }}
      shallowRouting
    >
      <QueryProviders>{children}</QueryProviders>
    </ProgressProvider>
  );
}

function QueryProviders({ children }: { children: ReactNode }) {
  const progress = useProgress();
  const [queryClient] = useState(
    () => {
      let activeMutations = 0;
      return new QueryClient({
        queryCache: new QueryCache({
          onError: (error) => toast.error(apiErrorMessage(error)),
        }),
        mutationCache: new MutationCache({
          onMutate: () => {
            activeMutations += 1;
            progress.start();
          },
          onError: (error) => toast.error(apiErrorMessage(error)),
          onSettled: () => {
            activeMutations = Math.max(0, activeMutations - 1);
            if (activeMutations === 0) progress.stop();
          },
        }),
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            refetchOnWindowFocus: false,
            retry: 1,
          },
          mutations: {
            retry: 0,
          },
        },
      });
    },
  );

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <Toaster position="bottom-right" />
      {process.env.NODE_ENV === "development" && <ReactQueryDevtools />}
    </QueryClientProvider>
  );
}
