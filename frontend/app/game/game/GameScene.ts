import * as Phaser from 'phaser';
import type { GameSnapshot } from './state';
import {
  COIN_LAYOUT,
  FLOOR_Y,
  GAME_HEIGHT,
  GOAL_X,
  PLATFORM_LAYOUT,
  PLAYER_START,
  WORLD_WIDTH,
} from './level';

const PLAYER_SIZE = { width: 34, height: 48 };
const MOVE_SPEED = 220;
const JUMP_SPEED = -420;

export class GameScene extends Phaser.Scene {
  private readonly onStateChange: (state: GameSnapshot) => void;
  private player?: Phaser.Physics.Arcade.Sprite;
  private cursors?: Phaser.Types.Input.Keyboard.CursorKeys;
  private keys?: {
    left: Phaser.Input.Keyboard.Key;
    right: Phaser.Input.Keyboard.Key;
    jump: Phaser.Input.Keyboard.Key;
  };
  private platforms?: Phaser.Physics.Arcade.StaticGroup;
  private coins?: Phaser.Physics.Arcade.Group;
  private goal?: Phaser.GameObjects.Rectangle;
  private status: GameSnapshot['status'] = 'ready';
  private coinsCollected = 0;
  private totalCoins = COIN_LAYOUT.length;
  private lives = 3;
  private touchState = { left: false, right: false, jump: false };
  private gamepadState = { left: false, right: false, jump: false };

  constructor(onStateChange: (state: GameSnapshot) => void) {
    super('GameScene');
    this.onStateChange = onStateChange;
  }

  create() {
    this.cameras.main.setBounds(0, 0, WORLD_WIDTH, GAME_HEIGHT);
    this.physics.world.setBounds(0, 0, WORLD_WIDTH, GAME_HEIGHT + 120);

    this.drawBackdrop();
    this.createPlatforms();
    this.createGoal();
    this.createCoins();
    this.createPlayer();
    this.createControls();
    this.bindSceneEvents();

    this.status = 'running';
    this.emitState();
  }

  update() {
    if (!this.player?.body || this.status === 'paused' || this.status === 'won' || this.status === 'lost') {
      return;
    }

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

    if (moveLeft === moveRight) {
      this.player.setVelocityX(0);
    } else if (moveLeft) {
      this.player.setVelocityX(-MOVE_SPEED);
    } else {
      this.player.setVelocityX(MOVE_SPEED);
    }

    if (wantsJump && this.player.body.blocked.down) {
      this.player.setVelocityY(JUMP_SPEED);
      this.touchState.jump = false;
      this.gamepadState.jump = false;
    }

    if (this.player.y > GAME_HEIGHT + 80) {
      this.loseLife();
    }

    const reachedGoal = this.player.x >= GOAL_X - 30;
    if (reachedGoal && this.status === 'running') {
      this.status = 'won';
      this.player.setVelocity(0, 0);
      this.player.body.enable = false;
    }

    this.emitState();
  }

  setTouchInput(nextState: Partial<typeof this.touchState>) {
    this.touchState = { ...this.touchState, ...nextState };
  }

  setGamepadInput(nextState: Partial<typeof this.gamepadState>) {
    this.gamepadState = { ...this.gamepadState, ...nextState };
  }

  pauseGame() {
    if (this.status !== 'running') {
      return;
    }

    this.status = 'paused';
    this.physics.world.pause();
    this.emitState();
  }

  resumeGame() {
    if (this.status !== 'paused') {
      return;
    }

    this.status = 'running';
    this.physics.world.resume();
    this.emitState();
  }

  restartGame() {
    this.scene.restart();
  }

