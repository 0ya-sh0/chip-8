import { ChangeDetectorRef, Component, DestroyRef, HostListener, inject, OnInit, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { WebSocketService } from './services/websocket.service';
import { firstValueFrom, Subscription } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { SoundService } from './services/sound.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit {
  private webSocketService_ = inject(WebSocketService);
  private destroyRef = inject(DestroyRef);
  private cdr = inject(ChangeDetectorRef);
  private soundService_ = inject (SoundService);
  protected readonly title = signal('frontend');
  public gameState: Array<Array<boolean>> = [];
  
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

  private listenToWebSocket(){
    this.webSocketService_.messages$
      .pipe(
        // Automatically unsubscribes when this component is destroyed
        takeUntilDestroyed(this.destroyRef) 
      )
      .subscribe({
        next: (message: any) => {
          this.handleIncomingWSMessage(message);
        },
        error: (err) => {
          console.error('WebSocket stream encountered an error:', err);
        },
        complete: () => {
          console.log('WebSocket stream has completed.');
        }
      });
  }

  private handleIncomingWSMessage(message: any): void {
    switch(message.type) {
      case 'display.draw':
        this.gameState = message?.data;
        this.cdr.detectChanges();
        break;
      case 'sound.play':
        if(!this.soundService_.isASoundPlaying)
          this.soundService_.playBitSound(500, 10);
        break;
      case 'sound.stop':
        if(this.soundService_.isASoundPlaying)
          this.soundService_.stopSound(); 
        break;
    }
  }

  async startGame() {
    try {
      const wsConnection = await this.webSocketService_.connect();
    } catch (err) {
      console.error("Error connecting to WebSocket", err);
    }
    this.soundService_.unlockAudio();
    this.listenToWebSocket();
  }

  @HostListener('window:keydown', ['$event'])
  handleKeyboardDown(event: KeyboardEvent): void {
    this.sendKeypress(false, this.returnChip8Bindings[event.code]);
  }

  @HostListener('window:keyup', ['$event'])
  handleKeyboardUp(event: KeyboardEvent): void {
    this.sendKeypress(true, this.returnChip8Bindings[event.code]);
  }

  sendKeypress(up: boolean, key: string) {
    this.webSocketService_.send({
      type: up ? "key.up" : "key.down",
      data: key
    })
  }
}
