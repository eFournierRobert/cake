import { inject, Service, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { catchError, map, of, tap } from 'rxjs';

export interface LoginBody {
  username: string;
  password: string;
}

@Service()
export class AuthService {
  private http = inject(HttpClient);
  private readonly baseUrl = '/api';

  readonly authenticated = signal(false);

  checkSession() {
    return this.http.get(`${this.baseUrl}/user`).pipe(
      tap(() => this.authenticated.set(true)),
        map(() => true),
        catchError(() => {
          this.authenticated.set(false);
          return of(false);
        }));
  }

  setAuthenticated(value: boolean): void {
    this.authenticated.set(value);
  }

  login(body: LoginBody) {
    return this.http.post(`${this.baseUrl}/login`, body)
  }

  logout() {
    return this.http.post(`${this.baseUrl}/logout`, {}, { withCredentials: true });
  }
}
