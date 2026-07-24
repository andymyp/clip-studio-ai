"use client";

import { useAuthStore } from "@/stores/auth-store";
import {
  CheckCircleIcon as CheckCircle,
  ScissorsIcon as Scissors,
  SparkleIcon as Sparkle,
} from "@phosphor-icons/react/dist/ssr";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

export function AuthLayout({
  eyebrow,
  title,
  description,
  children,
}: {
  eyebrow: string;
  title: string;
  description: string;
  children: ReactNode;
}) {
  const router = useRouter();
  const { user, accessToken, hydrated, clearSession } = useAuthStore();

  useEffect(() => {
    if (hydrated && accessToken) {
      router.replace("/dashboard");
    }
  }, [accessToken, hydrated, router]);

  if (!hydrated || accessToken) {
    return (
      <div className="grid min-h-screen place-items-center bg-[#f7f5f2]">
        <img src="./logo.png" className="h-14 animate-pulse" />
      </div>
    );
  }

  return (
    <main className="grid min-h-screen bg-[#f7f5f2] lg:grid-cols-[1.05fr_0.95fr]">
      <section className="relative hidden overflow-hidden bg-[#20172b] p-12 text-white lg:flex lg:flex-col">
        <div className="absolute -right-36 -top-32 size-96 rounded-full bg-violet-600/30 blur-3xl" />
        <div className="absolute -bottom-36 -left-24 size-96 rounded-full bg-fuchsia-500/15 blur-3xl" />
        <Link href="/" className="relative flex items-center gap-3">
          <Image src="/logo_square.png" alt="" width={42} height={42} className="rounded-xl" />
          <span className="font-heading text-xl font-bold tracking-[-0.03em]">ClipStudio AI</span>
        </Link>
        <div className="relative my-auto max-w-xl">
          <div className="mb-7 flex items-center gap-3 text-violet-300">
            <Scissors className="size-7" />
            <span className="text-xs font-bold uppercase tracking-[0.22em]">
              Intelligent clipping
            </span>
          </div>
          <h2 className="font-heading text-5xl font-bold leading-[1.05] tracking-[-0.055em]">
            Turn long conversations into moments worth sharing.
          </h2>
          <p className="mt-6 max-w-lg text-base leading-7 text-white/55">
            Find the strongest ideas, score every moment, and move from source video to
            polished short-form clips in one focused workspace.
          </p>
          <div className="mt-10 grid gap-3 text-sm text-white/70">
            {["AI-ranked highlights", "Transcript-aware analysis", "Creator-ready exports"].map(
              (item) => (
                <div key={item} className="flex items-center gap-3">
                  <CheckCircle weight="fill" className="size-5 text-violet-400" />
                  {item}
                </div>
              ),
            )}
          </div>
        </div>
        <p className="relative text-xs text-white/30">Built for thoughtful creators.</p>
      </section>

      <section className="flex min-h-screen items-center justify-center px-5 py-10 sm:px-10">
        <div className="w-full max-w-md">
          <Link href="/" className="mb-12 flex items-center gap-2 lg:hidden">
            <Image src="/logo_square.png" alt="" width={36} height={36} className="rounded-xl" />
            <span className="font-heading font-bold">ClipStudio AI</span>
          </Link>
          <div className="inline-flex items-center gap-2 rounded-full border bg-white px-3 py-1.5 text-xs font-semibold text-violet-700 shadow-sm">
            <Sparkle weight="fill" />
            {eyebrow}
          </div>
          <h1 className="mt-5 font-heading text-4xl font-bold tracking-tighter">{title}</h1>
          <p className="mt-3 text-sm leading-6 text-muted-foreground">{description}</p>
          {children}
        </div>
      </section>
    </main>
  );
}
