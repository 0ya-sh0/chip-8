import { Injectable, OnDestroy } from '@angular/core';

/**
 * SoundService drives the CHIP-8 buzzer.
 *
 * The server sends edges, not levels: exactly one `sound.play` when the buzzer
 * starts and exactly one `sound.stop` when it ends, with the duration decided by
 * the emulated sound timer. So the tone here must be *sustained* until stopped,
 * not a fixed-length blip - a one-shot beep would either cut a long buzz short
 * or run past a short one.
 */
@Injectable({
  providedIn: 'root',
})
export class SoundService implements OnDestroy {
  /** A4, the tone most CHIP-8 emulators use for the buzzer. */
  private static readonly FREQUENCY = 440;
  /** Kept low: a square wave at full scale is unpleasant. */
  private static readonly VOLUME = 0.08;
  /** Short ramps avoid the click a hard start or stop produces. */
  private static readonly RAMP = 0.008;

  private audioCtx: AudioContext | null = null;
  private oscillator: OscillatorNode | null = null;
  private gain: GainNode | null = null;

  get isASoundPlaying(): boolean {
    return this.oscillator !== null;
  }

  /**
   * Browsers start an AudioContext suspended until the user interacts with the
   * page, so this must be called from a user gesture (selecting a ROM).
   */
  unlockAudio(): void {
    const ctx = this.context();
    if (ctx && ctx.state === 'suspended') {
      void ctx.resume();
    }
  }

  /** Starts the buzzer. Calling it while already playing does nothing. */
  startTone(frequency = SoundService.FREQUENCY): void {
    const ctx = this.context();
    if (!ctx || this.oscillator) {
      return;
    }

    const osc = ctx.createOscillator();
    const gain = ctx.createGain();

    osc.type = 'square';
    osc.frequency.setValueAtTime(frequency, ctx.currentTime);
    osc.connect(gain);
    gain.connect(ctx.destination);

    const now = ctx.currentTime;
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(SoundService.VOLUME, now + SoundService.RAMP);

    // No stop time: the tone runs until stopSound is called.
    osc.start(now);

    this.oscillator = osc;
    this.gain = gain;
  }

  /** Stops the buzzer. Safe to call when nothing is playing. */
  stopSound(): void {
    const ctx = this.audioCtx;
    const osc = this.oscillator;
    const gain = this.gain;
    if (!ctx || !osc) {
      return;
    }

    this.oscillator = null;
    this.gain = null;

    const now = ctx.currentTime;
    try {
      if (gain) {
        gain.gain.cancelScheduledValues(now);
        gain.gain.setValueAtTime(gain.gain.value, now);
        gain.gain.linearRampToValueAtTime(0, now + SoundService.RAMP);
      }
      osc.stop(now + SoundService.RAMP);
    } catch {
      // The oscillator may already have stopped; nothing to do.
    }
    osc.onended = () => {
      osc.disconnect();
      gain?.disconnect();
    };
  }

  /**
   * The AudioContext is created lazily. Constructing one at service
   * instantiation logs a warning in most browsers, because that happens before
   * any user gesture has occurred.
   */
  private context(): AudioContext | null {
    if (this.audioCtx) {
      return this.audioCtx;
    }
    const Ctor = window.AudioContext ?? (window as any).webkitAudioContext;
    if (!Ctor) {
      console.warn('sound: Web Audio is unavailable; the buzzer will be silent');
      return null;
    }
    this.audioCtx = new Ctor();
    return this.audioCtx;
  }

  ngOnDestroy(): void {
    this.stopSound();
    void this.audioCtx?.close();
    this.audioCtx = null;
  }
}
