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
const ATTACK_COOLDOWN_MS = 320;
const ATTACK_ACTIVE_MS = 120;
const ATTACK_RANGE_X = 56;
const ATTACK_RANGE_Y = 52;
const ATTACK_DASH_SPEED = 64;
const AIR_JUMP_COUNT = 1;
const DEFAULT_LIVES = 100;
const LANDING_SNAP_Y_TOLERANCE = 18;
const LANDING_SNAP_X_MARGIN = 14;
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
    attack: Phaser.Input.Keyboard.Key;
  };
  private platforms?: Phaser.Physics.Arcade.StaticGroup;
  private coins?: Phaser.Physics.Arcade.Group;
  private status: GameStatus = 'ready';
  private coinsCollected = 0;
  private totalCoins = 0;
  private lives = DEFAULT_LIVES;
  private deathCount = 0;
  private defeatedEnemies = 0;
  private runCoinsCollected = 0;
  private touchState: InputState = { left: false, right: false, jump: false, attack: false };
  private gamepadState: InputState = { left: false, right: false, jump: false, attack: false };
  private previousJumpRequested = false;
  private previousAttackRequested = false;
  private jumpQueuedUntil = 0;
  private airJumpsRemaining = AIR_JUMP_COUNT;
  private lastGroundedAt = 0;
  private lastPublishedState?: GameSnapshot;
  private movingPlatforms: MovingPlatformInstance[] = [];
  private attachedPlatform?: MovingPlatformInstance;
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
  private attackCooldownUntil = 0;
  private attackActiveUntil = 0;
  private attackEffect?: Phaser.GameObjects.Arc;
  private airJumpEffects: Phaser.GameObjects.GameObject[] = [];

  constructor(onStateChange: (state: GameSnapshot) => void) {
    super('GameScene');
    this.onStateChange = onStateChange;
  }

  init(data: SceneStartData = {}) {
    this.levelIndex = data.levelIndex ?? 0;
    this.level = LEVELS[this.levelIndex] ?? DEFAULT_LEVEL;
    this.lives = data.lives ?? DEFAULT_LIVES;
    this.deathCount = data.deathCount ?? 0;
    this.defeatedEnemies = data.defeatedEnemies ?? 0;
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
    const wantsAttack = this.touchState.attack || this.gamepadState.attack || !!this.keys?.attack.isDown;

    if (isGrounded) {
      this.lastGroundedAt = now;
      this.airJumpsRemaining = AIR_JUMP_COUNT;
    }

    if (wantsJump && !this.previousJumpRequested) {
      this.jumpQueuedUntil = now + JUMP_BUFFER_MS;
    }

    if (wantsAttack && !this.previousAttackRequested) {
      this.tryAttack();
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
      this.attachedPlatform = undefined;
      this.player.setVelocityY(JUMP_SPEED);
      this.jumpQueuedUntil = 0;
      this.lastGroundedAt = now - COYOTE_TIME_MS - delta;
      this.touchState.jump = false;
      this.gamepadState.jump = false;
    } else if (hasJumpBuffer && this.airJumpsRemaining > 0) {
      this.attachedPlatform = undefined;
      this.player.setVelocityY(JUMP_SPEED * 0.94);
      this.airJumpsRemaining -= 1;
      this.jumpQueuedUntil = 0;
      this.touchState.jump = false;
      this.gamepadState.jump = false;
      this.renderAirJumpEffect();
    } else if (!wantsJump && body.velocity.y < SHORT_HOP_SPEED) {
      this.player.setVelocityY(SHORT_HOP_SPEED);
    }

    this.previousJumpRequested = wantsJump;
    this.previousAttackRequested = wantsAttack;

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
      lives: DEFAULT_LIVES,
      deathCount: 0,
      defeatedEnemies: 0,
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
      defeatedEnemies: this.defeatedEnemies,
      runCoinsCollected: this.getRunCoinCount(),
      runStartedAt: this.runStartedAt,
      totalFrozenMs: this.getTotalFrozenMs(),
    } satisfies SceneStartData);
  }

  private bindSceneEvents() {
    this.events.on(Phaser.Scenes.Events.PRE_UPDATE, this.handleScenePreUpdate, this);

    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.events.off(Phaser.Scenes.Events.PRE_UPDATE, this.handleScenePreUpdate, this);
      this.attackEffect?.destroy();
      this.attackEffect = undefined;
      this.clearAirJumpEffects();
      this.attachedPlatform = undefined;
      this.touchState = { left: false, right: false, jump: false, attack: false };
      this.gamepadState = { left: false, right: false, jump: false, attack: false };
      this.previousJumpRequested = false;
      this.previousAttackRequested = false;
      this.jumpQueuedUntil = 0;
      this.airJumpsRemaining = AIR_JUMP_COUNT;
      this.lastGroundedAt = 0;
      this.lastPublishedState = undefined;
      this.freezeStartedAt = null;
    });
  }

  private handleScenePreUpdate() {
    if (this.status !== 'running') {
      return;
    }

    this.updateMovingPlatforms();
  }

  private createControls() {
    this.cursors = this.input.keyboard?.createCursorKeys();

    if (this.input.keyboard) {
      this.keys = {
        left: this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.A),
        right: this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.D),
        jump: this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.W),
        attack: this.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.J),
      };
    }
  }

  private resetLevelState() {
    this.status = 'ready';
    this.coinsCollected = 0;
    this.totalCoins = this.level.coins.length;
    this.touchState = { left: false, right: false, jump: false, attack: false };
    this.gamepadState = { left: false, right: false, jump: false, attack: false };
    this.previousJumpRequested = false;
    this.previousAttackRequested = false;
    this.jumpQueuedUntil = 0;
    this.airJumpsRemaining = AIR_JUMP_COUNT;
    this.lastGroundedAt = this.time.now;
    this.lastPublishedState = undefined;
    this.movingPlatforms = [];
    this.attachedPlatform = undefined;
    this.enemies = [];
    this.checkpoints = [];
    this.respawnPoint = {
      x: this.level.playerStart.x,
      y: this.level.playerStart.y,
      label: '起点',
    };
    this.checkpointLabel = '起点';
    this.damageCooldownUntil = 0;
    this.attackCooldownUntil = 0;
    this.attackActiveUntil = 0;
    this.attackEffect?.destroy();
    this.attackEffect = undefined;
    this.clearAirJumpEffects();
    this.freezeStartedAt = null;
  }

  private updateMovingPlatforms() {
    if (this.movingPlatforms.length === 0) {
      return;
    }

    const playerBody = this.player?.body as Phaser.Physics.Arcade.Body | undefined;
    let nextAttachedPlatform: MovingPlatformInstance | undefined;

    for (const platform of this.movingPlatforms) {
      const previousTop = platform.body.top;
      const previousLeft = platform.body.left;
      const previousRight = platform.body.right;
      const playerStandingGap = playerBody ? playerBody.bottom - previousTop : 0;
      const wasStandingOnPlatform =
        !!playerBody &&
        this.status === 'running' &&
        (this.attachedPlatform === platform || playerBody.velocity.y >= -40) &&
        playerStandingGap >= -10 &&
        playerStandingGap <= 16 &&
        playerBody.right > previousLeft + 10 &&
        playerBody.left < previousRight - 10;

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
      const nextX = Math.round(platform.originX + (movement.axis === 'x' ? offset : 0));
      const nextY = Math.round(platform.originY + (movement.axis === 'y' ? offset : 0));

      platform.block.setPosition(nextX, nextY);
      platform.body.updateFromGameObject();
      platform.deltaX = nextX - platform.previousX;
      platform.deltaY = nextY - platform.previousY;

      if (!playerBody || this.status !== 'running') {
        continue;
      }

      if (wasStandingOnPlatform) {
        nextAttachedPlatform = platform;
        playerBody.x += platform.deltaX;
        playerBody.y += platform.deltaY;

        const snappedTop = platform.body.top - playerBody.height;
        const correctionY = snappedTop - playerBody.y;
        if (Math.abs(correctionY) <= 12) {
          playerBody.y += correctionY;
        }

        this.player?.setPosition(playerBody.x + playerBody.halfWidth, playerBody.y + playerBody.halfHeight);
        continue;
      }

      const canSnapLanding =
        this.attachedPlatform === undefined &&
        playerBody.velocity.y >= 0 &&
        playerBody.bottom <= platform.body.top + LANDING_SNAP_Y_TOLERANCE &&
        playerBody.bottom >= platform.body.top - LANDING_SNAP_Y_TOLERANCE &&
        playerBody.right > platform.body.left + LANDING_SNAP_X_MARGIN &&
        playerBody.left < platform.body.right - LANDING_SNAP_X_MARGIN;

      if (canSnapLanding) {
        nextAttachedPlatform = platform;
        playerBody.y = platform.body.top - playerBody.height;
        playerBody.velocity.y = 0;
        this.lastGroundedAt = this.time.now;
        this.airJumpsRemaining = AIR_JUMP_COUNT;
        this.player?.setPosition(playerBody.x + playerBody.halfWidth, playerBody.y + playerBody.halfHeight);
      }
    }

    this.attachedPlatform = nextAttachedPlatform;
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
      this.defeatEnemy(enemyObject);
      this.player.setVelocityY(JUMP_SPEED * 0.58);
      this.emitState(true);
      return;
    }

    this.loseLife();
  }

  private tryAttack() {
    if (!this.player || this.status !== 'running' || this.time.now < this.attackCooldownUntil) {
      return;
    }

    const now = this.time.now;
    this.attackCooldownUntil = now + ATTACK_COOLDOWN_MS;
    this.attackActiveUntil = now + ATTACK_ACTIVE_MS;

    const playerBody = this.player.body as Phaser.Physics.Arcade.Body;
    const facing = this.player.flipX ? -1 : 1;
    this.player.setVelocityX(playerBody.velocity.x + facing * ATTACK_DASH_SPEED);
    this.player.setScale(1.18, 0.9);
    const attackCenterX = this.player.x + facing * ATTACK_RANGE_X * 0.5;
    const attackBounds = new Phaser.Geom.Rectangle(
      attackCenterX - ATTACK_RANGE_X / 2,
      this.player.y - ATTACK_RANGE_Y / 2,
      ATTACK_RANGE_X,
      ATTACK_RANGE_Y
    );

    let hitEnemy = false;
    for (const enemy of this.enemies) {
      if (!enemy.sprite.active || !enemy.sprite.body) {
        continue;
      }

      const enemyBounds = enemy.sprite.getBounds();
      if (!Phaser.Geom.Intersects.RectangleToRectangle(attackBounds, enemyBounds)) {
        continue;
      }

      this.defeatEnemy(enemy.sprite);
      hitEnemy = true;
    }

    this.renderAttackEffect(facing, hitEnemy);
    this.player.setTint(hitEnemy ? 0xffef8f : 0xffb27d);
    this.time.delayedCall(ATTACK_ACTIVE_MS, () => {
      if (this.player && this.status !== 'lost') {
        this.player.clearTint();
        this.player.setScale(1, 1);
      }
      this.emitState(true);
    });

    this.emitState(true);
  }

  private renderAttackEffect(facing: 1 | -1, hitEnemy: boolean) {
    if (!this.player) {
      return;
    }

    this.attackEffect?.destroy();
    const effectColor = hitEnemy ? 0xffd84f : 0xe55433;
    const startAngle = facing > 0 ? 312 : 48;
    const endAngle = facing > 0 ? 48 : 132;
    const effect = this.add.arc(
      this.player.x + facing * 26,
      this.player.y - 4,
      30,
      startAngle,
      endAngle,
      false,
      effectColor,
      0.28
    );
    effect.setStrokeStyle(8, 0xfff8dd, hitEnemy ? 0.95 : 0.7);
    effect.setDepth(18);
    this.attackEffect = effect;

    this.tweens.add({
      targets: effect,
      alpha: 0,
      scaleX: 1.28,
      scaleY: 0.82,
      x: effect.x + facing * 14,
      duration: ATTACK_ACTIVE_MS,
      ease: 'Cubic.Out',
      onComplete: () => {
        if (this.attackEffect === effect) {
          this.attackEffect = undefined;
        }
        effect.destroy();
      },
    });
  }

  private renderAirJumpEffect() {
    if (!this.player) {
      return;
    }

    this.clearAirJumpEffects();

    const ring = this.add.circle(this.player.x, this.player.y + 18, 12, 0xffd84f, 0.22);
    ring.setStrokeStyle(4, 0xfff8dd, 0.9);
    ring.setDepth(17);

    const leftTrail = this.add.triangle(this.player.x - 12, this.player.y + 12, 0, 16, 10, 0, 20, 16, 0xe55433, 0.72);
    leftTrail.setDepth(16);
    leftTrail.setAngle(-18);

    const rightTrail = this.add.triangle(this.player.x + 12, this.player.y + 12, 0, 16, 10, 0, 20, 16, 0xffd84f, 0.72);
    rightTrail.setDepth(16);
    rightTrail.setAngle(18);

    this.airJumpEffects = [ring, leftTrail, rightTrail];

    this.tweens.add({
      targets: ring,
      scaleX: 2.1,
      scaleY: 1.45,
      alpha: 0,
      y: ring.y + 10,
      duration: 220,
      ease: 'Quad.Out',
      onComplete: () => ring.destroy(),
    });

    this.tweens.add({
      targets: [leftTrail, rightTrail],
      y: '+=18',
      scaleX: 0.5,
      scaleY: 1.8,
      alpha: 0,
      duration: 220,
      ease: 'Cubic.Out',
      onComplete: () => {
        leftTrail.destroy();
        rightTrail.destroy();
      },
    });

    this.time.delayedCall(240, () => {
      this.airJumpEffects = this.airJumpEffects.filter((effect) => effect.active);
    });
  }

  private clearAirJumpEffects() {
    this.airJumpEffects.forEach((effect) => effect.destroy());
    this.airJumpEffects = [];
  }

  private defeatEnemy(enemyObject: Phaser.GameObjects.GameObject) {
    enemyObject.destroy();
    this.defeatedEnemies += 1;
  }

  private activateCheckpoint(label: string) {
    const nextCheckpoint = this.checkpoints.find((checkpoint) => checkpoint.definition.label === label);
    if (!nextCheckpoint || nextCheckpoint.active) {
      return;
    }

    this.checkpoints.forEach((checkpoint) => {
      checkpoint.active = checkpoint === nextCheckpoint;
      checkpoint.beacon.setFillStyle(checkpoint.active ? 0x22c55e : 0x38bdf8, 0.96);
      checkpoint.beacon.setStrokeStyle(2, checkpoint.active ? 0xdcfce7 : 0xe0f2fe);
      checkpoint.halo.setFillStyle(checkpoint.active ? 0x22c55e : 0x38bdf8, checkpoint.active ? 0.2 : 0.14);
      checkpoint.halo.setStrokeStyle(2, checkpoint.active ? 0xbbf7d0 : 0x7dd3fc, checkpoint.active ? 0.56 : 0.42);
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

    this.attachedPlatform = undefined;
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

    this.attachedPlatform = undefined;
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
    this.airJumpsRemaining = AIR_JUMP_COUNT;
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
    const enemyScore = this.defeatedEnemies * 180;
    const lifeBonus = this.lives * 250;
    const stageBonus = this.levelIndex * 350;
    const speedBonus = Math.max(0, 2600 - elapsedSeconds * 14);
    const deathPenalty = this.deathCount * 90;
    const finishBonus = this.status === 'won' ? 550 : this.status === 'level-complete' ? 220 : 0;

    return Math.max(0, coinScore + enemyScore + lifeBonus + stageBonus + speedBonus + finishBonus - deathPenalty);
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
      defeatedEnemies: this.defeatedEnemies,
      elapsedSeconds,
      checkpointLabel: this.checkpointLabel,
      canJump: this.canPlayerJump(),
      canAttack: this.time.now >= this.attackCooldownUntil,
      isAttacking: this.time.now < this.attackActiveUntil,
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
        snapshot.defeatedEnemies !== last.defeatedEnemies ||
        snapshot.elapsedSeconds !== last.elapsedSeconds ||
        snapshot.checkpointLabel !== last.checkpointLabel ||
        snapshot.canJump !== last.canJump ||
        snapshot.canAttack !== last.canAttack ||
        snapshot.isAttacking !== last.isAttacking ||
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
