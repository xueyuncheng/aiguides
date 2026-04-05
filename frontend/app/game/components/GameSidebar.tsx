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
      <Card className="border-amber-200/80 bg-[#fff8e5]/90 text-slate-900 shadow-xl shadow-orange-200/40 backdrop-blur">
        <CardHeader>
          <CardTitle>{gameState.levelName}</CardTitle>
          <CardDescription className="text-slate-600">{gameState.levelTagline}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
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
          <StatusRow label="跳跃状态" value={gameState.canJump ? '地面起跳或二段跳已就绪' : '空中移动中'} />
        </CardContent>
      </Card>

      <Card className="border-amber-200/80 bg-[#fff8e5]/90 text-slate-900 shadow-xl shadow-orange-200/40 backdrop-blur">
        <CardHeader>
          <CardTitle>跑分状态</CardTitle>
          <CardDescription className="text-slate-600">现在不只是通关，还能看完整段怀旧流程的表现。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm text-slate-700">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
            <MetricTile icon={<Sparkles className="h-4 w-4 text-cyan-300" />} label="总得分" value={`${gameState.score}`} />
            <MetricTile icon={<Swords className="h-4 w-4 text-orange-300" />} label="踩掉小怪" value={`${gameState.defeatedEnemies}`} />
            <MetricTile icon={<Clock3 className="h-4 w-4 text-sky-300" />} label="总用时" value={formatElapsed(gameState.elapsedSeconds)} />
            <MetricTile icon={<Flag className="h-4 w-4 text-amber-300" />} label="总收集" value={`${gameState.runCoinsCollected} / ${gameState.runCoinsTotal}`} />
            <MetricTile icon={<Heart className="h-4 w-4 text-rose-300" />} label="跌落次数" value={`${gameState.deathCount}`} />
          </div>
          <StatusRow label="输入设备" value={gamepadConnected ? '手柄在线' : '键盘 / 触控'} />
          <StatusRow label="攻击状态" value={gameState.isAttacking ? '冲刺挥击中' : gameState.canAttack ? '已就绪' : '冷却中'} />
        </CardContent>
      </Card>

      <Card className="border-amber-200/80 bg-[#fff8e5]/90 text-slate-900 shadow-xl shadow-orange-200/40 backdrop-blur">
        <CardHeader>
          <CardTitle>操作说明</CardTitle>
          <CardDescription className="text-slate-600">这版现在是偏经典主机味道的 {gameState.totalLevels} 关横版试玩。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-slate-700">
          <p>目标是穿过 {gameState.totalLevels} 关，收集金币、踩掉巡逻小怪，并避开尖刺和熔岩。</p>
          <p>键盘支持方向键或 A / D 移动，W / Space / ↑ 跳跃与二段跳，J 冲刺攻击，P 或 Esc 暂停，R 重开。</p>
          <p>过关后可直接进下一关；后半段会逐步切进更密集的移动浮台、机关砖塔和终盘云桥。</p>
          <p>移动端按钮继续保留 pointer capture，长按输入更稳，跳跃仍然保留缓冲、 coyote time 和一次空中二段跳。</p>
          <p>手柄支持左摇杆或 D-pad 左右移动，A / B 跳跃，X / RB 攻击，Start / Menu 暂停。</p>
        </CardContent>
      </Card>

      <Card className="border-amber-200/80 bg-[#fff8e5]/90 text-slate-900 shadow-xl shadow-orange-200/40 backdrop-blur sm:col-span-2 xl:col-span-1">
        <CardHeader>
          <CardTitle>手柄调试</CardTitle>
          <CardDescription className="text-slate-600">这里显示浏览器当前读到的 Gamepad 状态。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-slate-700">
          {gamepadDebug.length === 0 ? (
            <p>当前没有检测到已连接手柄。插上后请先按一下任意键。</p>
          ) : (
            gamepadDebug.map((pad) => (
              <div key={`${pad.index}-${pad.id}`} className="rounded-2xl border border-orange-100 bg-white/75 p-3">
                <p className="font-medium text-slate-900">#{pad.index} {pad.id}</p>
                <p className="mt-1 text-slate-600">映射: {pad.mapping}</p>
                <p className="mt-1 text-slate-600">轴: {pad.axes}</p>
                <p className="mt-1 text-slate-600">按下按钮: {pad.pressedButtons}</p>
              </div>
            ))
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

function formatElapsed(elapsedSeconds: number) {
  const minutes = Math.floor(elapsedSeconds / 60);
  const seconds = elapsedSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}
