'use client';

import dynamic from 'next/dynamic';
import { Press_Start_2P } from 'next/font/google';
import Link from 'next/link';
import { startTransition, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Layers3, Maximize, Minimize, Play, RotateCcw } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { useAuth } from '@/app/contexts/AuthContext';
import { GameSidebar } from './components/GameSidebar';
import { GameTouchControls } from './components/GameTouchControls';
import { useGamepadControls } from './hooks/useGamepadControls';
import { areGameSnapshotsEqual, INITIAL_GAME_STATE, type GameSnapshot } from './game/state';
import type { ControlInputState, GameSceneHandle } from './types';

const PhaserGameCanvas = dynamic(
  () => import('./components/PhaserGameCanvas').then((mod) => mod.PhaserGameCanvas),
  { ssr: false }
);

const retroDisplay = Press_Start_2P({ subsets: ['latin'], weight: '400' });

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

  const currentLevelProgress = useMemo(() => {
    if (gameState.status === 'won') {
      return 1;
    }

    const finishLine = gameState.goalX - 30;
    if (finishLine <= 0) {
      return 0;
    }

    return Math.max(0, Math.min(1, gameState.playerX / finishLine));
  }, [gameState.goalX, gameState.playerX, gameState.status]);

  const progress = useMemo(
    () => Math.max(0, Math.min(100, (((gameState.levelNumber - 1) + currentLevelProgress) / gameState.totalLevels) * 100)),
    [currentLevelProgress, gameState.levelNumber, gameState.totalLevels]
  );

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

  const handleAdvanceLevel = useCallback(() => {
    sceneRef.current?.advanceToNextLevel();
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

      if (event.key === 'Enter' && gameState.status === 'level-complete') {
        event.preventDefault();
        handleAdvanceLevel();
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
  }, [gameState.status, handleAdvanceLevel, handleRestart, handleTogglePause]);

  if (loading || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  const statusLabel = formatStatus(gameState.status);
  const canvasFrameStyle = isFullscreen
    ? undefined
    : { maxWidth: 'min(100%, max(18rem, calc((100dvh - 20rem) * 16 / 9)))' };

  return (
    <main className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(255,243,168,0.42),_transparent_22%),linear-gradient(180deg,#7fd7ff_0%,#9be1ff_26%,#d9f7ff_58%,#fff5d7_100%)] text-slate-950">
      <div className="mx-auto flex min-h-screen w-full max-w-[1440px] flex-col gap-4 px-4 py-4 sm:px-6 lg:px-8">
        <Button asChild variant="ghost" className="-ml-3 h-7 w-fit text-slate-700 hover:bg-white/55 hover:text-slate-950">
          <Link href="/chat">
            <ArrowLeft />
            返回聊天
          </Link>
        </Button>

        <section className="flex items-center justify-between gap-3 rounded-[20px] border border-white/60 bg-white/50 px-3 py-2.5 shadow-[0_12px_34px_rgba(255,173,78,0.12)] backdrop-blur-xl">
          <div className="inline-flex items-center gap-2 rounded-full border border-white/80 bg-white/72 px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm shadow-orange-200/10">
            <Layers3 className="h-3.5 w-3.5 text-cyan-500" />
            <span>第 {gameState.levelNumber} / {gameState.totalLevels} 关</span>
          </div>

          <div className="flex items-center gap-1.5">
            <Button
              variant="outline"
              size="sm"
              className="border-orange-200 bg-white/80 text-slate-900 hover:bg-white hover:text-slate-950"
              onClick={() => void handleToggleFullscreen()}
            >
              {isFullscreen ? <Minimize /> : <Maximize />}
              {isFullscreen ? '退出全屏' : '全屏'}
            </Button>
          </div>
        </section>

        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px]">
          <section className="space-y-4">
            <div
              ref={gameShellRef}
              className="overflow-hidden rounded-[28px] border border-orange-200/80 bg-[#fff8e8]/80 p-2 shadow-2xl shadow-orange-200/40 data-[fullscreen=true]:h-screen data-[fullscreen=true]:rounded-none data-[fullscreen=true]:border-0 data-[fullscreen=true]:bg-slate-950 data-[fullscreen=true]:p-0"
              data-fullscreen={isFullscreen}
            >
              <div className="mx-auto w-full" style={canvasFrameStyle}>
                <div className="relative aspect-[16/9] w-full overflow-hidden rounded-[22px] bg-slate-900">
                  <div className="pointer-events-none absolute inset-x-0 top-0 z-10 flex flex-wrap items-start justify-between gap-2 p-3 sm:p-4">
                    <div className="rounded-full border border-white/15 bg-[#22304d]/74 px-3 py-2 text-[11px] uppercase tracking-[0.2em] text-amber-100 backdrop-blur">
                      A / D 移动 · Space 跳跃 · J 攻击 · P 暂停 · R 重开
                    </div>
                    <div className="rounded-full border border-white/15 bg-[#22304d]/74 px-3 py-2 text-[11px] uppercase tracking-[0.2em] text-amber-100 backdrop-blur">
                      {gamepadConnected ? 'Gamepad' : 'Keyboard'} · {statusLabel}
                    </div>
                  </div>
                  <PhaserGameCanvas onStateChange={handleStateChange} sceneRef={sceneRef} />
                  <GameStatusOverlay
                    levelNumber={gameState.levelNumber}
                    totalLevels={gameState.totalLevels}
                    status={gameState.status}
                    coinsCollected={gameState.coinsCollected}
                    totalCoins={gameState.totalCoins}
                    score={gameState.score}
                    elapsedSeconds={gameState.elapsedSeconds}
                    checkpointLabel={gameState.checkpointLabel}
                    onAdvanceLevel={handleAdvanceLevel}
                    onRestart={handleRestart}
                    onTogglePause={handleTogglePause}
                  />
                </div>
              </div>
            </div>

            <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
              <div className="rounded-[24px] border border-orange-200/80 bg-[#fff8e6]/92 px-4 py-4 shadow-lg shadow-orange-200/30 backdrop-blur">
                <div className="mb-2 flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-500">
                  <span>整段进度</span>
                  <span>{Math.round(progress)}%</span>
                </div>
                <div className="h-3 overflow-hidden rounded-full bg-orange-100">
                  <div className="h-full rounded-full bg-gradient-to-r from-orange-500 via-amber-400 to-lime-500" style={{ width: `${progress}%` }} />
                </div>
                <div className="mt-3 flex flex-wrap gap-2 text-xs text-slate-600">
                  <span className="rounded-full border border-orange-100 bg-white/75 px-3 py-1">存档点: {gameState.checkpointLabel}</span>
                  <span className="rounded-full border border-orange-100 bg-white/75 px-3 py-1">{gameState.coinsCollected} / {gameState.totalCoins} 金币</span>
                  <span className="rounded-full border border-orange-100 bg-white/75 px-3 py-1">{gameState.defeatedEnemies} 次踩怪</span>
                </div>
              </div>

              <div className="md:hidden">
                <GameTouchControls onInputChange={setTouchInput} />
              </div>
            </div>
          </section>

          <GameSidebar
            gameState={gameState}
            gamepadConnected={gamepadConnected}
            gamepadDebug={gamepadDebug}
            onAdvanceLevel={handleAdvanceLevel}
            onRestart={handleRestart}
          />
        </div>
      </div>
    </main>
  );
}

