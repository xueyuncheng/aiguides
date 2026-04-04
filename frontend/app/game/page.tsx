'use client';

import dynamic from 'next/dynamic';
import Link from 'next/link';
import { startTransition, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Clock3, Flag, Layers3, Maximize, Minimize, Pause, Play, RotateCcw, Sparkles } from 'lucide-react';
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

  const isPaused = gameState.status === 'paused';
  const isRunFinished = gameState.status === 'won' || gameState.status === 'lost';
  const isStageLocked = gameState.status === 'level-complete' || isRunFinished;
  const distanceToGoal = Math.max(0, Math.round((gameState.goalX - 30) - gameState.playerX));
  const statusLabel = formatStatus(gameState.status);
  const runTimer = formatElapsed(gameState.elapsedSeconds);

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
                现在已经升级成双关卡短流程：有巡逻敌人、危险地形、checkpoint 和跑分结算，方向从单关原型切到可持续扩展的小型平台游戏。
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
              disabled={isStageLocked}
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
          <HeroStat label="关卡" value={`${gameState.levelNumber}/${gameState.totalLevels}`} icon={<Layers3 className="h-4 w-4 text-cyan-300" />} />
          <HeroStat label="得分" value={`${gameState.score}`} icon={<Sparkles className="h-4 w-4 text-amber-300" />} />
          <HeroStat label="计时" value={runTimer} icon={<Clock3 className="h-4 w-4 text-sky-300" />} />
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
                    <p className="mt-1">`P / Esc` 暂停，`R` 重开，`W / Space` 起跳，过关后 `Enter` 进入下一关</p>
                  </div>
                  <div className="rounded-2xl border border-white/10 bg-slate-950/55 px-3 py-2 text-right text-xs text-slate-100 backdrop-blur">
                    <p className="uppercase tracking-[0.24em] text-slate-400">输入状态</p>
                    <p className="mt-1">{gamepadConnected ? '已接入手柄' : '键盘 / 触控模式'} · {statusLabel}</p>
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

            <div className="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
              <div className="rounded-3xl border border-white/10 bg-white/5 px-4 py-4 backdrop-blur">
                <div className="mb-2 flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-300">
                  <span>整段进度</span>
                  <span>{Math.round(progress)}%</span>
                </div>
                <div className="h-3 overflow-hidden rounded-full bg-white/10">
                  <div className="h-full rounded-full bg-gradient-to-r from-amber-400 via-orange-500 to-rose-500" style={{ width: `${progress}%` }} />
                </div>
                <div className="mt-3 flex flex-wrap gap-2 text-xs text-slate-300">
                  <span className="rounded-full border border-white/10 bg-black/10 px-3 py-1">双关卡短流程</span>
                  <span className="rounded-full border border-white/10 bg-black/10 px-3 py-1">checkpoint 复活</span>
                  <span className="rounded-full border border-white/10 bg-black/10 px-3 py-1">敌人 / 危险区 / 移动平台</span>
                </div>
              </div>

              <GameTouchControls onInputChange={setTouchInput} />
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
      <div className="w-full max-w-md rounded-[32px] border border-white/15 bg-slate-950/80 p-6 text-center shadow-2xl shadow-black/50">
        <p className="text-[11px] uppercase tracking-[0.34em] text-cyan-300/80">
          {isPaused ? 'Paused' : isLevelComplete ? 'Checkpoint Locked In' : isWon ? 'Run Complete' : 'Try Again'}
        </p>
        <h2 className="mt-3 text-3xl font-semibold tracking-tight text-white">
          {isPaused ? '游戏已暂停' : isLevelComplete ? `第 ${levelNumber} 关完成` : isWon ? '整段流程通关' : '这次翻车了'}
        </h2>
        <p className="mt-3 text-sm leading-6 text-slate-200">
          {isPaused
            ? '可以直接继续，也可以重开这一局。键盘、触控和手柄的暂停逻辑已统一。'
            : isLevelComplete
              ? `当前关卡收集 ${coinsCollected} / ${totalCoins} 枚硬币，最近 checkpoint 是“${checkpointLabel}”。按 Enter 或点按钮进入第 ${Math.min(levelNumber + 1, totalLevels)} 关。`
              : isWon
                ? `本次流程总分 ${score}，总用时 ${formatElapsed(elapsedSeconds)}。你已经把当前双关卡短流程跑完了。`
                : `最近 checkpoint 是“${checkpointLabel}”。点重开会从第一关重新开始整段流程。`}
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
