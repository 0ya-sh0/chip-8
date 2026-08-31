import {
  AfterViewInit,
  Component,
  DestroyRef,
  ElementRef,
  HostListener,
  inject,
  input,
  OnDestroy,
  OnInit,
  signal,
  viewChild,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { SoundService } from '../../services/sound.service';
import { WebSocketService, ControlMessage } from '../../services/websocket.service';
import { environment } from '../../environments/environment';
import { RomDetails } from '../../common/models';

/** Gameboy-ish palette, matching $pixel-on / $pixel-off in variables.scss. */
const PIXEL_OFF: [number, number, number] = [0x8b, 0xac, 0x0f];
const PIXEL_ON: [number, number, number] = [0x30, 0x62, 0x30];

@Component({
  selector: 'app-game-viewport',
  imports: [],
  templateUrl: './game-viewport.html',
  styleUrl: './game-viewport.scss',
})
export class GameViewport implements OnInit, AfterViewInit, OnDestroy {
  rom = input<RomDetails | null>(null);
  returnChip8Bindings = input<{ [k: string]: string }>({});

  private webSocketService_ = inject(WebSocketService);
  private destroyRef = inject(DestroyRef);
  private soundService_ = inject(SoundService);

  private canvasRef = viewChild.required<ElementRef<HTMLCanvasElement>>('screen');
  private ctx: CanvasRenderingContext2D | null = null;
  private image: ImageData | null = null;

  /** Set from the server's display.init message rather than hardcoded. */
  width = signal(64);
  height = signal(32);
  error = signal<string | null>(null);

  ngOnInit() {
    // Frames can start arriving before the canvas exists, so subscribe here and
    // let the render path no-op until the context is ready.
    this.listenToServer();
  }

  ngAfterViewInit() {
    this.initCanvas();
    void this.startGame();
  }

  ngOnDestroy() {
    this.webSocketService_.disconnect();
  }

  private initCanvas() {
    const canvas = this.canvasRef().nativeElement;
    canvas.width = this.width();
    canvas.height = this.height();

    // alpha:false lets the compositor skip blending an opaque surface.
    const ctx = canvas.getContext('2d', { alpha: false });
    if (!ctx) {
      this.error.set('Canvas is unavailable in this browser.');
      return;
    }
    this.ctx = ctx;
    this.image = ctx.createImageData(this.width(), this.height());

    // Start fully opaque; only RGB is rewritten per frame.
    const px = this.image.data;
    for (let i = 3; i < px.length; i += 4) {
      px[i] = 255;
    }
    this.paintBlank();
  }

  private async startGame() {
    const rom = this.rom();
    // ROM ids start at 0, so this must be a null check. Testing truthiness here
    // made the very first ROM in the catalog impossible to load.
    if (!rom || rom.id === null || rom.id === undefined) {
      this.error.set('No ROM selected.');
      return;
    }

    try {
      await this.webSocketService_.connect(environment.connectToGame(rom.id));
      this.error.set(null);
      // Browsers keep AudioContext suspended until a user gesture; selecting a
      // ROM is that gesture.
      this.soundService_.unlockAudio();
    } catch (err) {
      console.error('game-viewport: failed to start', err);
      this.error.set('Could not connect to the emulator server.');
    }
  }

  private listenToServer() {
    this.webSocketService_.frames$
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((frame) => this.renderFrame(frame));

    this.webSocketService_.control$
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((message) => this.handleControl(message));
  }

  private handleControl(message: ControlMessage): void {
    switch (message.type) {
      case 'display.init':
        // The server tells us the geometry so the client never has to hardcode
        // it to unpack the bitmap.
        if (message.width && message.height) {
          this.width.set(message.width);
          this.height.set(message.height);
          this.initCanvas();
        }
        break;
      case 'sound.play':
        this.soundService_.startTone();
        break;
      case 'sound.stop':
        this.soundService_.stopSound();
        break;
      case 'error':
        this.error.set(message.data ?? 'Emulator error.');
        break;
    }
  }

  /**
   * renderFrame unpacks a 1bpp bitmap into the canvas.
   *
   * Layout: scanline order, MSB first within each byte, so pixel (x, y) is bit
   * `7 - (x % 8)` of byte `(y * width + x) / 8`. Walking the bytes in order
   * means the pixel index only ever increments, so no per-pixel arithmetic is
   * needed.
   */
  private renderFrame(frame: Uint8Array): void {
    const ctx = this.ctx;
    const image = this.image;
    if (!ctx || !image) {
      return;
    }

    const expected = (this.width() * this.height()) / 8;
    if (frame.length !== expected) {
      console.warn(`game-viewport: frame is ${frame.length} bytes, expected ${expected}`);
      return;
    }

    const px = image.data;
    let p = 0;
    for (let i = 0; i < frame.length; i++) {
      const byte = frame[i];
      for (let bit = 7; bit >= 0; bit--) {
        const colour = (byte >> bit) & 1 ? PIXEL_ON : PIXEL_OFF;
        px[p] = colour[0];
        px[p + 1] = colour[1];
        px[p + 2] = colour[2];
        p += 4;
      }
    }
    ctx.putImageData(image, 0, 0);
  }

  private paintBlank(): void {
    if (!this.ctx || !this.image) {
      return;
    }
    const px = this.image.data;
    for (let p = 0; p < px.length; p += 4) {
      px[p] = PIXEL_OFF[0];
      px[p + 1] = PIXEL_OFF[1];
      px[p + 2] = PIXEL_OFF[2];
    }
    this.ctx.putImageData(this.image, 0, 0);
  }

  @HostListener('window:keydown', ['$event'])
  handleKeyboardDown(event: KeyboardEvent): void {
    // The OS repeats keydown ~30x/sec while a key is held. The server ignores
    // repeats, but there is no reason to send them.
    if (event.repeat) {
      return;
    }
    this.sendKeypress('key.down', event);
  }

  @HostListener('window:keyup', ['$event'])
  handleKeyboardUp(event: KeyboardEvent): void {
    this.sendKeypress('key.up', event);
  }

  /**
   * A key held when the window loses focus never receives its keyup, so without
   * this the emulator would see it as stuck down forever.
   */
  @HostListener('window:blur')
  handleBlur(): void {
    this.webSocketService_.send({ type: 'focus.lost' });
  }

  private sendKeypress(type: 'key.down' | 'key.up', event: KeyboardEvent): void {
    const key = this.returnChip8Bindings()[event.code];
    if (!key) {
      return;
    }
    // Stop the browser acting on keys the game has claimed (e.g. quick-find).
    event.preventDefault();
    this.webSocketService_.send({ type, data: key });
  }
}
