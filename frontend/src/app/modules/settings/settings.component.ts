import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

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
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule],
  template: `
    <div class="settings-container">
      <h1>Configuracion</h1>

      <div class="tabs">
        <button [class.active]="tab() === 'users'" (click)="tab.set('users')"><mat-icon>people</mat-icon> Usuarios</button>
        <button [class.active]="tab() === 'api'" (click)="tab.set('api'); loadSettings()"><mat-icon>key</mat-icon> API Keys</button>
      </div>

      @if (tab() === 'users') {
        <div class="section">
          <div class="section-header">
            <h2>Gestion de Usuarios</h2>
            <button mat-raised-button color="primary" (click)="showCreate.set(!showCreate())">
              <mat-icon>add</mat-icon> Crear usuario
            </button>
          </div>

          @if (showCreate()) {
            <div class="create-form">
              <input [(ngModel)]="newUser.name" placeholder="Nombre" class="form-input">
              <input [(ngModel)]="newUser.lastName" placeholder="Apellido" class="form-input">
              <input [(ngModel)]="newUser.email" placeholder="Email" class="form-input" type="email">
              <input [(ngModel)]="newUser.password" placeholder="Password" class="form-input" type="password">
              <select [(ngModel)]="newUser.role" class="form-select">
                <option value="STUDENT">Alumno</option>
                <option value="TEACHER">Profesor</option>
                <option value="ADMIN">Administrador</option>
              </select>
              <div class="form-actions">
                <button mat-button (click)="showCreate.set(false)">Cancelar</button>
                <button mat-raised-button color="primary" (click)="createUser()">Crear</button>
              </div>
            </div>
          }

          <div class="user-list">
            @for (user of users(); track user.id) {
              <div class="user-card">
                <div class="user-info">
                  <div class="user-name">{{ user.name }} {{ user.lastName }}</div>
                  <div class="user-email">{{ user.email }}</div>
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
          </div>
        </div>
      }

      @if (tab() === 'api') {
        <div class="section">
          <div class="section-header">
            <h2>API Keys</h2>
            <button mat-raised-button color="primary" (click)="showNewKey.set(!showNewKey())">
              <mat-icon>add</mat-icon> Agregar key
            </button>
          </div>

          @if (showNewKey()) {
            <div class="create-form">
              <input [(ngModel)]="newKey.key" placeholder="Nombre (ej: OPENAI_API_KEY)" class="form-input">
              <input [(ngModel)]="newKey.value" placeholder="Valor de la API key" class="form-input" type="password">
              <input [(ngModel)]="newKey.description" placeholder="Descripcion (opcional)" class="form-input">
              <div class="form-actions">
                <button mat-button (click)="showNewKey.set(false)">Cancelar</button>
                <button mat-raised-button color="primary" (click)="createSetting()">Guardar</button>
              </div>
            </div>
          }

          <div class="settings-list">
            @for (setting of settings(); track setting.key) {
              <div class="setting-card">
                <div class="setting-header">
                  <span class="setting-key">{{ setting.key }}</span>
                  <span class="setting-desc">{{ setting.description }}</span>
                </div>
                <div class="setting-row">
                  <input [type]="showValue[setting.key] ? 'text' : 'password'"
                         [(ngModel)]="setting.value"
                         class="setting-input">
                  <button mat-icon-button (click)="showValue[setting.key] = !showValue[setting.key]">
                    <mat-icon>{{ showValue[setting.key] ? 'visibility_off' : 'visibility' }}</mat-icon>
                  </button>
                  <button mat-raised-button color="primary" (click)="saveSetting(setting)">Guardar</button>
                  <button mat-icon-button color="warn" (click)="deleteSetting(setting.key)">
                    <mat-icon>delete</mat-icon>
                  </button>
                </div>
              </div>
            }
            @if (settings().length === 0 && !showNewKey()) {
              <p class="empty">No hay API keys configuradas. Hace click en "Agregar key" para crear una.</p>
            }
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
    .tabs { display: flex; gap: 0.5rem; margin-bottom: 2rem; }
    .tabs button { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 1rem; border: none; background: var(--surface); color: var(--text-secondary); cursor: pointer; border-radius: 8px; font-size: 0.9rem; }
    .tabs button.active { background: var(--accent); color: var(--bg); font-weight: 600; }
    .tabs button:hover:not(.active) { background: var(--border); }
    .tabs button mat-icon { font-size: 18px; width: 18px; height: 18px; }
    .section { background: var(--surface); border-radius: 12px; padding: 1.5rem; }
    .section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
    .section-header h2 { margin: 0; }
    .create-form { background: var(--bg); padding: 1rem; border-radius: 8px; margin-bottom: 1.5rem; display: flex; flex-direction: column; gap: 0.75rem; }
    .form-input, .form-select { padding: 0.6rem 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--input-bg); color: var(--text); font-size: 0.9rem; outline: none; }
    .form-input:focus { border-color: var(--accent); }
    .form-select { cursor: pointer; }
    .form-actions { display: flex; justify-content: flex-end; gap: 0.5rem; }
    .user-list { display: flex; flex-direction: column; gap: 0.5rem; }
    .user-card { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem 1rem; background: var(--bg); border-radius: 8px; }
    .user-name { font-weight: 600; color: var(--text); font-size: 0.9rem; }
    .user-email { color: var(--text-secondary); font-size: 0.8rem; }
    .user-actions { display: flex; align-items: center; gap: 0.5rem; }
    .role-select { padding: 0.35rem 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--bg); color: var(--text); font-size: 0.8rem; cursor: pointer; }
    .settings-list { display: flex; flex-direction: column; gap: 0.75rem; }
    .setting-card { background: var(--bg); padding: 1rem; border-radius: 8px; }
    .setting-header { margin-bottom: 0.5rem; }
    .setting-key { font-weight: 600; color: var(--accent); font-size: 0.9rem; }
    .setting-desc { color: var(--text-secondary); font-size: 0.8rem; margin-left: 0.5rem; }
    .setting-row { display: flex; gap: 0.5rem; align-items: center; }
    .setting-input { flex: 1; padding: 0.6rem 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--input-bg); color: var(--text); font-size: 0.9rem; outline: none; font-family: monospace; }
    .setting-input:focus { border-color: var(--accent); }
    .message { margin-top: 1rem; padding: 0.75rem; border-radius: 8px; background: #1b5e20; color: white; text-align: center; }
    .message.error { background: #b71c1c; }
    .empty { color: var(--text-secondary); text-align: center; padding: 2rem; }
  `]
})
export class SettingsComponent implements OnInit {
  tab = signal<'users' | 'api'>('users');
  users = signal<User[]>([]);
  settings = signal<Setting[]>([]);
  showCreate = signal(false);
  showNewKey = signal(false);
  message = signal('');
  messageType = signal<'success' | 'error'>('success');
  newUser = { name: '', lastName: '', email: '', password: '', role: 'STUDENT' };
  newKey = { key: '', value: '', description: '' };
  showValue: Record<string, boolean> = {};

