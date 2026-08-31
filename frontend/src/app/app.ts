import { Component, OnInit, signal } from '@angular/core';
import { GameViewport } from "./components/game-viewport/game-viewport";
import { RomPicker } from "./components/rom-picker/rom-picker";
import { KeyBindings, RomDetails } from './common/models';

@Component({
  selector: 'app-root',
  imports: [GameViewport, RomPicker],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit {
  protected readonly title = signal('frontend');
  selectedRom = signal<RomDetails | null>(null);
  
  storedKeyBindings:KeyBindings = {
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

  openGame(rom: RomDetails) {
    this.selectedRom.set(rom);
  }

}
