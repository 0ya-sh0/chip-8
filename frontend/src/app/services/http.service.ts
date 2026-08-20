import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class HttpService {
  private httpClient = inject(HttpClient);
  
  httpGET(url: string, options?: any) {
    return this.httpClient.get(url, options);
  }
}
