import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  async redirects() {
    return [
      { source: "/chat", destination: "/", permanent: true },
      { source: "/projects", destination: "/", permanent: true },
      { source: "/settings", destination: "/tools", permanent: true },
      { source: "/plugins", destination: "/tools", permanent: true },
      { source: "/knowledge", destination: "/memory", permanent: true },
    ];
  },
};

export default nextConfig;
