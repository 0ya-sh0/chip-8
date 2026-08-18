import { ChangeDetectorRef, Component, DestroyRef, inject, OnInit, signal } from '@angular/core';
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

  async ngOnInit() {
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
    if(message.type === 'display.draw') {
      this.gameState = message?.data;
      this.cdr.detectChanges();
    }
    if(message.type === 'sound.play') {
      this.soundService_.playBitSound(900, 0.1); 
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
}
