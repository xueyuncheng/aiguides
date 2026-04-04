export const GAME_WIDTH = 960;
export const GAME_HEIGHT = 540;

export type PlatformTheme = 'moss' | 'ember' | 'cloud';

export interface MovingPlatformConfig {
  axis: 'x' | 'y';
  distance: number;
  duration: number;
  phase?: number;
}

export interface PlatformDefinition {
  x: number;
  y: number;
  width: number;
  height: number;
  theme?: PlatformTheme;
  movement?: MovingPlatformConfig;
}

export interface EnemyDefinition {
  x: number;
  y: number;
  minX: number;
  maxX: number;
  speed: number;
}

export interface HazardDefinition {
  x: number;
  y: number;
  width: number;
  height: number;
  kind: 'spikes' | 'lava';
}

export interface CheckpointDefinition {
  x: number;
  y: number;
  label: string;
}

export interface LevelTheme {
  sky: number;
  mist: number;
  hillNear: number;
  hillFar: number;
  accent: number;
  ground: number;
}

export interface LevelConfig {
  id: string;
  name: string;
  tagline: string;
  worldWidth: number;
  floorY: number;
  playerStart: { x: number; y: number };
  goalX: number;
  platforms: PlatformDefinition[];
  coins: { x: number; y: number }[];
  enemies: EnemyDefinition[];
  hazards: HazardDefinition[];
  checkpoints: CheckpointDefinition[];
  theme: LevelTheme;
}