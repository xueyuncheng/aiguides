'use client';

import { useCallback, useEffect, useRef } from 'react';
import type { Session } from '@/app/components/SessionSidebar';

type LoadSessions = (silent?: boolean) => Promise<Session[] | undefined>;

export function useTitlePoller(loadSessions: LoadSessions, intervalMs: number, maxPolls: number) {
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const stopPoll = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  const startPoll = useCallback((sessionId: string) => {
    stopPoll();
    let pollCount = 0;
    intervalRef.current = setInterval(async () => {
      const fetchedSessions = await loadSessions(true);
      const current = fetchedSessions?.find((s) => s.session_id === sessionId);
      if (current?.title) {
        stopPoll();
        return;
      }
      pollCount++;
      if (pollCount >= maxPolls) stopPoll();
    }, intervalMs);
  }, [loadSessions, intervalMs, maxPolls, stopPoll]);

  useEffect(() => () => stopPoll(), [stopPoll]);

  return { startPoll, stopPoll };
}
