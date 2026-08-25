import { Injectable } from '@angular/core';
import { Subject, BehaviorSubject, Observable } from 'rxjs';
import { filter, takeUntil } from 'rxjs/operators';

export interface WebSocketMessage {
  type: string;
  data: any;
  timestamp?: number;
}

@Injectable({
  providedIn: 'root',
})
export class WebSocketService {
  private socket: WebSocket | null = null;

  private messagesSubject = new Subject<WebSocketMessage>();
  private connectionStatusSubject = new BehaviorSubject<boolean>(false);
  private destroy$ = new Subject<void>();

  // Public observables
  messages$ = this.messagesSubject.asObservable();
  connectionStatus$ = this.connectionStatusSubject.asObservable();
  isConnected$ = this.connectionStatusSubject.asObservable();

  constructor() { }

  /**
   * Connect to WebSocket server
   */
  connect(url: string): Promise<void | boolean> {
    return new Promise((resolve, reject) => {
      try {
        this.socket = new WebSocket(url);

        this.socket.onopen = () => {
          console.log('WebSocket connected');
          this.connectionStatusSubject.next(true);
          resolve(true);
        };

        this.socket.onmessage = (event: MessageEvent) => {
          try {
            const message: WebSocketMessage = JSON.parse(event.data);
            this.messagesSubject.next(message);
          } catch (error) {
            console.error('Failed to parse message:', error);
          }
        };

        this.socket.onerror = (error: Event) => {
          console.error('WebSocket error:', error);
          this.connectionStatusSubject.next(false);
          reject(error);
        };

        this.socket.onclose = () => {
          console.log('WebSocket disconnected');
          this.connectionStatusSubject.next(false);
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Send message to server
   */
  send(message: WebSocketMessage): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket is not connected');
    }
  }

  /**
   * Subscribe to specific message types
   */
  onMessage(type: string): Observable<WebSocketMessage> {
    return this.messages$.pipe(
      filter(msg => msg.type === type),
      takeUntil(this.destroy$)
    );
  }

  /**
   * Disconnect from server
   */
  disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    this.disconnect();
  }
}