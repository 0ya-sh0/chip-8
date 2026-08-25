import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
export class HttpService {
  private httpClient = inject(HttpClient);
  
  httpGET<T>(url: string, options?: { [key: string]: any }) {
    return this.httpClient.get<T>(url, options);
  }
}
