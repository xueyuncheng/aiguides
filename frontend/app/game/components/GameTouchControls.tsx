'use client';

import type { PointerEvent as ReactPointerEvent } from 'react';
import { ArrowBigLeftDash, ArrowBigRightDash, ArrowUp, Swords } from 'lucide-react';
import type { ControlInputState } from '../types';

const CONTROL_BUTTON_CLASS =
  'flex h-14 min-w-14 items-center justify-center rounded-2xl border border-white/20 bg-slate-950/60 text-white shadow-lg shadow-black/30 backdrop-blur transition active:scale-95 active:bg-orange-500/80';

interface GameTouchControlsProps {
  onInputChange: (nextState: ControlInputState) => void;
}

export function GameTouchControls({ onInputChange }: GameTouchControlsProps) {
  return (
    <div className="grid grid-cols-2 gap-3 rounded-[28px] border border-white/10 bg-white/5 p-3 backdrop-blur sm:w-[320px]">
      <TouchButton
        label="向左"
        icon={<ArrowBigLeftDash className="h-5 w-5" />}
        onPress={() => onInputChange({ left: true })}
        onRelease={() => onInputChange({ left: false })}
      />
      <TouchButton
        label="跳跃"
        icon={<ArrowUp className="h-5 w-5" />}
        onPress={() => onInputChange({ jump: true })}
        onRelease={() => onInputChange({ jump: false })}
      />
      <TouchButton
        label="向右"
        icon={<ArrowBigRightDash className="h-5 w-5" />}
        onPress={() => onInputChange({ right: true })}
        onRelease={() => onInputChange({ right: false })}
      />
      <TouchButton
        label="攻击"
        icon={<Swords className="h-5 w-5" />}
        onPress={() => onInputChange({ attack: true })}
        onRelease={() => onInputChange({ attack: false })}
      />
    </div>
  );
}

function TouchButton({
  label,
  icon,
  onPress,
  onRelease,
}: {
  label: string;
  icon: React.ReactNode;
  onPress: () => void;
  onRelease: () => void;
}) {
  const handlePointerDown = (event: ReactPointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    onPress();
  };

  const handlePointerUp = (event: ReactPointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    onRelease();
  };

  return (
    <button
      type="button"
      className={CONTROL_BUTTON_CLASS}
      onPointerDown={handlePointerDown}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
      onPointerLeave={handlePointerUp}
      onContextMenu={(event) => event.preventDefault()}
    >
      <span className="flex flex-col items-center gap-1 text-xs font-medium tracking-[0.18em] text-slate-100 uppercase">
        {icon}
        <span>{label}</span>
      </span>
    </button>
  );
}
