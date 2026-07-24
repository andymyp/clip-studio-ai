"use client";

import {
  ArrowRightIcon as ArrowRight,
  CircleNotchIcon as CircleNotch,
  LockKeyIcon as LockKey,
} from "@phosphor-icons/react";
import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useRouter } from "@bprogress/next/app";
import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useAuthMutation } from "@/hooks/mutations/use-auth-mutation";
import { createAuthSchema, type AuthValues } from "@/lib/validations";
import { useAuthStore } from "@/stores/auth-store";

export function AuthForm({ mode }: { mode: "signin" | "signup" }) {
  const router = useRouter();
  const { accessToken, hydrated, setSession } = useAuthStore();
  const signingUp = mode === "signup";
  const auth = useAuthMutation(mode);
  const form = useForm<AuthValues>({
    resolver: zodResolver(createAuthSchema(signingUp)),
    defaultValues: { email: "", password: "", confirmPassword: "" },
  });

  useEffect(() => {
    if (hydrated && accessToken) {
      router.replace("/dashboard");
    }
  }, [accessToken, hydrated, router]);

  async function submit(values: AuthValues) {
    try {
      const { user, tokens } = await auth.mutateAsync(values);
      setSession(user, tokens.access_token, tokens.refresh_token);
      router.replace("/dashboard");
    } catch {
      // Global MutationCache displays the normalized API error with Sonner.
    }
  }

  const submitting = form.formState.isSubmitting || auth.isPending;

  return (
    <Form {...form}>
      <form className="mt-8" onSubmit={form.handleSubmit(submit)} noValidate>
        <fieldset disabled={submitting} className="space-y-4">
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Email address</FormLabel>
                <FormControl>
                  <Input
                    type="email"
                    autoComplete="email"
                    placeholder="you@studio.com"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="password"
            render={({ field }) => (
              <FormItem>
                <div className="flex items-center justify-between gap-4 mb-0.5">
                  <FormLabel>Password</FormLabel>
                  <span className="text-xs text-muted-foreground">8-72 characters</span>
                </div>
                <div className="relative">
                  <LockKey className="pointer-events-none absolute left-3.5 top-3.5 z-10 size-4 text-muted-foreground" />
                  <FormControl>
                    <Input
                      className="pl-10"
                      type="password"
                      autoComplete={signingUp ? "new-password" : "current-password"}
                      placeholder="••••••••••••"
                      {...field}
                    />
                  </FormControl>
                </div>
                <FormMessage />
              </FormItem>
            )}
          />

          {signingUp && (
            <FormField
              control={form.control}
              name="confirmPassword"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Confirm password</FormLabel>
                  <div className="relative">
                    <LockKey className="pointer-events-none absolute left-3.5 top-3.5 z-10 size-4 text-muted-foreground" />
                    <FormControl>
                      <Input
                        className="pl-10"
                        type="password"
                        autoComplete="new-password"
                        placeholder="••••••••••••"
                        {...field}
                      />
                    </FormControl>
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          <Button className="h-11 w-full rounded-xl text-sm" disabled={submitting}>
            {submitting ? (
              <CircleNotch className="animate-spin" />
            ) : (
              <ArrowRight weight="bold" />
            )}
            {submitting
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
        </fieldset>
      </form>
    </Form>
  );
}
