import { inject, Service } from '@angular/core';
import { HttpClient } from '@angular/common/http';

export interface User {
  uuid: string;
  username: string;
  role: string;
  first_name: string;
  last_name: string;
}

@Service()
export class UserService {
  private http = inject(HttpClient);
  private readonly baseUrl = '/api';

  getCurrentUser() {
    return this.http.get(`${this.baseUrl}/user`, { withCredentials: true });
  }
}
