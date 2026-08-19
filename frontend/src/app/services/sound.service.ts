import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class SoundService {
  private audioCtx!: AudioContext;

  private currentOscillator: OscillatorNode | null = null;
  private currentGain: GainNode | null = null;

  public isASoundPlaying = false;

  constructor() {
    this.audioCtx = new (window.AudioContext ||
      (window as any).webkitAudioContext)();
  }

  unlockAudio(): void {
    if (this.audioCtx.state === 'suspended') {
      this.audioCtx.resume();
    }
  }

  playBitSound(frequency = 600, duration = 0.1): void {
    if (this.audioCtx.state === 'suspended') return;

    // Stop any currently playing sound
    this.stopSound();

    const osc = this.audioCtx.createOscillator();
    const gain = this.audioCtx.createGain();

    this.currentOscillator = osc;
    this.currentGain = gain;
    this.isASoundPlaying = true;

    osc.type = 'square';
    osc.frequency.setValueAtTime(
      frequency,
      this.audioCtx.currentTime
    );

    osc.connect(gain);
    gain.connect(this.audioCtx.destination);

    const now = this.audioCtx.currentTime;

    gain.gain.setValueAtTime(1, now);
    gain.gain.exponentialRampToValueAtTime(
      0.00001,
      now + duration
    );

    osc.start(now);
    osc.stop(now + duration);

    osc.onended = () => {
      if (this.currentOscillator === osc) {
        this.currentOscillator = null;
        this.currentGain = null;
        this.isASoundPlaying = false;
      }
    };
  }

  stopSound(): void {
    if (this.currentOscillator) {
      try {
        // Fade out quickly to avoid a speaker click/pop
        const now = this.audioCtx.currentTime;

        if (this.currentGain) {
          this.currentGain.gain.cancelScheduledValues(now);
          this.currentGain.gain.setValueAtTime(
            Math.max(this.currentGain.gain.value, 0.00001),
            now
          );
          this.currentGain.gain.exponentialRampToValueAtTime(
            0.00001,
            now + 0.01
          );
        }

        this.currentOscillator.stop(now + 0.01);
      } catch {
        // Oscillator may already have stopped
      }

      this.currentOscillator = null;
      this.currentGain = null;
    }

    this.isASoundPlaying = false;
  }
}
