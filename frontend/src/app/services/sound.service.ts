import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class SoundService {
  private audioCtx!: AudioContext;

  constructor() {
    // Context is created but might start suspended by the browser
    this.audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
  }

  // Call this ONCE during your game's "Start" or "Splash" screen click
  unlockAudio(): void {
    if (this.audioCtx.state === 'suspended') {
      this.audioCtx.resume();
    }
  }

  // Call this automatically anywhere in your game loop or logic
  playBitSound(frequency = 600, duration = 0.1): void {
    if (this.audioCtx.state === 'suspended') return;

    const osc = this.audioCtx.createOscillator();
    const gain = this.audioCtx.createGain();

    osc.type = 'square'; 
    osc.frequency.setValueAtTime(frequency, this.audioCtx.currentTime);

    osc.connect(gain);
    gain.connect(this.audioCtx.destination);

    osc.start();
    // Smooth fade out to prevent speaker popping
    gain.gain.exponentialRampToValueAtTime(0.00001, this.audioCtx.currentTime + duration);
    osc.stop(this.audioCtx.currentTime + duration);
  }
}
