import { Component, DestroyRef, inject, OnInit, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { WebSocketService } from './services/websocket.service';
import { firstValueFrom, Subscription } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit {
  private webSocketService_ = inject(WebSocketService);
  private destroyRef = inject(DestroyRef);
  protected readonly title = signal('frontend');
  public gameState: Array<Array<boolean>> = [];

  async ngOnInit() {
    try {
      const wsConnection = await this.webSocketService_.connect();
      this.listenToWebSocket();
    } catch (err) {
      console.error("Error connecting to WebSocket", err);
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
    if(message.type === 'display.draw') this.gameState = message?.data;
  }
}
