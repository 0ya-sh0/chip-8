import { Component, DestroyRef, HostListener, inject, OnInit, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { GameViewport } from "./components/game-viewport/game-viewport";
import { RomPicker } from "./components/rom-picker/rom-picker";

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, GameViewport, RomPicker],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit {
  protected readonly title = signal('frontend');
  public gameState: Array<Array<boolean>> = [];
  selectedRom: any = signal(null);
  
  storedKeyBindings: { "1": string; "2": string; "3": string; C: string; "4": string; "5": string; "6": string; D: string; "7": string; "8": string; "9": string; E: string; A: string; "0": string; B: string; F: string; } = {
    1: '',
    2: '',
    3: '',
    C: '',
    4: '',
    5: '',
    6: '',
    D: '',
    7: '',
    8: '',
    9: '',
    E: '',
    A: '',
    0: '',
    B: '',
    F: ''
  };
  returnChip8Bindings: { [k: string]: string; } = {};

  async ngOnInit() {
    this.loadKeyBindings();
  }

  loadKeyBindings() {
    //This is how bindings will be stored as they will need to be against the chip 8 config. 
    this.storedKeyBindings = {
      "1": "Digit1",
      "2": "Digit2",
      "3": "Digit3",
      "C": "Digit4",
      "4": "KeyQ",
      "5": "KeyW",
      "6": "KeyE",
      "D": "KeyR",
      "7": "KeyA",
      "8": "KeyS",
      "9": "KeyD",
      "E": "KeyF",
      "A": "KeyZ",
      "0": "KeyX",
      "B": "KeyC",
      "F": "KeyV",
    };

    this.returnChip8Bindings = Object.fromEntries(
      Object.entries(this.storedKeyBindings).map(([key, value]) => [value, key])
    );
  }

  openGame(rom: any) {
    // Implementation for opening the game with the selected ROM
    console.log('Selected ROM:', rom);

    this.selectedRom.set(rom);
  }

}
