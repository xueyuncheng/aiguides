import * as Phaser from 'phaser';
import type { GameSnapshot, GameStatus } from './state';
import {
  DEFAULT_LEVEL,
  GAME_HEIGHT,
  LEVELS,
  TOTAL_RUN_COINS,
  type LevelConfig,
} from './level';
import {
  createCheckpoints,
  createCoins,
  createEnemies,
  createGoal,
  createHazards,
  createPlatforms,
  createPlayer,
  drawBackdrop,
} from './scene/builders';
import type {
  CheckpointInstance,
  EnemyInstance,
  InputState,
  MovingPlatformInstance,
  RespawnPoint,
  SceneStartData,
} from './scene/types';

const MOVE_SPEED = 220;
const JUMP_SPEED = -420;
const SHORT_HOP_SPEED = -210;
const JUMP_BUFFER_MS = 140;
const COYOTE_TIME_MS = 110;
const DAMAGE_COOLDOWN_MS = 900;
const STATE_POSITION_STEP = 18;

export class GameScene extends Phaser.Scene {
  private readonly onStateChange: (state: GameSnapshot) => void;
  private levelIndex = 0;
  private level: LevelConfig = DEFAULT_LEVEL;
  private player?: Phaser.Physics.Arcade.Sprite;
  private cursors?: Phaser.Types.Input.Keyboard.CursorKeys;
  private keys?: {
    left: Phaser.Input.Keyboard.Key;
    right: Phaser.Input.Keyboard.Key;
    jump: Phaser.Input.Keyboard.Key;
  };
  private platforms?: Phaser.Physics.Arcade.StaticGroup;
  private coins?: Phaser.Physics.Arcade.Group;
  private status: GameStatus = 'ready';
  private coinsCollected = 0;
  private totalCoins = 0;
  private lives = 3;
  private deathCount = 0;
  private runCoinsCollected = 0;
  private touchState: InputState = { left: false, right: false, jump: false };
  private gamepadState: InputState = { left: false, right: false, jump: false };
  private previousJumpRequested = false;
  private jumpQueuedUntil = 0;
  private lastGroundedAt = 0;
  private lastPublishedState?: GameSnapshot;
  private movingPlatforms: MovingPlatformInstance[] = [];
  private enemies: EnemyInstance[] = [];
  private checkpoints: CheckpointInstance[] = [];
  private respawnPoint: RespawnPoint = {
    x: DEFAULT_LEVEL.playerStart.x,
    y: DEFAULT_LEVEL.playerStart.y,
    label: '起点',
  };
  private checkpointLabel = '起点';
  private runStartedAt = 0;
  private totalFrozenMs = 0;
  private freezeStartedAt: number | null = null;
  private damageCooldownUntil = 0;

  constructor(onStateChange: (state: GameSnapshot) => void) {
    super('GameScene');
    this.onStateChange = onStateChange;
  }

  init(data: SceneStartData = {}) {
    this.levelIndex = data.levelIndex ?? 0;
    this.level = LEVELS[this.levelIndex] ?? DEFAULT_LEVEL;
    this.lives = data.lives ?? 3;
    this.deathCount = data.deathCount ?? 0;
    this.runCoinsCollected = data.runCoinsCollected ?? 0;
    this.runStartedAt = data.runStartedAt ?? 0;
    this.totalFrozenMs = data.totalFrozenMs ?? 0;
    this.freezeStartedAt = null;
  }

  create() {
    if (!this.runStartedAt) {
      this.runStartedAt = this.time.now;
    }

    this.resetLevelState();
    this.cameras.main.setBounds(0, 0, this.level.worldWidth, GAME_HEIGHT);
    this.physics.world.setBounds(0, 0, this.level.worldWidth, GAME_HEIGHT + 140);

    drawBackdrop(this, this.level, this.levelIndex);
    const platformResult = createPlatforms(this, this.level);
    this.platforms = platformResult.platforms;
    this.movingPlatforms = platformResult.movingPlatforms;
    createGoal(this, this.level);
    this.player = createPlayer(this, this.level, this.platforms, this.movingPlatforms);
    this.coins = createCoins(this, this.level, this.player, () => {
      this.coinsCollected += 1;
      this.emitState(true);
    });
    this.checkpoints = createCheckpoints(this, this.level, this.player, (label) => {
      this.activateCheckpoint(label);
    });
    createHazards(this, this.level, this.player, () => {
      this.loseLife();
    });
    this.enemies = createEnemies(
      this,
      this.level,
      this.player,
      this.platforms,
      this.movingPlatforms,
      this.handlePlayerEnemyCollision,
      this
    );
    this.createControls();
    this.bindSceneEvents();

    this.status = 'running';
    this.emitState(true);
  }

