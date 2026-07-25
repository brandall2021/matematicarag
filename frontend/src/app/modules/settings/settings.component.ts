import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSelectModule } from '@angular/material/select';
import { ThemeService } from '../../core/services/theme.service';

interface User {
  id: string;
  email: string;
  name: string;
  lastName: string;
  role: string;
  createdAt: string;
}

interface Setting {
  key: string;
  value: string;
  description: string;
}

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule, MatSelectModule],
  template: `
    <div class="settings-container">
      <h1>Configuracion</h1>

      <div class="tabs">
        <button [class.active]="tab() === 'users'" (click)="tab.set('users')">Usuarios</button>
        <button [class.active]="tab() === 'api'" (click)="tab.set('api'); loadSettings()">API Keys</button>
        <button [class.active]="tab() === 'appearance'" (click)="tab.set('appearance')">Apariencia</button>
      </div>

      @if (tab() === 'users') {
        <div class="section">
          <h2>Gestion de Usuarios</h2>
          <div class="user-list">
            @for (user of users(); track user.id) {
              <div class="user-card">
                <div class="user-info">
                  <div class="user-name">{{ user.name }} {{ user.lastName }}</div>
                  <div class="user-email">{{ user.email }}</div>
                  <div class="user-date">Registro: {{ user.createdAt | date:'dd/MM/yyyy' }}</div>
                </div>
                <div class="user-actions">
                  <select [ngModel]="user.role" (ngModelChange)="updateRole(user.id, $event)" class="role-select">
                    <option value="STUDENT">Alumno</option>
                    <option value="TEACHER">Profesor</option>
                    <option value="ADMIN">Administrador</option>
                  </select>
                  <button mat-icon-button color="warn" (click)="deleteUser(user.id, user.name)">
                    <mat-icon>delete</mat-icon>
                  </button>
                </div>
              </div>
            }
            @if (users().length === 0) {
              <p class="empty">No hay usuarios registrados.</p>
            }
          </div>
        </div>
      }

      @if (tab() === 'api') {
        <div class="section">
          <h2>Configuracion de API Keys</h2>
          @for (setting of settings(); track setting.key) {
            <div class="setting-item">
              <label>{{ setting.description || setting.key }}</label>
              <div class="setting-row">
                <input [type]="setting.key.includes('KEY') || setting.key.includes('SECRET') ? 'password' : 'text'"
                       [ngModel]="setting.value"
                       (ngModelChange)="setting.value = $event"
                       class="setting-input"
                       [placeholder]="setting.key">
                <button mat-raised-button color="primary" (click)="saveSetting(setting)">Guardar</button>
              </div>
            </div>
          }
          @if (settings().length === 0) {
            <p class="empty">No hay configuraciones guardadas.</p>
          }
        </div>
      }

      @if (tab() === 'appearance') {
        <div class="section">
          <h2>Apariencia</h2>
          <div class="theme-toggle">
            <span>Tema actual: {{ themeService.currentTheme() === 'dark' ? 'Oscuro' : 'Claro' }}</span>
            <button mat-raised-button (click)="themeService.toggle()">
              <mat-icon>{{ themeService.currentTheme() === 'dark' ? 'light_mode' : 'dark_mode' }}</mat-icon>
              Cambiar a {{ themeService.currentTheme() === 'dark' ? 'Claro' : 'Oscuro' }}
            </button>
          </div>
        </div>
      }

      @if (message()) {
        <div class="message" [class.error]="messageType() === 'error'">{{ message() }}</div>
      }
    </div>
  `,
  styles: [`
    .settings-container { padding: 2rem; max-width: 900px; margin: 0 auto; }
    h1 { color: var(--accent); margin-bottom: 1.5rem; }
    h2 { color: var(--text); margin-bottom: 1rem; font-size: 1.1rem; }
    .tabs { display: flex; gap: 0.5rem; margin-bottom: 2rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
    .tabs button { padding: 0.5rem 1rem; border: none; background: transparent; color: var(--text-secondary); cursor: pointer; border-radius: 8px 8px 0 0; font-size: 0.9rem; }
    .tabs button.active { background: var(--accent); color: var(--bg); font-weight: 600; }
    .tabs button:hover:not(.active) { background: var(--surface); }
    .section { background: var(--surface); border-radius: 12px; padding: 1.5rem; }
    .user-list { display: flex; flex-direction: column; gap: 0.75rem; }
    .user-card { display: flex; justify-content: space-between; align-items: center; padding: 1rem; background: var(--bg); border-radius: 8px; }
    .user-name { font-weight: 600; color: var(--text); }
    .user-email { color: var(--text-secondary); font-size: 0.85rem; }
    .user-date { color: var(--text-secondary); font-size: 0.8rem; margin-top: 0.25rem; }
    .user-actions { display: flex; align-items: center; gap: 0.5rem; }
    .role-select { padding: 0.4rem 0.75rem; border-radius: 6px; border: 1px solid var(--border); background: var(--bg); color: var(--text); font-size: 0.85rem; cursor: pointer; }
    .setting-item { margin-bottom: 1.25rem; }
    .setting-item label { display: block; color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 0.5rem; }
    .setting-row { display: flex; gap: 0.5rem; }
    .setting-input { flex: 1; padding: 0.6rem 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--bg); color: var(--text); font-size: 0.9rem; outline: none; }
    .setting-input:focus { border-color: var(--accent); }
    .theme-toggle { display: flex; justify-content: space-between; align-items: center; }
    .theme-toggle span { color: var(--text); }
    .message { margin-top: 1rem; padding: 0.75rem; border-radius: 8px; background: #1b5e20; color: white; text-align: center; }
    .message.error { background: #b71c1c; }
    .empty { color: var(--text-secondary); text-align: center; padding: 2rem; }
  `]
})
export class SettingsComponent implements OnInit {
  tab = signal<'users' | 'api' | 'appearance'>('users');
  users = signal<User[]>([]);
  settings = signal<Setting[]>([]);
  message = signal('');
  messageType = signal<'success' | 'error'>('success');

