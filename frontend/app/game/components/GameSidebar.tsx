'use client';

import { Flag, Gamepad2, Heart, Sparkles, Trophy } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import { GOAL_X } from '../game/level';
import type { GameSnapshot } from '../game/state';
import type { GamepadDebugInfo } from '../types';

interface GameSidebarProps {
  gameState: GameSnapshot;
  gamepadConnected: boolean;
  gamepadDebug: GamepadDebugInfo[];
  onRestart: () => void;
}

export function GameSidebar({ gameState, gamepadConnected, gamepadDebug, onRestart }: GameSidebarProps) {
  const distanceToGoal = Math.max(0, Math.round((GOAL_X - 30) - gameState.playerX));
  const completion = gameState.status === 'won' ? 100 : Math.min(100, Math.round((gameState.playerX / (GOAL_X - 30)) * 100));

  return (
    <aside className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur">
        <CardHeader>
          <CardTitle>本局速览</CardTitle>
          <CardDescription className="text-slate-300">把主要状态压成一眼可读的 HUD。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
            <MetricTile icon={<Sparkles className="h-4 w-4 text-amber-300" />} label="硬币" value={`${gameState.coinsCollected} / ${gameState.totalCoins}`} />
            <MetricTile icon={<Heart className="h-4 w-4 text-rose-300" />} label="生命" value={`${gameState.lives}`} />
            <MetricTile icon={<Flag className="h-4 w-4 text-sky-300" />} label="距终点" value={distanceToGoal === 0 ? '已抵达' : `${distanceToGoal}px`} />
            <MetricTile icon={<Gamepad2 className="h-4 w-4 text-emerald-300" />} label="输入设备" value={gamepadConnected ? '手柄在线' : '键盘 / 触控'} />
          </div>
          <div className="rounded-3xl border border-white/10 bg-black/15 px-4 py-3">
            <div className="mb-2 flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-400">
              <span>关卡完成度</span>
              <span>{completion}%</span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-white/10">
              <div className="h-full rounded-full bg-gradient-to-r from-sky-400 via-cyan-300 to-emerald-300" style={{ width: `${completion}%` }} />
            </div>
          </div>
          <StatusRow label="状态" value={formatStatus(gameState.status)} />
          <StatusRow label="跳跃窗口" value={gameState.canJump ? '地面 / 缓冲可用' : '空中判定'} />
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur">
        <CardHeader>
          <CardTitle>操作说明</CardTitle>
          <CardDescription className="text-slate-300">把当前版本最常用的输入统一列出来。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-slate-200">
          <p>目标是收齐硬币并冲到最右侧旗帜，掉出画面会扣命。</p>
          <p>键盘支持方向键或 `A / D` 移动，`W / Space / ↑` 跳跃，`P` 或 `Esc` 暂停，`R` 重开。</p>
          <p>手柄支持左摇杆或 D-pad 左右移动，`A / B` 跳跃，`Start / Menu` 暂停。</p>
          <p>移动端按钮已加上 pointer capture，长按时不会轻易因为滑出按钮而丢输入。</p>
          <p>跳跃新增了缓冲和 coyote time，落点边缘也更容易接住输入。</p>
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur sm:col-span-2 xl:col-span-1">
        <CardHeader>
          <CardTitle>手柄调试</CardTitle>
          <CardDescription className="text-slate-300">这里显示浏览器当前读到的 Gamepad 状态。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-slate-200">
          {gamepadDebug.length === 0 ? (
            <p>当前没有检测到已连接手柄。插上后请先按一下任意键。</p>
          ) : (
            gamepadDebug.map((pad) => (
              <div key={`${pad.index}-${pad.id}`} className="rounded-2xl border border-white/10 bg-black/10 p-3">
                <p className="font-medium text-white">#{pad.index} {pad.id}</p>
                <p className="mt-1 text-slate-300">映射: {pad.mapping}</p>
                <p className="mt-1 text-slate-300">轴: {pad.axes}</p>
                <p className="mt-1 text-slate-300">按下按钮: {pad.pressedButtons}</p>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      {(gameState.status === 'won' || gameState.status === 'lost') && (
        <Card className="border-white/10 bg-gradient-to-br from-amber-500/20 via-orange-500/15 to-rose-500/20 text-white shadow-xl shadow-slate-950/20 backdrop-blur sm:col-span-2 xl:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Trophy className="text-amber-300" />
              {gameState.status === 'won' ? '恭喜通关' : '这次失误了'}
            </CardTitle>
            <CardDescription className="text-slate-200">
              {gameState.status === 'won'
                ? '你已经跑到终点，可以继续让我补第二关。'
                : '点重开就能再来一把。'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button className="w-full" onClick={onRestart}>
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
    <div className="flex items-center justify-between rounded-2xl border border-white/10 bg-black/10 px-3 py-2">
      <span className="text-slate-300">{label}</span>
      <span className="font-medium text-white">{value}</span>
    </div>
  );
}

function MetricTile({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-black/10 px-3 py-3">
      <div className="mb-2 flex items-center gap-2 text-xs uppercase tracking-[0.2em] text-slate-400">
        {icon}
        <span>{label}</span>
      </div>
      <p className="text-base font-semibold text-white">{value}</p>
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
  return '进行中';
}
