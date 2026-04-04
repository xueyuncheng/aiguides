'use client';

import dynamic from 'next/dynamic';
import Link from 'next/link';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Maximize, Minimize, Pause, Play, RotateCcw } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { useAuth } from '@/app/contexts/AuthContext';
import { GameSidebar } from './components/GameSidebar';
import { GameTouchControls } from './components/GameTouchControls';
import { GOAL_X } from './game/level';
import { useGamepadControls } from './hooks/useGamepadControls';
import { INITIAL_GAME_STATE, type GameSnapshot } from './game/state';
import type { GameSceneHandle } from './types';

const PhaserGameCanvas = dynamic(
  () => import('./components/PhaserGameCanvas').then((mod) => mod.PhaserGameCanvas),
  { ssr: false }
);

export default function GamePage() {
  const router = useRouter();
  const { user, loading } = useAuth();
  const sceneRef = useRef<GameSceneHandle | null>(null);
  const gameShellRef = useRef<HTMLDivElement | null>(null);
  const [gameState, setGameState] = useState<GameSnapshot>(INITIAL_GAME_STATE);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const { gamepadConnected, gamepadDebug } = useGamepadControls(sceneRef, gameState);

  useEffect(() => {
    if (!loading && !user) {
      router.push('/login');
    }
  }, [loading, router, user]);

  const progress = useMemo(() => {
    if (gameState.status === 'won') {
      return 100;
    }

    const finishLine = GOAL_X - 30;
    if (finishLine <= 0) {
      return 0;
    }

    return Math.max(0, Math.min(100, (gameState.playerX / finishLine) * 100));
  }, [gameState.playerX, gameState.status]);

  const handleTogglePause = useCallback(() => {
    if (!sceneRef.current) {
      return;
    }

    if (gameState.status === 'paused') {
      sceneRef.current.resumeGame();
      return;
    }

    if (gameState.status === 'running') {
      sceneRef.current.pauseGame();
    }
  }, [gameState.status]);

  const handleRestart = useCallback(() => {
    sceneRef.current?.restartGame();
  }, []);

  const setTouchInput = useCallback((nextState: { left?: boolean; right?: boolean; jump?: boolean }) => {
    sceneRef.current?.setTouchInput(nextState);
  }, []);

  const handleToggleFullscreen = useCallback(async () => {
    const container = gameShellRef.current;
    if (!container) {
      return;
    }

    const fullscreenElement =
      document.fullscreenElement ??
      ((document as Document & { webkitFullscreenElement?: Element }).webkitFullscreenElement ?? null);

    if (fullscreenElement) {
      if (document.exitFullscreen) {
        await document.exitFullscreen();
        return;
      }

      const webkitExitFullscreen = (document as Document & { webkitExitFullscreen?: () => Promise<void> | void })
        .webkitExitFullscreen;
      if (webkitExitFullscreen) {
        await webkitExitFullscreen.call(document);
      }
      return;
    }

    if (container.requestFullscreen) {
      await container.requestFullscreen();
      return;
    }

    const webkitRequestFullscreen = (
      container as HTMLDivElement & { webkitRequestFullscreen?: () => Promise<void> | void }
    ).webkitRequestFullscreen;
    if (webkitRequestFullscreen) {
      await webkitRequestFullscreen.call(container);
    }
  }, []);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const syncFullscreenState = () => {
      const fullscreenElement =
        document.fullscreenElement ??
        ((document as Document & { webkitFullscreenElement?: Element }).webkitFullscreenElement ?? null);
      setIsFullscreen(Boolean(fullscreenElement));
    };

    syncFullscreenState();
    document.addEventListener('fullscreenchange', syncFullscreenState);
    document.addEventListener('webkitfullscreenchange', syncFullscreenState as EventListener);

    return () => {
      document.removeEventListener('fullscreenchange', syncFullscreenState);
      document.removeEventListener('webkitfullscreenchange', syncFullscreenState as EventListener);
    };
  }, []);

  if (loading || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  const isPaused = gameState.status === 'paused';
  const isFinished = gameState.status === 'won' || gameState.status === 'lost';

  return (
    <main className="min-h-screen bg-slate-950 text-white">
      <div className="mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="space-y-2">
            <Button asChild variant="ghost" className="-ml-3 w-fit text-slate-200 hover:bg-white/10 hover:text-white">
              <Link href="/chat">
                <ArrowLeft />
                返回聊天
              </Link>
            </Button>
            <div>
              <h1 className="text-3xl font-semibold tracking-tight">/game 超级跳跃试玩</h1>
              <p className="text-sm text-slate-300">
                单关平台跳跃 Demo。支持键盘、屏幕按钮和标准手柄。
              </p>
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <Button
              variant="outline"
              className="border-white/20 bg-white/10 text-white hover:bg-white/20 hover:text-white"
              onClick={() => void handleToggleFullscreen()}
            >
              {isFullscreen ? <Minimize /> : <Maximize />}
              {isFullscreen ? '退出全屏' : '全屏'}
            </Button>
            <Button
              variant="outline"
              className="border-white/20 bg-white/10 text-white hover:bg-white/20 hover:text-white"
              onClick={handleTogglePause}
              disabled={isFinished}
            >
              {isPaused ? <Play /> : <Pause />}
              {isPaused ? '继续' : '暂停'}
            </Button>
            <Button
              variant="outline"
              className="border-white/20 bg-white/10 text-white hover:bg-white/20 hover:text-white"
              onClick={handleRestart}
            >
              <RotateCcw />
              重开
            </Button>
          </div>
        </div>

        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
          <section className="space-y-4">
            <div
              ref={gameShellRef}
              className="overflow-hidden rounded-[28px] border border-white/10 bg-slate-900/80 p-2 shadow-2xl shadow-slate-950/40 data-[fullscreen=true]:h-screen data-[fullscreen=true]:rounded-none data-[fullscreen=true]:border-0 data-[fullscreen=true]:p-0"
              data-fullscreen={isFullscreen}
            >
              <div className="aspect-[16/9] w-full overflow-hidden rounded-[22px] bg-slate-900">
                <PhaserGameCanvas onStateChange={setGameState} sceneRef={sceneRef as React.MutableRefObject<unknown>} />
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
              <div className="rounded-3xl border border-white/10 bg-white/5 px-4 py-4 backdrop-blur">
                <div className="mb-2 flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-300">
                  <span>进度</span>
                  <span>{Math.round(progress)}%</span>
                </div>
                <div className="h-3 overflow-hidden rounded-full bg-white/10">
                  <div className="h-full rounded-full bg-gradient-to-r from-amber-400 via-orange-500 to-rose-500" style={{ width: `${progress}%` }} />
                </div>
              </div>

              <GameTouchControls onInputChange={setTouchInput} />
            </div>
          </section>

          <GameSidebar
            gameState={gameState}
            gamepadConnected={gamepadConnected}
            gamepadDebug={gamepadDebug}
            onRestart={handleRestart}
          />
        </div>
      </div>
    </main>
  );
}