  constructor(private http: HttpClient, public themeService: ThemeService) {}

  ngOnInit() {
    this.loadUsers();
  }

  loadUsers() {
    this.http.get<User[]>(`${environment.apiUrl}/api/users`).subscribe({
      next: (users) => this.users.set(users),
      error: () => this.showMessage('Error al cargar usuarios', 'error')
    });
  }

  loadSettings() {
    this.http.get<Setting[]>(`${environment.apiUrl}/api/settings`).subscribe({
      next: (s) => this.settings.set(s),
      error: () => this.showMessage('Error al cargar configuraciones', 'error')
    });
  }

  updateRole(userId: string, newRole: string) {
    this.http.put(`${environment.apiUrl}/api/users/${userId}/role`, { role: newRole }).subscribe({
      next: () => {
        this.loadUsers();
        this.showMessage('Rol actualizado');
      },
      error: () => this.showMessage('Error al actualizar rol', 'error')
    });
  }

  deleteUser(userId: string, name: string) {
    if (!confirm(`Eliminar usuario "${name}"?`)) return;
    this.http.delete(`${environment.apiUrl}/api/users/${userId}`).subscribe({
      next: () => {
        this.loadUsers();
        this.showMessage('Usuario eliminado');
      },
      error: () => this.showMessage('Error al eliminar usuario', 'error')
    });
  }

  saveSetting(setting: Setting) {
    this.http.put(`${environment.apiUrl}/api/settings/${setting.key}`, setting).subscribe({
      next: () => this.showMessage('Configuracion guardada'),
      error: () => this.showMessage('Error al guardar', 'error')
    });
  }

  private showMessage(msg: string, type: 'success' | 'error' = 'success') {
    this.message.set(msg);
    this.messageType.set(type);
    setTimeout(() => this.message.set(''), 3000);
  }
}
