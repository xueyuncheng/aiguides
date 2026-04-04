import type { GameSnapshot } from './game/state';

export type ControlInputState = {
  left?: boolean;
  right?: boolean;
  jump?: boolean;
};

export type GameSceneHandle = {
  pauseGame: () => void;
  resumeGame: () => void;
  restartGame: () => void;
  setTouchInput: (nextState: ControlInputState) => void;
  setGamepadInput: (nextState: ControlInputState) => void;
};

export type GamepadDebugInfo = {
  index: number;
  id: string;
  mapping: string;
  axes: string;
  pressedButtons: string;
};

export type GameStatus = GameSnapshot['status'];