function GameStatusOverlay({
  levelNumber,
  totalLevels,
  status,
  coinsCollected,
  totalCoins,
  score,
  elapsedSeconds,
  checkpointLabel,
  onAdvanceLevel,
  onRestart,
  onTogglePause,
}: {
  levelNumber: number;
  totalLevels: number;
  status: GameSnapshot['status'];
  coinsCollected: number;
  totalCoins: number;
  score: number;
  elapsedSeconds: number;
  checkpointLabel: string;
  onAdvanceLevel: () => void;
  onRestart: () => void;
  onTogglePause: () => void;
}) {
  if (status === 'running' || status === 'ready') {
    return null;
  }

  const isPaused = status === 'paused';
  const isLevelComplete = status === 'level-complete';
  const isWon = status === 'won';

  return (
    <div className="absolute inset-0 z-20 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <div className="w-full max-w-md rounded-[32px] border border-amber-200/30 bg-slate-950/82 p-6 text-center shadow-2xl shadow-black/50">
        <p className="text-[11px] uppercase tracking-[0.34em] text-amber-300/80">
          {isPaused ? 'Paused' : isLevelComplete ? 'Checkpoint Locked In' : isWon ? 'Run Complete' : 'Try Again'}
        </p>
        <h2 className={`${retroDisplay.className} mt-3 text-2xl leading-[1.5] tracking-[0.08em] text-white`}>
          {isPaused ? '游戏已暂停' : isLevelComplete ? `第 ${levelNumber} 关完成` : isWon ? '整段流程通关' : '这次翻车了'}
        </h2>
        <p className="mt-3 text-sm leading-6 text-slate-200">
          {isPaused
            ? '可以直接继续，也可以重开这一局。键盘、触控和手柄的暂停逻辑已经统一。'
            : isLevelComplete
              ? `当前关卡收集 ${coinsCollected} / ${totalCoins} 枚金币，最近存档点是“${checkpointLabel}”。按 Enter 或点按钮进入第 ${Math.min(levelNumber + 1, totalLevels)} 关。`
              : isWon
                ? `本次流程总分 ${score}，总用时 ${formatElapsed(elapsedSeconds)}。你已经把当前 ${totalLevels} 关怀旧流程跑完了。`
                : `最近存档点是“${checkpointLabel}”。点重开会从第一关重新开始整段流程。`}
        </p>
        <div className="mt-4 grid gap-2 rounded-3xl border border-white/10 bg-black/20 px-4 py-3 text-left text-sm text-slate-200 sm:grid-cols-2">
          <div>
            <p className="text-[11px] uppercase tracking-[0.24em] text-slate-400">当前关卡</p>
            <p className="mt-1 font-medium text-white">{levelNumber} / {totalLevels}</p>
          </div>
          <div>
            <p className="text-[11px] uppercase tracking-[0.24em] text-slate-400">当前得分</p>
            <p className="mt-1 font-medium text-white">{score}</p>
          </div>
        </div>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          {isPaused && (
            <Button className="pointer-events-auto" onClick={onTogglePause}>
              <Play />
              继续游戏
            </Button>
          )}
          {isLevelComplete && (
            <Button className="pointer-events-auto" onClick={onAdvanceLevel}>
              <Play />
              进入下一关
            </Button>
          )}
          <Button
            variant={isPaused || isLevelComplete ? 'outline' : 'default'}
            className={isPaused || isLevelComplete ? 'pointer-events-auto border-white/20 bg-white/10 text-white hover:bg-white/20 hover:text-white' : 'pointer-events-auto'}
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
  if (status === 'level-complete') {
    return '过关结算';
  }
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

function formatElapsed(elapsedSeconds: number) {
  const minutes = Math.floor(elapsedSeconds / 60);
  const seconds = elapsedSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}