  constructor(private http: HttpClient) {}

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

  createUser() {
    if (!this.newUser.name || !this.newUser.email || !this.newUser.password) {
      this.showMessage('Nombre, email y password son requeridos', 'error');
      return;
    }
    this.http.post(`${environment.apiUrl}/api/auth/register`, this.newUser).subscribe({
      next: () => {
        this.loadUsers();
        this.showCreate.set(false);
        this.newUser = { name: '', lastName: '', email: '', password: '', role: 'STUDENT' };
        this.showMessage('Usuario creado');
      },
      error: (err) => this.showMessage(err.error?.error || 'Error al crear usuario', 'error')
    });
  }

  updateRole(userId: string, newRole: string) {
    this.http.put(`${environment.apiUrl}/api/users/${userId}/role`, { role: newRole }).subscribe({
      next: () => { this.loadUsers(); this.showMessage('Rol actualizado'); },
      error: () => this.showMessage('Error al actualizar rol', 'error')
    });
  }

  deleteUser(userId: string, name: string) {
    if (!confirm(`Eliminar usuario "${name}"?`)) return;
    this.http.delete(`${environment.apiUrl}/api/users/${userId}`).subscribe({
      next: () => { this.loadUsers(); this.showMessage('Usuario eliminado'); },
      error: () => this.showMessage('Error al eliminar usuario', 'error')
    });
  }

  createSetting() {
    if (!this.newKey.key || !this.newKey.value) {
      this.showMessage('Nombre y valor son requeridos', 'error');
      return;
    }
    this.http.put(`${environment.apiUrl}/api/settings/${this.newKey.key}`, this.newKey).subscribe({
      next: () => {
        this.loadSettings();
        this.showNewKey.set(false);
        this.newKey = { key: '', value: '', description: '' };
        this.showMessage('API key guardada');
      },
      error: () => this.showMessage('Error al guardar', 'error')
    });
  }

  saveSetting(setting: Setting) {
    this.http.put(`${environment.apiUrl}/api/settings/${setting.key}`, setting).subscribe({
      next: () => this.showMessage('Guardado'),
      error: () => this.showMessage('Error al guardar', 'error')
    });
  }

  deleteSetting(key: string) {
    if (!confirm(`Eliminar "${key}"?`)) return;
    this.http.delete(`${environment.apiUrl}/api/settings/${key}`).subscribe({
      next: () => { this.loadSettings(); this.showMessage('Eliminado'); },
      error: () => this.showMessage('Error al eliminar', 'error')
    });
  }

  private showMessage(msg: string, type: 'success' | 'error' = 'success') {
    this.message.set(msg);
    this.messageType.set(type);
    setTimeout(() => this.message.set(''), 3000);
  }
}
