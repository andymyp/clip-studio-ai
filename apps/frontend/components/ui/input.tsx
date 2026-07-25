"use client";

import { EyeIcon as Eye, EyeSlashIcon as EyeSlash } from "@phosphor-icons/react";
import * as React from "react";

import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  const [passwordVisible, setPasswordVisible] = React.useState(false);
  const password = type === "password";

  const input = (
    <input
      type={password && passwordVisible ? "text" : type}
      data-slot="input"
      className={cn(
        "h-11 w-full rounded-xl border border-input bg-background px-3.5 text-sm shadow-xs outline-none transition-[border,box-shadow] placeholder:text-muted-foreground focus-visible:border-primary focus-visible:ring-3 focus-visible:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-50",
        password && "pr-11",
        className,
      )}
      {...props}
    />
  );

  if (!password) return input;

  return (
    <div className="relative">
      {input}
      <button
        type="button"
        aria-label={passwordVisible ? "Hide password" : "Show password"}
        aria-pressed={passwordVisible}
        className="absolute right-1.5 top-1.5 grid size-8 place-items-center rounded-lg text-muted-foreground transition hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 disabled:pointer-events-none"
        onClick={() => setPasswordVisible((visible) => !visible)}
      >
        {passwordVisible ? <EyeSlash className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  );
}

export { Input };
