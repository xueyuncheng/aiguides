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
      {
        source: '/auth/:path*',
        destination: 'http://localhost:18080/auth/:path*',
      },
    ];
  },
};

export default nextConfig;
