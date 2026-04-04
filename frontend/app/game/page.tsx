'use client';

import dynamic from 'next/dynamic';
import Link from 'next/link';
import { startTransition, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Coins, Flag, Heart, Maximize, Minimize, Pause, Play, RotateCcw, Sparkles } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { useAuth } from '@/app/contexts/AuthContext';
import { GameSidebar } from './components/GameSidebar';
import { GameTouchControls } from './components/GameTouchControls';
import { GOAL_X } from './game/level';
import { useGamepadControls } from './hooks/useGamepadControls';
import { areGameSnapshotsEqual, INITIAL_GAME_STATE, type GameSnapshot } from './game/state';
import type { ControlInputState, GameSceneHandle } from './types';

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

  const handleStateChange = useCallback((nextState: GameSnapshot) => {
    startTransition(() => {
      setGameState((currentState) => (areGameSnapshotsEqual(currentState, nextState) ? currentState : nextState));
    });
  }, []);

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

  const setTouchInput = useCallback((nextState: ControlInputState) => {
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

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const handleKeydown = (event: KeyboardEvent) => {
      if (event.repeat) {
        return;
      }

      if (event.key === 'r' || event.key === 'R') {
        event.preventDefault();
        handleRestart();
        return;
      }

      if (event.key === 'p' || event.key === 'P' || event.key === 'Escape') {
        event.preventDefault();
        handleTogglePause();
      }
    };

    window.addEventListener('keydown', handleKeydown);
    return () => {
      window.removeEventListener('keydown', handleKeydown);
    };
  }, [handleRestart, handleTogglePause]);

  if (loading || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  const isPaused = gameState.status === 'paused';
  const isFinished = gameState.status === 'won' || gameState.status === 'lost';
  const distanceToGoal = Math.max(0, Math.round((GOAL_X - 30) - gameState.playerX));
  const statusLabel = formatStatus(gameState.status);

  return (
    <main className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.18),_transparent_26%),linear-gradient(180deg,#020617_0%,#0f172a_55%,#111827_100%)] text-white">
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
              <p className="mb-2 text-[11px] uppercase tracking-[0.34em] text-cyan-300/80">Arcade Playground</p>
              <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">/game 超级跳跃试玩</h1>
              <p className="max-w-2xl text-sm text-slate-300">
                保留当前单关平台跳跃玩法，同时把状态同步、跳跃容错和覆盖 HUD 一并优化到更适合持续试玩的版本。
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

        <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <HeroStat label="硬币" value={`${gameState.coinsCollected}/${gameState.totalCoins}`} icon={<Coins className="h-4 w-4 text-amber-300" />} />
          <HeroStat label="生命" value={`${gameState.lives}`} icon={<Heart className="h-4 w-4 text-rose-300" />} />
          <HeroStat label="状态" value={statusLabel} icon={<Sparkles className="h-4 w-4 text-cyan-300" />} />
          <HeroStat label="距终点" value={distanceToGoal === 0 ? '终点已达' : `${distanceToGoal}px`} icon={<Flag className="h-4 w-4 text-emerald-300" />} />
        </section>

        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
          <section className="space-y-4">
            <div
              ref={gameShellRef}
              className="overflow-hidden rounded-[28px] border border-white/10 bg-slate-900/80 p-2 shadow-2xl shadow-slate-950/40 data-[fullscreen=true]:h-screen data-[fullscreen=true]:rounded-none data-[fullscreen=true]:border-0 data-[fullscreen=true]:p-0"
              data-fullscreen={isFullscreen}
            >
              <div className="relative aspect-[16/9] w-full overflow-hidden rounded-[22px] bg-slate-900">
                <div className="pointer-events-none absolute inset-x-0 top-0 z-10 flex items-start justify-between gap-3 p-3">
                  <div className="rounded-2xl border border-white/10 bg-slate-950/55 px-3 py-2 text-xs text-slate-100 backdrop-blur">
                    <p className="uppercase tracking-[0.24em] text-slate-400">控制</p>
                    <p className="mt-1">`P / Esc` 暂停，`R` 重开，`W / Space` 起跳</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-slate-950/55 px-3 py-2 text-right text-xs text-slate-100 backdrop-blur">
                    <p className="uppercase tracking-[0.24em] text-slate-400">输入状态</p>
                    <p className="mt-1">{gamepadConnected ? '已接入手柄' : '键盘 / 触控模式'}</p>
                  </div>
                </div>
                <PhaserGameCanvas onStateChange={handleStateChange} sceneRef={sceneRef} />
                <GameStatusOverlay
                  status={gameState.status}
                  coinsCollected={gameState.coinsCollected}
                  totalCoins={gameState.totalCoins}
                  onRestart={handleRestart}
                  onTogglePause={handleTogglePause}
                />
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
                <div className="mt-3 flex flex-wrap gap-2 text-xs text-slate-300">
                  <span className="rounded-full border border-white/10 bg-black/10 px-3 py-1">更少 React 刷新</span>
                  <span className="rounded-full border border-white/10 bg-black/10 px-3 py-1">跳跃缓冲</span>
                  <span className="rounded-full border border-white/10 bg-black/10 px-3 py-1">边缘起跳容错</span>
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

function HeroStat({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-[28px] border border-white/10 bg-white/6 px-4 py-4 shadow-lg shadow-slate-950/10 backdrop-blur">
      <div className="mb-2 flex items-center gap-2 text-xs uppercase tracking-[0.24em] text-slate-400">
        {icon}
        <span>{label}</span>
      </div>
      <p className="text-2xl font-semibold tracking-tight text-white">{value}</p>
    </div>
  );
}

function GameStatusOverlay({
  status,
  coinsCollected,
  totalCoins,
  onRestart,
  onTogglePause,
}: {
  status: GameSnapshot['status'];
  coinsCollected: number;
  totalCoins: number;
  onRestart: () => void;
  onTogglePause: () => void;
}) {
  if (status === 'running' || status === 'ready') {
    return null;
  }

  const isPaused = status === 'paused';
  const isWon = status === 'won';

  return (
    <div className="absolute inset-0 z-20 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <div className="w-full max-w-md rounded-[32px] border border-white/15 bg-slate-950/80 p-6 text-center shadow-2xl shadow-black/50">
        <p className="text-[11px] uppercase tracking-[0.34em] text-cyan-300/80">
          {isPaused ? 'Paused' : isWon ? 'Stage Clear' : 'Try Again'}
        </p>
        <h2 className="mt-3 text-3xl font-semibold tracking-tight text-white">
          {isPaused ? '游戏已暂停' : isWon ? '你已经通关' : '这次掉空了'}
        </h2>
        <p className="mt-3 text-sm leading-6 text-slate-200">
          {isPaused
            ? '可以直接继续，也可以重开这一局。键盘、触控和手柄的暂停逻辑已统一。'
            : `本局已收集 ${coinsCollected} / ${totalCoins} 枚硬币，${isWon ? '现在可以继续扩关' : '重开后会立刻回到起点'}。`}
        </p>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          {isPaused && (
            <Button className="pointer-events-auto" onClick={onTogglePause}>
              <Play />
              继续游戏
            </Button>
          )}
          <Button
            variant={isPaused ? 'outline' : 'default'}
            className={isPaused ? 'pointer-events-auto border-white/20 bg-white/10 text-white hover:bg-white/20 hover:text-white' : 'pointer-events-auto'}
            onClick={onRestart}
          >
            <RotateCcw />
            重新开始
          </Button>
        </div>
      </div>
    </div>
  );
}

function formatStatus(status: GameSnapshot['status']) {
  if (status === 'won') {
    return '通关';
  }
  if (status === 'lost') {
    return '失败';
  }
  if (status === 'paused') {
    return '暂停中';
  }
  if (status === 'running') {
    return '进行中';
  }
  return '准备中';
}
