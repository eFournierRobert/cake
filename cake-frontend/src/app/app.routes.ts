import { Routes } from '@angular/router';
import { guestGuard } from './core/guards/guest-guard';

export const routes: Routes = [
  {
    path: 'login',
    canActivate: [guestGuard],
    loadComponent: async () => {
      const m = await import('./features/login/login.component/login.component');
      return m.LoginComponent;
    },
  },
];
