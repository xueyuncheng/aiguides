'use client';

import { Flag, Heart, MapPinned, Sparkles, Trophy } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import type { GameSnapshot } from '../game/state';
import type { GamepadDebugInfo } from '../types';

interface GameSidebarProps {
  gameState: GameSnapshot;
  gamepadConnected: boolean;
  gamepadDebug: GamepadDebugInfo[];
  onAdvanceLevel: () => void;
  onRestart: () => void;
}

export function GameSidebar({ gameState, gamepadConnected, gamepadDebug, onAdvanceLevel, onRestart }: GameSidebarProps) {
  const finishLine = Math.max(1, gameState.goalX - 30);
  const distanceToGoal = Math.max(0, Math.round(finishLine - gameState.playerX));
  const completion = gameState.status === 'won' ? 100 : Math.min(100, Math.round((gameState.playerX / finishLine) * 100));
  const shouldShowDebugPanel = gamepadConnected || gamepadDebug.length > 0;

  return (
    <aside className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1 xl:self-start">
      <Card className="border-amber-200/80 bg-[#fff8e5]/90 text-slate-900 shadow-xl shadow-orange-200/40 backdrop-blur">
        <CardHeader className="pb-4">
          <CardTitle>关卡概览</CardTitle>
          <CardDescription className="text-slate-600">{gameState.levelTagline}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <div className="rounded-[24px] border border-orange-100 bg-white/75 px-4 py-4">
            <p className="text-xs uppercase tracking-[0.22em] text-slate-500">当前场景</p>
            <p className="mt-2 text-lg font-semibold text-slate-900">{gameState.levelName}</p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
            <MetricTile icon={<Sparkles className="h-4 w-4 text-amber-300" />} label="关卡硬币" value={`${gameState.coinsCollected} / ${gameState.totalCoins}`} />
            <MetricTile icon={<Heart className="h-4 w-4 text-rose-300" />} label="生命" value={`${gameState.lives}`} />
            <MetricTile icon={<Flag className="h-4 w-4 text-sky-300" />} label="距终点" value={distanceToGoal === 0 ? '已抵达' : `${distanceToGoal}px`} />
            <MetricTile icon={<MapPinned className="h-4 w-4 text-emerald-300" />} label="存档点" value={gameState.checkpointLabel} />
          </div>
          <div className="rounded-3xl border border-orange-100 bg-white/75 px-4 py-3">
            <div className="mb-2 flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-500">
              <span>当前关卡进度</span>
              <span>{completion}%</span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-orange-100">
              <div className="h-full rounded-full bg-gradient-to-r from-orange-500 via-amber-400 to-lime-500" style={{ width: `${completion}%` }} />
            </div>
          </div>
          <StatusRow label="流程关卡" value={`${gameState.levelNumber} / ${gameState.totalLevels}`} />
          <StatusRow label="状态" value={formatStatus(gameState.status)} />
          <StatusRow label="跳跃状态" value={gameState.canJump ? '落地可起跳 / 二段跳可用' : '空中移动中'} />
        </CardContent>
      </Card>

      <Card className="border-amber-200/80 bg-[#fff8e5]/90 text-slate-900 shadow-xl shadow-orange-200/40 backdrop-blur sm:col-span-2 xl:col-span-1">
        <CardHeader className="pb-4">
          <CardTitle>简明提示</CardTitle>
          <CardDescription className="text-slate-600">把长说明收成一块，不再挤占首屏空间。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-slate-700">
          <p>目标是穿过 {gameState.totalLevels} 关，拿金币、躲机关、踩掉巡逻小怪，最后冲过旗台。</p>
          <p>键盘用 A / D 或方向键移动，W / Space / ↑ 起跳，J 攻击，P 或 Esc 暂停，R 立即重开。</p>
          <p>如果接入手柄，可用左摇杆或方向键移动，A / B 跳跃，X / RB 攻击，Start / Menu 暂停。</p>

          {shouldShowDebugPanel && (
            <details className="rounded-[22px] border border-orange-100 bg-white/75 px-4 py-3">
              <summary className="cursor-pointer list-none text-sm font-medium text-slate-900">查看输入诊断</summary>
              <div className="mt-3 space-y-3 text-sm text-slate-700">
                {gamepadDebug.map((pad) => (
                  <div key={`${pad.index}-${pad.id}`} className="rounded-2xl border border-orange-100 bg-[#fff8e5] p-3">
                    <p className="font-medium text-slate-900">#{pad.index} {pad.id}</p>
                    <p className="mt-1 text-slate-600">映射: {pad.mapping}</p>
                    <p className="mt-1 text-slate-600">轴: {pad.axes}</p>
                    <p className="mt-1 text-slate-600">按下按钮: {pad.pressedButtons}</p>
                  </div>
                ))}
              </div>
            </details>
          )}
        </CardContent>
      </Card>

      {(gameState.status === 'level-complete' || gameState.status === 'won' || gameState.status === 'lost') && (
        <Card className="border-amber-200/80 bg-gradient-to-br from-[#fff2b9] via-[#ffd980] to-[#ffb96f] text-slate-900 shadow-xl shadow-orange-200/45 backdrop-blur sm:col-span-2 xl:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Trophy className="text-amber-600" />
              {gameState.status === 'level-complete' ? '继续推进流程' : gameState.status === 'won' ? '恭喜通关' : '这次失误了'}
            </CardTitle>
            <CardDescription className="text-slate-700">
              {gameState.status === 'level-complete'
                ? '这一关已经结算完成，可以直接切入下一关。'
                : gameState.status === 'won'
                  ? `当前 ${gameState.totalLevels} 关怀旧流程已经打穿，后面还可以继续扩 Boss 关或隐藏路线。`
                  : '这次掉空或被碰掉了，点重开就能整段再来一把。'}
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3">
            {gameState.status === 'level-complete' && (
              <Button className="w-full" onClick={onAdvanceLevel}>
                进入下一关
              </Button>
            )}
            <Button className="w-full" variant={gameState.status === 'level-complete' ? 'outline' : 'default'} onClick={onRestart}>
              再来一次
            </Button>
          </CardContent>
        </Card>
      )}
    </aside>
  );
}

function StatusRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-2xl border border-orange-100 bg-white/75 px-3 py-2">
      <span className="text-slate-600">{label}</span>
      <span className="font-medium text-slate-900">{value}</span>
    </div>
  );
}

function MetricTile({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-orange-100 bg-white/75 px-3 py-3">
      <div className="mb-2 flex items-center gap-2 text-xs uppercase tracking-[0.2em] text-slate-500">
        {icon}
        <span>{label}</span>
      </div>
      <p className="text-base font-semibold text-slate-900">{value}</p>
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
  return '进行中';
}

