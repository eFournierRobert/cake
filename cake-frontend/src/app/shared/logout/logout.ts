import { Component, inject } from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';
import { Router } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

@Component({
  imports: [TranslatePipe],
  selector: 'app-logout',
  styleUrl: './logout.css',
  templateUrl: './logout.html',
})
export class Logout {
  private readonly router = inject(Router);
  private readonly authService = inject(AuthService);

  logout() {
    this.authService.logout().subscribe(
      next => {
        this.authService.setAuthenticated(false);
        this.router.navigate(['login']);
      }
    );
  }
}
