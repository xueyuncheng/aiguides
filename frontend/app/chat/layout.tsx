'use client';

import type { ReactNode } from 'react';
import ChatPageClient from './ChatPageClient';

export default function ChatLayout({ children }: { children: ReactNode }) {
  return (
    <>
      {children}
      <ChatPageClient />
    </>
  );
}
