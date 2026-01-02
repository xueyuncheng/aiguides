import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  compress: false,
  output: 'standalone',
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
      {
        source: '/config',
        destination: 'http://localhost:18080/config',
      },
      {
        source: '/health',
        destination: 'http://localhost:18080/health',
      },
    ];
  },
};

export default nextConfig;
