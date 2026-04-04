'use client';

const CONTROL_BUTTON_CLASS =
  'h-14 min-w-14 rounded-2xl border border-white/30 bg-white/15 text-white shadow-lg backdrop-blur active:scale-95';

interface GameTouchControlsProps {
  onInputChange: (nextState: { left?: boolean; right?: boolean; jump?: boolean }) => void;
}

export function GameTouchControls({ onInputChange }: GameTouchControlsProps) {
  return (
    <div className="grid grid-cols-3 gap-3 sm:w-[220px]">
      <TouchButton
        label="左"
        onPress={() => onInputChange({ left: true })}
        onRelease={() => onInputChange({ left: false })}
      />
      <TouchButton
        label="跳"
        onPress={() => onInputChange({ jump: true })}
        onRelease={() => onInputChange({ jump: false })}
      />
      <TouchButton
        label="右"
        onPress={() => onInputChange({ right: true })}
        onRelease={() => onInputChange({ right: false })}
      />
    </div>
  );
}

function TouchButton({
  label,
  onPress,
  onRelease,
}: {
  label: string;
  onPress: () => void;
  onRelease: () => void;
}) {
  return (
    <button
      type="button"
      className={CONTROL_BUTTON_CLASS}
      onPointerDown={onPress}
      onPointerUp={onRelease}
      onPointerCancel={onRelease}
      onPointerLeave={onRelease}
    >
      {label}
    </button>
  );
}
