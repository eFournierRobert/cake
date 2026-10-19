import { CanActivateFn, Router } from '@angular/router';
import { inject } from '@angular/core';
import { AuthService } from '../services/auth.service';
import { map } from 'rxjs';

export const guestGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  return auth.checkSession().pipe(
    map(isAuthenticated => isAuthenticated ? router.createUrlTree(['/spaces']) : true)
  )
};
