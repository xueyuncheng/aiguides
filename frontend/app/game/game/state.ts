import { WORLD_WIDTH } from './level';

export interface GameSnapshot {
  coinsCollected: number;
  totalCoins: number;
  status: 'ready' | 'running' | 'paused' | 'won' | 'lost';
  playerX: number;
  lives: number;
  canJump: boolean;
  worldWidth: number;
}

export const INITIAL_GAME_STATE: GameSnapshot = {
  coinsCollected: 0,
  totalCoins: 0,
  status: 'ready',
  playerX: 0,
  lives: 3,
  canJump: false,
  worldWidth: WORLD_WIDTH,
};
