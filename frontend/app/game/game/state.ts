import { DEFAULT_LEVEL, LEVELS, TOTAL_RUN_COINS } from './level';

export type GameStatus = 'ready' | 'running' | 'paused' | 'level-complete' | 'won' | 'lost';

export interface GameSnapshot {
  levelNumber: number;
  totalLevels: number;
  levelName: string;
  levelTagline: string;
  coinsCollected: number;
  totalCoins: number;
  runCoinsCollected: number;
  runCoinsTotal: number;
  status: GameStatus;
  playerX: number;
  lives: number;
  deathCount: number;
  score: number;
  defeatedEnemies: number;
  elapsedSeconds: number;
  checkpointLabel: string;
  canJump: boolean;
  canAttack: boolean;
  isAttacking: boolean;
  worldWidth: number;
  goalX: number;
}

export function areGameSnapshotsEqual(left: GameSnapshot, right: GameSnapshot) {
  return (
    left.levelNumber === right.levelNumber &&
    left.totalLevels === right.totalLevels &&
    left.levelName === right.levelName &&
    left.levelTagline === right.levelTagline &&
    left.coinsCollected === right.coinsCollected &&
    left.totalCoins === right.totalCoins &&
    left.runCoinsCollected === right.runCoinsCollected &&
    left.runCoinsTotal === right.runCoinsTotal &&
    left.status === right.status &&
    left.playerX === right.playerX &&
    left.lives === right.lives &&
    left.deathCount === right.deathCount &&
    left.score === right.score &&
    left.defeatedEnemies === right.defeatedEnemies &&
    left.elapsedSeconds === right.elapsedSeconds &&
    left.checkpointLabel === right.checkpointLabel &&
    left.canJump === right.canJump &&
    left.canAttack === right.canAttack &&
    left.isAttacking === right.isAttacking &&
    left.worldWidth === right.worldWidth &&
    left.goalX === right.goalX
  );
}

export const INITIAL_GAME_STATE: GameSnapshot = {
  levelNumber: 1,
  totalLevels: LEVELS.length,
  levelName: DEFAULT_LEVEL.name,
  levelTagline: DEFAULT_LEVEL.tagline,
  coinsCollected: 0,
  totalCoins: DEFAULT_LEVEL.coins.length,
  runCoinsCollected: 0,
  runCoinsTotal: TOTAL_RUN_COINS,
  status: 'ready',
  playerX: 0,
  lives: 100,
  deathCount: 0,
  score: 0,
  defeatedEnemies: 0,
  elapsedSeconds: 0,
  checkpointLabel: '起点',
  canJump: false,
  canAttack: true,
  isAttacking: false,
  worldWidth: DEFAULT_LEVEL.worldWidth,
  goalX: DEFAULT_LEVEL.goalX,
};
