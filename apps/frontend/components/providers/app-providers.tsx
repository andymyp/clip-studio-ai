"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { ProgressProvider } from "@bprogress/next/app";
import type { ReactNode } from "react";
import { useState } from "react";

export function AppProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            refetchOnWindowFocus: false,
          },
        },
      }),
  );

  return (
    <ProgressProvider
      height="3px"
      color="#7c3aed"
      options={{ showSpinner: false }}
      shallowRouting
    >
      <QueryClientProvider client={queryClient}>
        {children}
        {process.env.NODE_ENV === "development" && <ReactQueryDevtools />}
      </QueryClientProvider>
    </ProgressProvider>
  );
}
