'use client';

import { Clock3, Flag, Heart, MapPinned, Sparkles, Swords, Trophy } from 'lucide-react';
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

  return (
    <aside className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur">
        <CardHeader>
          <CardTitle>{gameState.levelName}</CardTitle>
          <CardDescription className="text-slate-300">{gameState.levelTagline}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
            <MetricTile icon={<Sparkles className="h-4 w-4 text-amber-300" />} label="关卡硬币" value={`${gameState.coinsCollected} / ${gameState.totalCoins}`} />
            <MetricTile icon={<Heart className="h-4 w-4 text-rose-300" />} label="生命" value={`${gameState.lives}`} />
            <MetricTile icon={<Flag className="h-4 w-4 text-sky-300" />} label="距终点" value={distanceToGoal === 0 ? '已抵达' : `${distanceToGoal}px`} />
            <MetricTile icon={<MapPinned className="h-4 w-4 text-emerald-300" />} label="Checkpoint" value={gameState.checkpointLabel} />
          </div>
          <div className="rounded-3xl border border-white/10 bg-black/15 px-4 py-3">
            <div className="mb-2 flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-400">
              <span>当前关卡进度</span>
              <span>{completion}%</span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-white/10">
              <div className="h-full rounded-full bg-gradient-to-r from-sky-400 via-cyan-300 to-emerald-300" style={{ width: `${completion}%` }} />
            </div>
          </div>
          <StatusRow label="流程关卡" value={`${gameState.levelNumber} / ${gameState.totalLevels}`} />
          <StatusRow label="状态" value={formatStatus(gameState.status)} />
          <StatusRow label="跳跃状态" value={gameState.canJump ? '起跳 / 二段跳可衔接' : '空中移动中'} />
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur">
        <CardHeader>
          <CardTitle>跑分状态</CardTitle>
          <CardDescription className="text-slate-300">现在不只是通关，还能看完整段流程表现。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm text-slate-200">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
            <MetricTile icon={<Sparkles className="h-4 w-4 text-cyan-300" />} label="总得分" value={`${gameState.score}`} />
            <MetricTile icon={<Swords className="h-4 w-4 text-orange-300" />} label="击败敌人" value={`${gameState.defeatedEnemies}`} />
            <MetricTile icon={<Clock3 className="h-4 w-4 text-sky-300" />} label="总用时" value={formatElapsed(gameState.elapsedSeconds)} />
            <MetricTile icon={<Flag className="h-4 w-4 text-amber-300" />} label="总收集" value={`${gameState.runCoinsCollected} / ${gameState.runCoinsTotal}`} />
            <MetricTile icon={<Heart className="h-4 w-4 text-rose-300" />} label="失误次数" value={`${gameState.deathCount}`} />
          </div>
          <StatusRow label="输入设备" value={gamepadConnected ? '手柄在线' : '键盘 / 触控'} />
          <StatusRow label="攻击状态" value={gameState.isAttacking ? '挥击中' : gameState.canAttack ? '可攻击' : '冷却中'} />
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur">
        <CardHeader>
          <CardTitle>操作说明</CardTitle>
          <CardDescription className="text-slate-300">这版已经从单关试玩扩成了一个短流程平台挑战。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-slate-200">
          <p>目标是穿过两关，收集硬币、踩掉敌人并躲开尖刺或熔岩。</p>
          <p>键盘支持方向键或 `A / D` 移动，`W / Space / ↑` 跳跃与二段跳，`J` 近战攻击，`P` 或 `Esc` 暂停，`R` 重开。</p>
          <p>过关后可直接进下一关；第二关加入了更长危险区和垂直移动平台。</p>
          <p>移动端按钮继续保留 pointer capture，长按输入更稳，跳跃也保留缓冲、coyote time 和一次带冲击特效的空中二段跳。</p>
          <p>手柄支持左摇杆或 D-pad 左右移动，`A / B` 跳跃，`X / RB` 攻击，`Start / Menu` 暂停。</p>
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

      {(gameState.status === 'level-complete' || gameState.status === 'won' || gameState.status === 'lost') && (
        <Card className="border-white/10 bg-gradient-to-br from-amber-500/20 via-orange-500/15 to-rose-500/20 text-white shadow-xl shadow-slate-950/20 backdrop-blur sm:col-span-2 xl:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Trophy className="text-amber-300" />
              {gameState.status === 'level-complete' ? '继续推进流程' : gameState.status === 'won' ? '恭喜通关' : '这次失误了'}
            </CardTitle>
            <CardDescription className="text-slate-200">
              {gameState.status === 'level-complete'
                ? '这一关已经结算完成，可以直接切入下一关。'
                : gameState.status === 'won'
                  ? '双关卡短流程已经打穿，接下来可以往 Boss 或随机挑战扩展。'
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

function formatElapsed(elapsedSeconds: number) {
  const minutes = Math.floor(elapsedSeconds / 60);
  const seconds = elapsedSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}
