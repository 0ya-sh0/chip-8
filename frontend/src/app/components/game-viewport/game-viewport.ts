import { Component, DestroyRef, HostListener, inject, input, OnDestroy, OnInit, signal, WritableSignal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { SoundService } from '../../services/sound.service';
import { WebSocketService } from '../../services/websocket.service';
import { environment } from '../../environments/environment';
import { RomDetails } from '../../common/models';

@Component({
  selector: 'app-game-viewport',
  imports: [],
  templateUrl: './game-viewport.html',
  styleUrl: './game-viewport.scss',
})
export class GameViewport implements OnInit, OnDestroy {
  rom = input<RomDetails>({ id: 0, name: ''});
  returnChip8Bindings = input<{ [k: string]: string; }>({});

  private webSocketService_ = inject(WebSocketService);
  private destroyRef = inject(DestroyRef);
  private soundService_ = inject (SoundService);

  gameState: WritableSignal<boolean[][]> = signal([]);

  ngOnInit() {
    this.startGame();
  }
  
  ngOnDestroy() {
    this.webSocketService_.disconnect();
  }

  async startGame() {
    try {
      if (!this.rom() || !this.rom()?.id) { throw new Error("No ROM selected. Cannot start game."); }
      this.webSocketService_.disconnect();
      const wsConnection = await this.webSocketService_.connect(environment.connectToGame(this.rom().id));
      this.soundService_.unlockAudio();
      this.listenToWebSocket();
    } catch (err) {
      console.error("Error starting game", err);
    }
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
        this.gameState.set(message.data);
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

  @HostListener('window:keydown', ['$event'])
  handleKeyboardDown(event: KeyboardEvent): void {
    this.sendKeypress(false, this.returnChip8Bindings()[event.code]);
  }

  @HostListener('window:keyup', ['$event'])
  handleKeyboardUp(event: KeyboardEvent): void {
    this.sendKeypress(true, this.returnChip8Bindings()[event.code]);
  }

  sendKeypress(up: boolean, key: string) {
    this.webSocketService_.send({
      type: up ? "key.up" : "key.down",
      data: key
    })
    console.log(`🚀 ~ GameViewport ~ sendKeypress ~ {
      type: up ? "key.up" : "key.down",
      data: key
    }:`, {
      type: up ? "key.up" : "key.down",
      data: key
    })
  }

}
