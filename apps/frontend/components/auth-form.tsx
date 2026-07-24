"use client";

import { ArrowRight, LockKey, Sparkle } from "@phosphor-icons/react";
import { useMutation } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "@bprogress/next/app";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { api, apiErrorMessage } from "@/lib/api";
import type { AuthResponse } from "@/lib/types";
import { useActivityStore } from "@/stores/activity-store";
import { useAuthStore } from "@/stores/auth-store";

export function AuthForm({ mode }: { mode: "signin" | "signup" }) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { accessToken, hydrated, setSession } = useAuthStore();
  const addLog = useActivityStore((state) => state.addLog);
  const signingUp = mode === "signup";

  useEffect(() => {
    if (hydrated && accessToken) {
      router.replace("/dashboard");
    }
  }, [accessToken, hydrated, router]);

  const auth = useMutation({
    mutationFn: async () => {
      const endpoint = signingUp ? "/auth/register" : "/auth/login";
      const { data } = await api.post<AuthResponse>(endpoint, { email, password });
      return data;
    },
    onSuccess: ({ user, tokens }) => {
      setSession(user, tokens.access_token, tokens.refresh_token);
      addLog({
        action: signingUp ? "Account created" : "Signed in",
        detail: user.email,
        status: "success",
      });
      router.replace("/dashboard");
    },
  });

  return (
    <form
      className="mt-8 space-y-5"
      onSubmit={(event) => {
        event.preventDefault();
        auth.mutate();
      }}
    >
      <div className="space-y-2">
        <label htmlFor="email" className="text-sm font-semibold">
          Email address
        </label>
        <Input
          id="email"
          type="email"
          autoComplete="email"
          placeholder="you@studio.com"
          required
          maxLength={320}
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
      </div>
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <label htmlFor="password" className="text-sm font-semibold">
            Password
          </label>
          <span className="text-xs text-muted-foreground">8–72 characters</span>
        </div>
        <div className="relative">
          <LockKey className="pointer-events-none absolute left-3.5 top-3.5 size-4 text-muted-foreground" />
          <Input
            id="password"
            className="pl-10"
            type="password"
            autoComplete={signingUp ? "new-password" : "current-password"}
            placeholder="••••••••••••"
            required
            minLength={8}
            maxLength={72}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </div>
      </div>

      {auth.isError && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 px-3.5 py-3 text-sm text-rose-700">
          {apiErrorMessage(auth.error)}
        </div>
      )}

      <Button className="h-11 w-full rounded-xl text-sm" disabled={auth.isPending}>
        {auth.isPending ? (
          <Sparkle className="animate-pulse" weight="fill" />
        ) : (
          <ArrowRight weight="bold" />
        )}
        {auth.isPending
          ? signingUp
            ? "Creating workspace"
            : "Signing in"
          : signingUp
            ? "Create account"
            : "Sign in"}
      </Button>

      <p className="text-center text-sm text-muted-foreground">
        {signingUp ? "Already have an account?" : "New to ClipStudio?"}{" "}
        <Link
          href={signingUp ? "/signin" : "/signup"}
          className="font-semibold text-violet-700 hover:text-violet-900"
        >
          {signingUp ? "Sign in" : "Create one"}
        </Link>
      </p>
    </form>
  );
}
