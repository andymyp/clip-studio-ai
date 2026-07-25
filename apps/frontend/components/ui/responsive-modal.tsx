"use client";

import * as React from "react";
import { XIcon } from "@phosphor-icons/react";
import { Dialog as DialogPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

type ResponsiveModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
};

function ResponsiveModal({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
  className,
}: ResponsiveModalProps) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay
          data-slot="responsive-modal-overlay"
          className="fixed inset-0 z-50 bg-black/55 backdrop-blur-[2px] data-[state=closed]:animate-out data-[state=open]:animate-in data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
        />
        <DialogPrimitive.Content
          data-slot="responsive-modal-content"
          className={cn(
            "fixed inset-x-0 bottom-0 z-50 max-h-[92dvh] overflow-y-auto rounded-t-3xl border bg-background shadow-2xl outline-none data-[state=closed]:animate-out data-[state=open]:animate-in data-[state=closed]:slide-out-to-bottom data-[state=open]:slide-in-from-bottom sm:inset-auto sm:left-1/2 sm:top-1/2 sm:w-[calc(100%-2rem)] sm:max-w-lg sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-2xl sm:data-[state=closed]:zoom-out-95 sm:data-[state=open]:zoom-in-95 sm:data-[state=closed]:slide-out-to-bottom-0 sm:data-[state=open]:slide-in-from-bottom-0",
            className,
          )}
        >
          <div className="mx-auto mt-3 h-1.5 w-10 rounded-full bg-muted-foreground/20 sm:hidden" />
          <div className="flex items-start gap-4 px-5 pb-4 pt-5 sm:px-6 sm:pt-6">
            <div className="min-w-0 flex-1">
              <DialogPrimitive.Title className="font-heading text-xl font-semibold tracking-[-0.025em]">
                {title}
              </DialogPrimitive.Title>
              {description && (
                <DialogPrimitive.Description className="mt-1.5 text-sm leading-6 text-muted-foreground">
                  {description}
                </DialogPrimitive.Description>
              )}
            </div>
            <DialogPrimitive.Close
              className="grid size-9 shrink-0 place-items-center rounded-full text-muted-foreground transition hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
              aria-label="Close modal"
            >
              <XIcon className="size-4" />
            </DialogPrimitive.Close>
          </div>
          <div className="px-5 pb-5 sm:px-6 sm:pb-6">{children}</div>
          {footer && (
            <div className="flex flex-col-reverse gap-2 border-t bg-muted/30 px-5 py-4 sm:flex-row sm:justify-end sm:px-6">
              {footer}
            </div>
          )}
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}

export { ResponsiveModal };
