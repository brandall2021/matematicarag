import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { roleGuard } from './core/guards/role.guard';

export const routes: Routes = [
  { path: '', redirectTo: '/chat', pathMatch: 'full' },
  { path: 'login', loadComponent: () => import('./modules/auth/login.component').then(m => m.LoginComponent) },
  { path: 'register', loadComponent: () => import('./modules/auth/register.component').then(m => m.RegisterComponent) },
  { path: 'chat', loadComponent: () => import('./modules/chat/chat.component').then(m => m.ChatComponent), canActivate: [authGuard] },
  { path: 'math', loadComponent: () => import('./modules/math/math.component').then(m => m.MathComponent), canActivate: [authGuard] },
  { path: 'documents', loadComponent: () => import('./modules/documents/documents.component').then(m => m.DocumentsComponent), canActivate: [authGuard] },
  { path: 'history', loadComponent: () => import('./modules/history/history.component').then(m => m.HistoryComponent), canActivate: [authGuard] },
  { path: 'admin', loadComponent: () => import('./modules/admin/admin.component').then(m => m.AdminComponent), canActivate: [authGuard, roleGuard], data: { roles: ['ADMIN'] } },
  { path: 'settings', loadComponent: () => import('./modules/settings/settings.component').then(m => m.SettingsComponent), canActivate: [authGuard] },
  { path: 'dashboard', loadComponent: () => import('./modules/dashboard/dashboard.component').then(m => m.DashboardComponent), canActivate: [authGuard, roleGuard], data: { roles: ['ADMIN', 'TEACHER'] } },
];
