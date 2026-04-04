import type { GameSnapshot } from './game/state';

export type GameSceneHandle = {
  pauseGame: () => void;
  resumeGame: () => void;
  restartGame: () => void;
  setTouchInput: (nextState: { left?: boolean; right?: boolean; jump?: boolean }) => void;
  setGamepadInput: (nextState: { left?: boolean; right?: boolean; jump?: boolean }) => void;
};

export type GamepadDebugInfo = {
  index: number;
  id: string;
  mapping: string;
  axes: string;
  pressedButtons: string;
};

export type GameStatus = GameSnapshot['status'];
