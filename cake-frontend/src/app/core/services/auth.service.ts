import { inject, Service } from '@angular/core';
import { HttpClient } from '@angular/common/http';

export interface LoginBody {
  username: string;
  password: string;
}

@Service()
export class AuthService {
  private http = inject(HttpClient);
  private readonly baseUrl = '/api';

  login(body: LoginBody) {
    return this.http.post(`${this.baseUrl}/login`, body)
  }

  logout() {
    return this.http.post(`${this.baseUrl}/logout`, {}, { withCredentials: true });
  }
}