  update(_: number, delta: number) {
    if (!this.player?.body || this.status !== 'running') {
      return;
    }

    this.updateMovingPlatforms();
    this.updateEnemies();

    const now = this.time.now;
    const body = this.player.body as Phaser.Physics.Arcade.Body;
    const isGrounded = body.blocked.down || body.touching.down;

    const moveLeft =
      this.touchState.left || this.gamepadState.left || !!this.cursors?.left.isDown || !!this.keys?.left.isDown;
    const moveRight =
      this.touchState.right || this.gamepadState.right || !!this.cursors?.right.isDown || !!this.keys?.right.isDown;
    const wantsJump =
      this.touchState.jump ||
      this.gamepadState.jump ||
      !!this.cursors?.up.isDown ||
      !!this.cursors?.space?.isDown ||
      !!this.keys?.jump.isDown;

    if (isGrounded) {
      this.lastGroundedAt = now;
    }

    if (wantsJump && !this.previousJumpRequested) {
      this.jumpQueuedUntil = now + JUMP_BUFFER_MS;
    }

    if (moveLeft === moveRight) {
      this.player.setVelocityX(0);
    } else if (moveLeft) {
      this.player.setVelocityX(-MOVE_SPEED);
      this.player.setFlipX(true);
    } else {
      this.player.setVelocityX(MOVE_SPEED);
      this.player.setFlipX(false);
    }

    const hasJumpBuffer = this.jumpQueuedUntil >= now;
    const hasGroundGrace = isGrounded || now - this.lastGroundedAt <= COYOTE_TIME_MS;
    if (hasJumpBuffer && hasGroundGrace) {
      this.player.setVelocityY(JUMP_SPEED);
      this.jumpQueuedUntil = 0;
      this.lastGroundedAt = now - COYOTE_TIME_MS - delta;
      this.touchState.jump = false;
      this.gamepadState.jump = false;
    } else if (!wantsJump && body.velocity.y < SHORT_HOP_SPEED) {
      this.player.setVelocityY(SHORT_HOP_SPEED);
    }

    this.previousJumpRequested = wantsJump;

    if (this.player.y > GAME_HEIGHT + 80) {
      this.loseLife();
      return;
    }

    if (this.player.x >= this.level.goalX - 30) {
      this.handleGoalReached();
      return;
    }

    this.emitState();
  }

  setTouchInput(nextState: Partial<InputState>) {
    this.touchState = { ...this.touchState, ...nextState };
  }

  setGamepadInput(nextState: Partial<InputState>) {
    this.gamepadState = { ...this.gamepadState, ...nextState };
  }

  pauseGame() {
    if (this.status !== 'running') {
      return;
    }

    this.status = 'paused';
    this.physics.world.pause();
    this.freezeRunTimer();
    this.emitState();
  }

  resumeGame() {
    if (this.status !== 'paused') {
      return;
    }

    this.status = 'running';
    this.physics.world.resume();
    this.resumeRunTimer();
    this.emitState();
  }

  restartGame() {
    this.scene.restart({
      levelIndex: 0,
      lives: 3,
      deathCount: 0,
      runCoinsCollected: 0,
      runStartedAt: 0,
      totalFrozenMs: 0,
    } satisfies SceneStartData);
  }

  advanceToNextLevel() {
    if (this.status !== 'level-complete' || this.levelIndex >= LEVELS.length - 1) {
      return;
    }

    this.scene.restart({
      levelIndex: this.levelIndex + 1,
      lives: this.lives,
      deathCount: this.deathCount,
      runCoinsCollected: this.getRunCoinCount(),
      runStartedAt: this.runStartedAt,
      totalFrozenMs: this.getTotalFrozenMs(),
    } satisfies SceneStartData);
  }

