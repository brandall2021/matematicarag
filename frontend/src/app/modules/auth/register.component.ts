import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, MatFormFieldModule, MatInputModule, MatButtonModule],
  template: `
    <div class="auth-container">
      <div class="auth-card">
        <h1>MatematicaRAG</h1>
        <p class="subtitle">Crear Cuenta</p>
        <form (ngSubmit)="onSubmit()">
          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Nombre</mat-label>
            <input matInput [(ngModel)]="name" name="name" required>
          </mat-form-field>
          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Email</mat-label>
            <input matInput [(ngModel)]="email" name="email" type="email" required autocomplete="email">
          </mat-form-field>
          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Contraseña</mat-label>
            <input matInput [(ngModel)]="password" name="password" type="password" required autocomplete="new-password">
          </mat-form-field>
          @if (error()) {
            <div class="error-message">{{ error() }}</div>
          }
          <button mat-raised-button color="primary" type="submit" class="full-width" [disabled]="loading()">
            {{ loading() ? 'Creando...' : 'Registrarse' }}
          </button>
        </form>
        <p class="switch-text">¿Ya tenés cuenta? <a routerLink="/login">Inicia sesión</a></p>
      </div>
      <div class="auth-footer">
        <a href="https://softgroup.com.ar" target="_blank" rel="noopener">softgroup.com.ar</a>
        &copy; 2026
      </div>
    </div>
  `,
  styles: [`
    .auth-container { display: flex; flex-direction: column; justify-content: center; align-items: center; min-height: 100vh; background: var(--bg); padding: 1rem; box-sizing: border-box; }
    .auth-card { background: var(--surface); padding: 2rem; border-radius: 12px; width: 100%; max-width: 400px; box-sizing: border-box; }
    h1 { color: var(--accent); text-align: center; font-family: 'Newsreader', serif; }
    .subtitle { color: var(--text-secondary); text-align: center; margin-bottom: 2rem; }
    .full-width { width: 100%; }
    .error-message { color: #f44336; text-align: center; margin-bottom: 1rem; }
    .switch-text { color: var(--text-secondary); text-align: center; margin-top: 1rem; }
    a { color: var(--accent); }
    .auth-footer { margin-top: auto; padding-top: 1rem; text-align: center; font-size: 0.75rem; color: var(--text-secondary); }
    .auth-footer a { color: var(--text-secondary); text-decoration: none; }
    .auth-footer a:hover { color: var(--accent); }
    @media (max-width: 480px) { .auth-card { padding: 1.5rem; } }
  `]
})
export class RegisterComponent {
  name = '';
  email = '';
  password = '';
  loading = signal(false);
  error = signal('');

  constructor(private auth: AuthService, private router: Router) {}

  onSubmit() {
    this.loading.set(true);
    this.error.set('');
    this.auth.register(this.email, this.password, this.name).subscribe({
      next: () => { this.router.navigate(['/chat']); this.loading.set(false); },
      error: (err) => { this.error.set(err.error?.error || 'Error al registrar'); this.loading.set(false); }
    });
  }
}
