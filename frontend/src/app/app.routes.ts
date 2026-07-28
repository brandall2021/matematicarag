import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { roleGuard } from './core/guards/role.guard';
import { LayoutComponent } from './shared/layout.component';

export const routes: Routes = [
  { path: 'login', loadComponent: () => import('./modules/auth/login.component').then(m => m.LoginComponent) },
  { path: 'register', loadComponent: () => import('./modules/auth/register.component').then(m => m.RegisterComponent) },
  {
    path: '',
    component: LayoutComponent,
    canActivate: [authGuard],
    children: [
      { path: '', redirectTo: 'chat', pathMatch: 'full' },
      { path: 'chat', loadComponent: () => import('./modules/chat/chat.component').then(m => m.ChatComponent) },
      { path: 'math', loadComponent: () => import('./modules/math/math.component').then(m => m.MathComponent) },
      { path: 'tutor', loadComponent: () => import('./modules/tutor/tutor.component').then(m => m.TutorComponent) },
      { path: 'documents', loadComponent: () => import('./modules/documents/documents.component').then(m => m.DocumentsComponent) },
      { path: 'bdvectorial', loadComponent: () => import('./modules/bdvectorial/bdvectorial.component').then(m => m.BdvectorialComponent) },
      { path: 'history', loadComponent: () => import('./modules/history/history.component').then(m => m.HistoryComponent) },
      { path: 'dashboard', loadComponent: () => import('./modules/dashboard/dashboard.component').then(m => m.DashboardComponent), canActivate: [roleGuard], data: { roles: ['ADMIN', 'TEACHER'] } },
      { path: 'settings', loadComponent: () => import('./modules/settings/settings.component').then(m => m.SettingsComponent), canActivate: [roleGuard], data: { roles: ['ADMIN'] } },
      { path: 'assessment', loadComponent: () => import('./modules/assessment/assessment.component').then(m => m.AssessmentComponent) },
      { path: 'analytics', loadComponent: () => import('./modules/analytics/analytics.component').then(m => m.AnalyticsComponent) },
      { path: 'my-progress', loadComponent: () => import('./modules/student-progress/student-progress.component').then(m => m.StudentProgressComponent) },
      { path: 'teacher', loadComponent: () => import('./modules/teacher-dashboard/teacher-dashboard.component').then(m => m.TeacherDashboardComponent), canActivate: [roleGuard], data: { roles: ['ADMIN', 'TEACHER'] } },
    ]
  },
];
