"use client";

import {
  ClockCounterClockwiseIcon as ClockCounterClockwise,
  FilmSlateIcon as FilmSlate,
  HouseIcon as House,
  ListIcon as List,
  SignOutIcon as SignOut,
  SparkleIcon as Sparkle,
} from "@phosphor-icons/react";
import Image from "next/image";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { useEffect } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth-store";

const navigation = [
  { href: "/dashboard", label: "Dashboard", icon: House },
  { href: "/clips", label: "Clips", icon: FilmSlate },
  { href: "/logs", label: "Logs", icon: ClockCounterClockwise },
];

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, accessToken, hydrated, clearSession } = useAuthStore();

  useEffect(() => {
    if (hydrated && !accessToken) {
      router.replace("/signin");
    }
  }, [accessToken, hydrated, router]);

  if (!hydrated || !accessToken) {
    return (
      <div className="grid min-h-screen place-items-center bg-[#f7f5f2]">
        <img src="./logo.png" className="h-14 animate-pulse" />
      </div>
    );
  }

  function signOut() {
    clearSession();
    router.replace("/signin");
  }

  return (
    <div className="min-h-screen bg-[#f7f5f2] text-[#1d1725]">
      <aside className="fixed inset-y-0 left-0 z-30 hidden w-64 flex-col border-r border-black/6 bg-[#20172b] p-5 text-white lg:flex">
        <Link href="/dashboard" className="flex items-center gap-3 px-1 py-2">
          <Image src="/logo_square.png" alt="" width={38} height={38} className="rounded-xl" />
          <div>
            <p className="font-heading text-lg font-bold tracking-[-0.03em]">ClipStudio</p>
            <p className="text-[10px] font-semibold uppercase tracking-[0.22em] text-violet-300">
              AI workspace
            </p>
          </div>
        </Link>

        <nav className="mt-10 space-y-1.5">
          {navigation.map((item) => {
            const active = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition",
                  active
                    ? "bg-white text-[#20172b] shadow-sm"
                    : "text-white/65 hover:bg-white/8 hover:text-white",
                )}
              >
                <item.icon weight={active ? "fill" : "regular"} className="size-5" />
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="mt-auto rounded-2xl border border-white/10 bg-white/5 p-3">
          <p className="truncate text-sm font-medium">{user?.email}</p>
          <p className="mt-1 text-xs text-white/45">Creator workspace</p>
          <Button
            variant="ghost"
            className="mt-3 w-full justify-start text-white/65 hover:bg-white/10 hover:text-white"
            onClick={signOut}
          >
            <SignOut />
            Sign out
          </Button>
        </div>
      </aside>

      <div className="lg:pl-64">
        <header className="sticky top-0 z-20 flex h-16 items-center justify-between border-b border-black/6 bg-[#f7f5f2]/90 px-5 backdrop-blur-xl md:px-8">
          <Link href="/dashboard" className="flex items-center gap-2 font-heading font-bold lg:hidden">
            <Image src="/logo_square.png" alt="" width={30} height={30} className="rounded-lg" />
            ClipStudio AI
          </Link>
          <div className="hidden items-center gap-2 text-sm text-muted-foreground lg:flex">
            <List className="size-4" />
            <span>ClipStudio AI</span>
          </div>
          <div className="flex size-9 items-center justify-center rounded-full bg-violet-600 text-xs font-bold uppercase text-white">
            {user?.email.slice(0, 2) ?? "CS"}
          </div>
        </header>
        <main className="mx-auto max-w-360 px-5 py-7 pb-24 md:px-8 md:py-10 lg:pb-10">
          {children}
        </main>
      </div>

      <nav className="fixed inset-x-4 bottom-4 z-30 flex items-center justify-around rounded-2xl border border-white/70 bg-[#20172b]/95 p-2 text-white shadow-2xl backdrop-blur lg:hidden">
        {navigation.map((item) => {
          const active = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex min-w-20 flex-col items-center gap-1 rounded-xl px-3 py-2 text-[10px] font-medium",
                active ? "bg-white text-[#20172b]" : "text-white/60",
              )}
            >
              <item.icon weight={active ? "fill" : "regular"} className="size-5" />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
