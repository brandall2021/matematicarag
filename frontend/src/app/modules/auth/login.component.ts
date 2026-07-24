import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, MatFormFieldModule, MatInputModule, MatButtonModule],
  template: `
    <div class="auth-container">
      <div class="auth-card">
        <h1>MatematicaRAG</h1>
        <p class="subtitle">Tutor Inteligente de Matematica</p>
        <form (ngSubmit)="onSubmit()">
          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Email</mat-label>
            <input matInput [(ngModel)]="email" name="email" type="email" required>
          </mat-form-field>
          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Password</mat-label>
            <input matInput [(ngModel)]="password" name="password" type="password" required>
          </mat-form-field>
          @if (error()) {
            <div class="error-message">{{ error() }}</div>
          }
          <button mat-raised-button color="primary" type="submit" class="full-width" [disabled]="loading()">
            {{ loading() ? 'Ingresando...' : 'Iniciar Sesion' }}
          </button>
        </form>
        <p class="switch-text">No tenes cuenta? <a routerLink="/register">Registrate</a></p>
      </div>
    </div>
  `,
  styles: [`
    .auth-container { display: flex; justify-content: center; align-items: center; min-height: 100vh; background: #1a1a2e; }
    .auth-card { background: #16213e; padding: 2rem; border-radius: 12px; width: 400px; }
    h1 { color: #e2b714; text-align: center; font-family: 'Newsreader', serif; }
    .subtitle { color: #888; text-align: center; margin-bottom: 2rem; }
    .full-width { width: 100%; }
    .error-message { color: #f44336; text-align: center; margin-bottom: 1rem; }
    .switch-text { color: #888; text-align: center; margin-top: 1rem; }
    a { color: #e2b714; }
  `]
})
export class LoginComponent {
  email = '';
  password = '';
  loading = signal(false);
  error = signal('');

  constructor(private auth: AuthService, private router: Router) {}

  onSubmit() {
    this.loading.set(true);
    this.error.set('');
    this.auth.login(this.email, this.password).subscribe({
      next: () => { this.router.navigate(['/chat']); this.loading.set(false); },
      error: (err) => { this.error.set(err.error?.error || 'Error al iniciar sesion'); this.loading.set(false); }
    });
  }
}
