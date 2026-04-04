'use client';

import { Trophy } from 'lucide-react';
import { Button } from '@/app/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/app/components/ui/card';
import type { GameSnapshot } from '../game/state';
import type { GamepadDebugInfo } from '../types';

interface GameSidebarProps {
  gameState: GameSnapshot;
  gamepadConnected: boolean;
  gamepadDebug: GamepadDebugInfo[];
  onRestart: () => void;
}

export function GameSidebar({ gameState, gamepadConnected, gamepadDebug, onRestart }: GameSidebarProps) {
  return (
    <aside className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur">
        <CardHeader>
          <CardTitle>游戏状态</CardTitle>
          <CardDescription className="text-slate-300">当前关卡只做了一条可通关路线。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <StatusRow label="硬币" value={`${gameState.coinsCollected} / ${gameState.totalCoins}`} />
          <StatusRow label="生命" value={`${gameState.lives}`} />
          <StatusRow label="跳跃" value={gameState.canJump ? '可起跳' : '空中'} />
          <StatusRow label="手柄" value={gamepadConnected ? '已连接' : '未连接'} />
          <StatusRow label="状态" value={formatStatus(gameState.status)} />
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-white/8 text-white shadow-xl shadow-slate-950/20 backdrop-blur">
        <CardHeader>
          <CardTitle>玩法提示</CardTitle>
          <CardDescription className="text-slate-300">先做了最小可玩版，后续可以继续扩展。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-slate-200">
          <p>目标是收集金币并冲到最右侧旗帜。</p>
          <p>掉出画面会扣命，命用完后可直接重开。</p>
          <p>手柄支持左摇杆或 D-pad 左右移动，`A / B` 跳跃，`Start/Menu` 暂停。</p>
          <p>如果刚插上仍未识别，先按一下手柄任意键，浏览器通常会在首次输入后暴露设备。</p>
          <p>如果你想继续做，我下一步可以补敌人、蘑菇、砖块和音效。</p>
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
