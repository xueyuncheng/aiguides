export { GAME_HEIGHT, GAME_WIDTH, type CheckpointDefinition, type EnemyDefinition, type HazardDefinition, type LevelConfig, type LevelTheme, type MovingPlatformConfig, type PlatformDefinition, type PlatformTheme } from './types';
export { clockworkKeepLevel } from './level-clockwork-keep';
export { emberRunLevel } from './level-ember-run';
export { moonlitClimbLevel } from './level-moonlit-climb';
export { skybridgeFinaleLevel } from './level-skybridge-finale';
export { sunsetSprintLevel } from './level-sunset-sprint';

import { clockworkKeepLevel } from './level-clockwork-keep';
import { emberRunLevel } from './level-ember-run';
import { moonlitClimbLevel } from './level-moonlit-climb';
import { skybridgeFinaleLevel } from './level-skybridge-finale';
import { sunsetSprintLevel } from './level-sunset-sprint';

export const LEVELS = [sunsetSprintLevel, emberRunLevel, moonlitClimbLevel, clockworkKeepLevel, skybridgeFinaleLevel];
export const DEFAULT_LEVEL = LEVELS[0];
export const TOTAL_RUN_COINS = LEVELS.reduce((sum, level) => sum + level.coins.length, 0);