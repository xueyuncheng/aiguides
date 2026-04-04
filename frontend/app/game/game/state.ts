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

export function areGameSnapshotsEqual(left: GameSnapshot, right: GameSnapshot) {
  return (
    left.coinsCollected === right.coinsCollected &&
    left.totalCoins === right.totalCoins &&
    left.status === right.status &&
    left.playerX === right.playerX &&
    left.lives === right.lives &&
    left.canJump === right.canJump &&
    left.worldWidth === right.worldWidth
  );
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