  private bindSceneEvents() {
    this.events.on(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.touchState = { left: false, right: false, jump: false };
      this.gamepadState = { left: false, right: false, jump: false };
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

  private createPlatforms() {
    const groundHeight = GAME_HEIGHT - FLOOR_Y;
    this.platforms = this.physics.add.staticGroup();

    const floor = this.add.rectangle(WORLD_WIDTH / 2, FLOOR_Y + groundHeight / 2, WORLD_WIDTH, groundHeight, 0x3f6212);
    this.physics.add.existing(floor, true);
    this.platforms.add(floor as unknown as Phaser.Physics.Arcade.Image);

    for (const platform of PLATFORM_LAYOUT) {
      const block = this.add.rectangle(
        platform.x + platform.width / 2,
        platform.y + platform.height / 2,
        platform.width,
        platform.height,
        0xb45309
      );
      block.setStrokeStyle(2, 0xf59e0b);
      this.physics.add.existing(block, true);
      this.platforms.add(block as unknown as Phaser.Physics.Arcade.Image);
    }
  }

  private createCoins() {
    this.coins = this.physics.add.group({ allowGravity: false, immovable: true });

    for (const coinPosition of COIN_LAYOUT) {
      const coin = this.add.circle(coinPosition.x, coinPosition.y, 10, 0xfacc15);
      coin.setStrokeStyle(3, 0xfde68a);
      this.physics.add.existing(coin);
      const coinBody = coin.body as Phaser.Physics.Arcade.Body;
      coinBody.setAllowGravity(false);
      coinBody.setCircle(10);
      this.coins.add(coin as unknown as Phaser.Physics.Arcade.Sprite);
    }
  }

  private createGoal() {
    this.add.rectangle(GOAL_X, FLOOR_Y - 90, 12, 190, 0xe5e7eb).setOrigin(0, 0);
    this.goal = this.add.rectangle(GOAL_X + 26, FLOOR_Y - 90, 56, 36, 0xef4444).setOrigin(0, 0);
  }

  private createPlayer() {
    const textureKey = 'player-block';
    if (!this.textures.exists(textureKey)) {
      const graphics = this.add.graphics();
      graphics.fillStyle(0xf97316);
      graphics.fillRoundedRect(0, 0, PLAYER_SIZE.width, PLAYER_SIZE.height, 8);
      graphics.fillStyle(0xfef3c7);
      graphics.fillRoundedRect(7, 8, PLAYER_SIZE.width - 14, PLAYER_SIZE.height - 20, 6);
      graphics.generateTexture(textureKey, PLAYER_SIZE.width, PLAYER_SIZE.height);
      graphics.destroy();
    }

    this.player = this.physics.add.sprite(PLAYER_START.x, PLAYER_START.y, textureKey);
    this.player.setCollideWorldBounds(true);
    this.player.setBounce(0.03);
    this.player.body?.setSize(PLAYER_SIZE.width, PLAYER_SIZE.height);

    if (this.platforms) {
      this.physics.add.collider(this.player, this.platforms);
    }

    if (this.coins) {
      this.physics.add.overlap(this.player, this.coins, (_, coin) => {
        coin.destroy();
        this.coinsCollected += 1;
        this.emitState();
      });
    }

    this.cameras.main.startFollow(this.player, true, 0.1, 0.1, -160, 40);
    this.cameras.main.setDeadzone(180, 140);
  }

  private drawBackdrop() {
    this.add.rectangle(WORLD_WIDTH / 2, GAME_HEIGHT / 2, WORLD_WIDTH, GAME_HEIGHT, 0x60a5fa);

    for (const cloudX of [180, 540, 960, 1320, 1760, 2140]) {
      this.add.ellipse(cloudX, 120, 92, 44, 0xffffff, 0.85);
      this.add.ellipse(cloudX + 40, 110, 76, 36, 0xffffff, 0.85);
      this.add.ellipse(cloudX - 36, 110, 70, 34, 0xffffff, 0.85);
    }

    for (const hill of [320, 860, 1540, 2050]) {
      this.add.triangle(hill, FLOOR_Y, 0, 160, 160, 0, 320, 160, 0x34d399).setOrigin(0.5, 1);
      this.add.triangle(hill + 190, FLOOR_Y, 0, 120, 120, 0, 240, 120, 0x10b981).setOrigin(0.5, 1);
    }
  }

  private loseLife() {
    this.lives -= 1;
    if (this.lives <= 0) {
      this.status = 'lost';
      this.physics.world.pause();
      this.player?.setVelocity(0, 0);
      this.player?.setPosition(PLAYER_START.x, PLAYER_START.y);
      this.emitState();
      return;
    }

    this.player?.setVelocity(0, 0);
    this.player?.setPosition(PLAYER_START.x, PLAYER_START.y);
    this.emitState();
  }

  private emitState() {
    this.onStateChange({
      coinsCollected: this.coinsCollected,
      totalCoins: this.totalCoins,
      status: this.status,
      playerX: this.player?.x ?? PLAYER_START.x,
      lives: this.lives,
      canJump: Boolean(this.player?.body?.blocked.down),
      worldWidth: WORLD_WIDTH,
    });
  }
}
