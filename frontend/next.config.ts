import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  compress: false,
  /* config options here */
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:18080/api/:path*',
      },
    ];
  },
};

export default nextConfig;
