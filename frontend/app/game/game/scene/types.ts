import type * as Phaser from 'phaser';
import type { CheckpointDefinition, EnemyDefinition, PlatformDefinition } from '../levels';

export type SceneStartData = {
  levelIndex?: number;
  lives?: number;
  deathCount?: number;
  defeatedEnemies?: number;
  runCoinsCollected?: number;
  runStartedAt?: number;
  totalFrozenMs?: number;
};

export type InputState = { left: boolean; right: boolean; jump: boolean; attack: boolean };

export type RespawnPoint = {
  x: number;
  y: number;
  label: string;
};

export type MovingPlatformInstance = {
  block: Phaser.GameObjects.Rectangle;
  body: Phaser.Physics.Arcade.StaticBody;
  config: PlatformDefinition;
  originX: number;
  originY: number;
  previousX: number;
  previousY: number;
  deltaX: number;
  deltaY: number;
};

export type EnemyInstance = {
  sprite: Phaser.Physics.Arcade.Sprite;
  config: EnemyDefinition;
  direction: 1 | -1;
};

export type CheckpointInstance = {
  definition: CheckpointDefinition;
  zone: Phaser.GameObjects.Zone;
  beacon: Phaser.GameObjects.Arc;
  halo: Phaser.GameObjects.Arc;
  active: boolean;
};
