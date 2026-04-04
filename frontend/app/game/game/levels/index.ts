export { GAME_HEIGHT, GAME_WIDTH, type CheckpointDefinition, type EnemyDefinition, type HazardDefinition, type LevelConfig, type LevelTheme, type MovingPlatformConfig, type PlatformDefinition, type PlatformTheme } from './types';
export { emberRunLevel } from './level-ember-run';
export { sunsetSprintLevel } from './level-sunset-sprint';

import { emberRunLevel } from './level-ember-run';
import { sunsetSprintLevel } from './level-sunset-sprint';

export const LEVELS = [sunsetSprintLevel, emberRunLevel];
export const DEFAULT_LEVEL = LEVELS[0];
export const TOTAL_RUN_COINS = LEVELS.reduce((sum, level) => sum + level.coins.length, 0);