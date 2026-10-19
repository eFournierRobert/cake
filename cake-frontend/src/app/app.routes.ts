import { Routes } from '@angular/router';
import { guestGuard } from './core/guards/guest-guard';
import { authGuard } from './core/guards/auth-guard';

export const routes: Routes = [
  {
    path: 'login',
    canActivate: [guestGuard],
    loadComponent: async () => {
      const m = await import('./features/login/login.component/login.component');
      return m.LoginComponent;
    },
  },
  {
    path: 'spaces',
    canActivate: [authGuard],
    loadComponent: async () => {
      const m = await import('./features/spaces/spaces/spaces');
      return m.Spaces;
    },
  },
];