  private bindSceneEvents() {
    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.touchState = { left: false, right: false, jump: false };
      this.gamepadState = { left: false, right: false, jump: false };
      this.previousJumpRequested = false;
      this.jumpQueuedUntil = 0;
      this.lastGroundedAt = 0;
      this.lastPublishedState = undefined;
      this.freezeStartedAt = null;
    });
  }

  private createControls() {
    this.cursors = this.input.keyboard?.createCursorKeys();

    if (this.input.keyboard) {
      this.keys = {
        left: this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.A),
        right: this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.D),
        jump: this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.W),
      };
    }
  }

  private resetLevelState() {
    this.status = 'ready';
    this.coinsCollected = 0;
    this.totalCoins = this.level.coins.length;
    this.touchState = { left: false, right: false, jump: false };
    this.gamepadState = { left: false, right: false, jump: false };
    this.previousJumpRequested = false;
    this.jumpQueuedUntil = 0;
    this.lastGroundedAt = this.time.now;
    this.lastPublishedState = undefined;
    this.movingPlatforms = [];
    this.enemies = [];
    this.checkpoints = [];
    this.respawnPoint = {
      x: this.level.playerStart.x,
      y: this.level.playerStart.y,
      label: '起点',
    };
    this.checkpointLabel = '起点';
    this.damageCooldownUntil = 0;
    this.freezeStartedAt = null;
  }

  private updateMovingPlatforms() {
    if (this.movingPlatforms.length === 0) {
      return;
    }

    const playerBody = this.player?.body as Phaser.Physics.Arcade.Body | undefined;

    for (const platform of this.movingPlatforms) {
      platform.previousX = platform.block.x;
      platform.previousY = platform.block.y;

      const movement = platform.config.movement;
      if (!movement) {
        platform.deltaX = 0;
        platform.deltaY = 0;
        continue;
      }

      const angle = ((this.time.now / movement.duration) + (movement.phase ?? 0)) * Math.PI * 2;
      const offset = Math.sin(angle) * movement.distance;
      const nextX = platform.originX + (movement.axis === 'x' ? offset : 0);
      const nextY = platform.originY + (movement.axis === 'y' ? offset : 0);

      platform.block.setPosition(nextX, nextY);
      platform.body.updateFromGameObject();
      platform.deltaX = nextX - platform.previousX;
      platform.deltaY = nextY - platform.previousY;

      if (!playerBody || this.status !== 'running') {
        continue;
      }

      const isStandingOnPlatform =
        (playerBody.blocked.down || playerBody.touching.down) &&
        Math.abs(playerBody.bottom - platform.body.top) < 14 &&
        playerBody.right > platform.body.left + 10 &&
        playerBody.left < platform.body.right - 10;

      if (isStandingOnPlatform) {
        this.player?.setPosition((this.player?.x ?? 0) + platform.deltaX, (this.player?.y ?? 0) + platform.deltaY);
      }
    }
  }

  private updateEnemies() {
    this.enemies = this.enemies.filter((enemy) => enemy.sprite.active);

    for (const enemy of this.enemies) {
      if (!enemy.sprite.body) {
        continue;
      }

      if (enemy.sprite.x <= enemy.config.minX) {
        enemy.direction = 1;
      } else if (enemy.sprite.x >= enemy.config.maxX) {
        enemy.direction = -1;
      }

      enemy.sprite.setVelocityX(enemy.config.speed * enemy.direction);
      enemy.sprite.setFlipX(enemy.direction < 0);
    }
  }

  private handlePlayerEnemyCollision(
    playerObject:
      | Phaser.Types.Physics.Arcade.GameObjectWithBody
      | Phaser.Physics.Arcade.Body
      | Phaser.Physics.Arcade.StaticBody
      | Phaser.Tilemaps.Tile,
    enemyObject:
      | Phaser.Types.Physics.Arcade.GameObjectWithBody
      | Phaser.Physics.Arcade.Body
      | Phaser.Physics.Arcade.StaticBody
      | Phaser.Tilemaps.Tile
  ) {
    if (this.status !== 'running' || !this.player) {
      return;
    }

    if (!('body' in playerObject) || !('body' in enemyObject)) {
      return;
    }

    const playerBody = playerObject.body as Phaser.Physics.Arcade.Body;
    const enemyBody = enemyObject.body as Phaser.Physics.Arcade.Body;

    const stompedEnemy =
      playerBody.velocity.y > 120 &&
      playerBody.bottom <= enemyBody.top + 18 &&
      playerBody.center.x > enemyBody.left - 12 &&
      playerBody.center.x < enemyBody.right + 12;

    if (stompedEnemy) {
      enemyObject.destroy();
      this.player.setVelocityY(JUMP_SPEED * 0.58);
      this.emitState(true);
      return;
    }

    this.loseLife();
  }

  private activateCheckpoint(label: string) {
    const nextCheckpoint = this.checkpoints.find((checkpoint) => checkpoint.definition.label === label);
    if (!nextCheckpoint || nextCheckpoint.active) {
      return;
    }

    this.checkpoints.forEach((checkpoint) => {
      checkpoint.active = checkpoint === nextCheckpoint;
      checkpoint.banner.setFillStyle(checkpoint.active ? 0x22c55e : 0x475569);
    });

    this.checkpointLabel = label;
    this.respawnPoint = {
      x: nextCheckpoint.definition.x,
      y: nextCheckpoint.definition.y,
      label,
    };
    this.emitState(true);
  }

  private handleGoalReached() {
    if (!this.player) {
      return;
    }

    this.player.setVelocity(0, 0);

    if (this.levelIndex < LEVELS.length - 1) {
      this.status = 'level-complete';
      this.physics.world.pause();
      this.freezeRunTimer();
      this.emitState(true);
      return;
    }

    this.status = 'won';
    this.physics.world.pause();
    this.freezeRunTimer();
    this.emitState(true);
  }

  private loseLife() {
    if (!this.player || this.status !== 'running' || this.time.now < this.damageCooldownUntil) {
      return;
    }

    this.damageCooldownUntil = this.time.now + DAMAGE_COOLDOWN_MS;
    this.deathCount += 1;
    this.lives -= 1;

    if (this.lives <= 0) {
      this.status = 'lost';
      this.physics.world.pause();
      this.freezeRunTimer();
      this.player.setVelocity(0, 0);
      this.player.setTint(0xfb7185);
      this.emitState(true);
      return;
    }

    this.player.setTint(0xfdba74);
    this.player.setVelocity(0, 0);
    this.player.setPosition(this.respawnPoint.x, this.respawnPoint.y);
    this.jumpQueuedUntil = 0;
    this.lastGroundedAt = this.time.now;
    this.time.delayedCall(DAMAGE_COOLDOWN_MS, () => {
      this.player?.clearTint();
    });
    this.emitState(true);
  }

  private freezeRunTimer() {
    if (this.freezeStartedAt === null) {
      this.freezeStartedAt = this.time.now;
    }
  }

  private resumeRunTimer() {
    if (this.freezeStartedAt !== null) {
      this.totalFrozenMs += this.time.now - this.freezeStartedAt;
      this.freezeStartedAt = null;
    }
  }

  private getElapsedSeconds() {
    const referenceTime = this.freezeStartedAt ?? this.time.now;
    return Math.max(0, Math.floor((referenceTime - this.runStartedAt - this.totalFrozenMs) / 1000));
  }

  private getTotalFrozenMs() {
    return this.totalFrozenMs + (this.freezeStartedAt === null ? 0 : this.time.now - this.freezeStartedAt);
  }

  private getRunCoinCount() {
    return this.runCoinsCollected + this.coinsCollected;
  }

  private getScore(elapsedSeconds: number) {
    const coinScore = this.getRunCoinCount() * 140;
    const lifeBonus = this.lives * 250;
    const stageBonus = this.levelIndex * 350;
    const speedBonus = Math.max(0, 2600 - elapsedSeconds * 14);
    const deathPenalty = this.deathCount * 90;
    const finishBonus = this.status === 'won' ? 550 : this.status === 'level-complete' ? 220 : 0;

    return Math.max(0, coinScore + lifeBonus + stageBonus + speedBonus + finishBonus - deathPenalty);
  }

  private emitState(force = false) {
    const elapsedSeconds = this.getElapsedSeconds();
    const snapshot = {
      levelNumber: this.levelIndex + 1,
      totalLevels: LEVELS.length,
      levelName: this.level.name,
      levelTagline: this.level.tagline,
      coinsCollected: this.coinsCollected,
      totalCoins: this.totalCoins,
      runCoinsCollected: this.getRunCoinCount(),
      runCoinsTotal: TOTAL_RUN_COINS,
      status: this.status,
      playerX: this.player?.x ?? this.level.playerStart.x,
      lives: this.lives,
      deathCount: this.deathCount,
      score: this.getScore(elapsedSeconds),
      elapsedSeconds,
      checkpointLabel: this.checkpointLabel,
      canJump: this.canPlayerJump(),
      worldWidth: this.level.worldWidth,
      goalX: this.level.goalX,
    } satisfies GameSnapshot;

    if (!force && this.lastPublishedState) {
      const last = this.lastPublishedState;
      const stateChanged =
        snapshot.levelNumber !== last.levelNumber ||
        snapshot.totalLevels !== last.totalLevels ||
        snapshot.levelName !== last.levelName ||
        snapshot.levelTagline !== last.levelTagline ||
        snapshot.coinsCollected !== last.coinsCollected ||
        snapshot.totalCoins !== last.totalCoins ||
        snapshot.runCoinsCollected !== last.runCoinsCollected ||
        snapshot.runCoinsTotal !== last.runCoinsTotal ||
        snapshot.status !== last.status ||
        snapshot.lives !== last.lives ||
        snapshot.deathCount !== last.deathCount ||
        snapshot.score !== last.score ||
        snapshot.elapsedSeconds !== last.elapsedSeconds ||
        snapshot.checkpointLabel !== last.checkpointLabel ||
        snapshot.canJump !== last.canJump ||
        snapshot.worldWidth !== last.worldWidth ||
        snapshot.goalX !== last.goalX;
      const movedEnough = Math.abs(snapshot.playerX - last.playerX) >= STATE_POSITION_STEP;

      if (!stateChanged && !movedEnough) {
        return;
      }
    }

    this.lastPublishedState = snapshot;
    this.onStateChange(snapshot);
  }

  private canPlayerJump() {
    const body = this.player?.body as Phaser.Physics.Arcade.Body | undefined;
    if (!body) {
      return false;
    }

    return body.blocked.down || body.touching.down || this.time.now - this.lastGroundedAt <= COYOTE_TIME_MS;
  }
}
