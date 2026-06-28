"use client";

import type { ReactNode } from "react";
import Navigation from "./Navigation";
import TopAppBar from "./TopAppBar";

export default function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-base-100">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-6 pb-24">
        <TopAppBar />
        <main className="mt-2">{children}</main>
      </div>
      <Navigation />
    </div>
  );
}
