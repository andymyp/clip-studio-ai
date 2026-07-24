import { Inter, Noto_Sans } from "next/font/google";
import type { Metadata } from "next";
import type { ReactNode } from "react";

import { AppProviders } from "@/components/providers/app-providers";
import { cn } from "@/lib/utils";

import "./globals.css";

const notoSansHeading = Noto_Sans({ subsets: ["latin"], variable: "--font-heading" });
const inter = Inter({ subsets: ["latin"], variable: "--font-sans" });

export const metadata: Metadata = {
  title: "ClipStudio AI",
  description: "AI-assisted video clipping workspace",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={cn("font-sans", inter.variable, notoSansHeading.variable)}
    >
      <body>
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
