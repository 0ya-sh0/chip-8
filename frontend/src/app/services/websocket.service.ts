import { Injectable, OnDestroy } from '@angular/core';
import { Subject, BehaviorSubject, Observable } from 'rxjs';
import { filter } from 'rxjs/operators';

/** JSON control message from the server. */
export interface ControlMessage {
  type: string;
  data?: string;
  /** Present on `display.init` only. */
  width?: number;
  height?: number;
}

/** JSON message sent to the server. */
export interface ClientMessage {
  type: string;
  data?: string;
}

/**
 * WebSocketService owns the connection to the playground server.
 *
 * The server uses the two WebSocket message types to separate two very
 * different kinds of traffic, so this service exposes two streams:
 *
 * - `frames$` carries binary framebuffers: 256 bytes, one bit per pixel, in
 *   scanline order, MSB first. Sending one boolean per pixel as JSON was ~12.4KB
 *   for the same 2048 pixels.
 * - `control$` carries occasional JSON messages (sound, display geometry).
 *
 * `binaryType = 'arraybuffer'` is what makes the binary path work; without it
 * the browser delivers a Blob and every frame has to go through an async read.
 */
@Injectable({
  providedIn: 'root',
})
export class WebSocketService implements OnDestroy {
  private socket: WebSocket | null = null;

  private controlSubject = new Subject<ControlMessage>();
  private framesSubject = new Subject<Uint8Array>();
  private connectionStatusSubject = new BehaviorSubject<boolean>(false);

  readonly control$ = this.controlSubject.asObservable();
  readonly frames$ = this.framesSubject.asObservable();
  readonly connectionStatus$ = this.connectionStatusSubject.asObservable();

  /** Opens a connection, resolving once the socket is open. */
  connect(url: string): Promise<void> {
    this.disconnect();

    return new Promise<void>((resolve, reject) => {
      let settled = false;

      const socket = new WebSocket(url);
      socket.binaryType = 'arraybuffer';
      this.socket = socket;

      socket.onopen = () => {
        settled = true;
        this.connectionStatusSubject.next(true);
        resolve();
      };

      socket.onmessage = (event: MessageEvent) => {
        if (typeof event.data === 'string') {
          try {
            this.controlSubject.next(JSON.parse(event.data) as ControlMessage);
          } catch (error) {
            console.error('websocket: bad control message', error);
          }
          return;
        }
        this.framesSubject.next(new Uint8Array(event.data as ArrayBuffer));
      };

      socket.onerror = (event: Event) => {
        this.connectionStatusSubject.next(false);
        // Only the failure to connect is the promise's business; later errors
        // surface through connectionStatus$ and onclose.
        if (!settled) {
          settled = true;
          reject(new Error('websocket connection failed'));
        }
      };

      socket.onclose = () => {
        this.connectionStatusSubject.next(false);
        if (!settled) {
          settled = true;
          reject(new Error('websocket closed before opening'));
        }
      };
    });
  }

  send(message: ClientMessage): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    }
  }

  /** Emits only control messages of the given type. */
  onControl(type: string): Observable<ControlMessage> {
    return this.control$.pipe(filter((msg) => msg.type === type));
  }

  disconnect(): void {
    const socket = this.socket;
    if (!socket) {
      return;
    }
    this.socket = null;
    // Drop the handlers before closing so a late event cannot push state for a
    // connection the caller has already abandoned.
    socket.onopen = null;
    socket.onmessage = null;
    socket.onerror = null;
    socket.onclose = null;
    socket.close();
    this.connectionStatusSubject.next(false);
  }

  ngOnDestroy(): void {
    this.disconnect();
    this.controlSubject.complete();
    this.framesSubject.complete();
  }
}
