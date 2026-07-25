import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-chat',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RouterLinkActive, MatButtonModule, MatIconModule],
  template: `
    <div class="chat-container">
      <div class="sidebar">
        <div class="sidebar-header">
          <h2>MatematicaRAG</h2>
        </div>
        <nav class="sidebar-nav">
          <a routerLink="/chat" routerLinkActive="active" class="nav-item"><mat-icon>chat</mat-icon> Chat</a>
          <a routerLink="/math" routerLinkActive="active" class="nav-item"><mat-icon>calculate</mat-icon> Matematica</a>
          <a routerLink="/documents" routerLinkActive="active" class="nav-item"><mat-icon>folder</mat-icon> Documentos</a>
          <a routerLink="/history" routerLinkActive="active" class="nav-item"><mat-icon>history</mat-icon> Historial</a>
          @if (auth.hasRole('ADMIN', 'TEACHER')) {
            <a routerLink="/dashboard" routerLinkActive="active" class="nav-item"><mat-icon>dashboard</mat-icon> Panel</a>
          }
          @if (auth.hasRole('ADMIN')) {
            <a routerLink="/admin" routerLinkActive="active" class="nav-item"><mat-icon>admin_panel_settings</mat-icon> Admin</a>
          }
          <a routerLink="/settings" routerLinkActive="active" class="nav-item"><mat-icon>settings</mat-icon> Configuracion</a>
        </nav>
        <div class="sidebar-sessions">
          <div class="sessions-header">
            <span>Sesiones</span>
            <button mat-icon-button (click)="newSession()"><mat-icon>add</mat-icon></button>
          </div>
          <div class="sessions">
            @for (session of sessions(); track session.id) {
              <div class="session-item" [class.active]="currentSessionId() === session.id" (click)="loadSession(session.id)">
                {{ session.title || 'Nueva sesion' }}
              </div>
            }
          </div>
        </div>
        <div class="sidebar-footer">
          <span class="user-name">{{ auth.currentUser()?.name }}</span>
          <button mat-button (click)="auth.logout()">Cerrar sesion</button>
        </div>
      </div>
      <div class="main-chat">
        <div class="messages">
          @for (msg of messages(); track msg.id) {
            <div class="message" [class.user]="msg.role === 'USER'" [class.assistant]="msg.role === 'ASSISTANT'">
              <div class="message-content">{{ msg.content }}</div>
            </div>
          }
        </div>
        <div class="input-area">
          <input [(ngModel)]="newMessage" (keydown.enter)="sendMessage()" placeholder="Escribi tu pregunta de matematica..." class="chat-input">
          <button mat-raised-button color="primary" (click)="sendMessage()" [disabled]="!newMessage">Enviar</button>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .chat-container { display: flex; height: 100vh; background: #1a1a2e; color: white; }
    .sidebar { width: 280px; background: #16213e; display: flex; flex-direction: column; }
    .sidebar-header { padding: 1rem; border-bottom: 1px solid #2a2a4a; }
    .sidebar-header h2 { color: #e2b714; font-size: 1.2rem; margin: 0; }
    .sidebar-nav { padding: 0.5rem; border-bottom: 1px solid #2a2a4a; }
    .nav-item { display: flex; align-items: center; gap: 0.75rem; padding: 0.6rem 0.75rem; border-radius: 8px; color: #ccc; text-decoration: none; font-size: 0.9rem; cursor: pointer; }
    .nav-item:hover { background: #2a2a4a; color: white; }
    .nav-item.active { background: #e2b714; color: #1a1a2e; }
    .nav-item mat-icon { font-size: 20px; width: 20px; height: 20px; }
    .sidebar-sessions { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
    .sessions-header { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem 1rem 0.25rem; font-size: 0.8rem; color: #888; text-transform: uppercase; }
    .sessions { flex: 1; overflow-y: auto; padding: 0.25rem 0.5rem; }
    .session-item { padding: 0.75rem; border-radius: 8px; cursor: pointer; margin-bottom: 0.25rem; font-size: 0.9rem; }
    .session-item:hover { background: #2a2a4a; }
    .session-item.active { background: #e2b714; color: #1a1a2e; }
    .sidebar-footer { padding: 0.75rem 1rem; border-top: 1px solid #2a2a4a; display: flex; justify-content: space-between; align-items: center; }
    .user-name { color: #888; font-size: 0.85rem; }
    .main-chat { flex: 1; display: flex; flex-direction: column; }
    .messages { flex: 1; overflow-y: auto; padding: 1rem; }
    .message { margin-bottom: 1rem; max-width: 70%; }
    .message.user { margin-left: auto; }
    .message.assistant { margin-right: auto; }
    .message-content { padding: 0.75rem 1rem; border-radius: 12px; line-height: 1.5; }
    .message.user .message-content { background: #e2b714; color: #1a1a2e; }
    .message.assistant .message-content { background: #2a2a4a; }
    .input-area { padding: 1rem; display: flex; gap: 0.5rem; border-top: 1px solid #2a2a4a; }
    .chat-input { flex: 1; padding: 0.75rem; border-radius: 8px; border: 1px solid #2a2a4a; background: #16213e; color: white; font-size: 1rem; outline: none; }
    .chat-input:focus { border-color: #e2b714; }
  `]
})
export class ChatComponent implements OnInit {
  sessions = signal<any[]>([]);
  messages = signal<any[]>([]);
  currentSessionId = signal<string>('');
  newMessage = '';

  constructor(private api: ApiService, public auth: AuthService) {}

  ngOnInit() {
    this.api.getSessions().subscribe(s => this.sessions.set(s));
  }

  newSession() {
    this.currentSessionId.set('');
    this.messages.set([]);
  }

  loadSession(id: string) {
    this.currentSessionId.set(id);
    this.api.getSessionMessages(id).subscribe(m => this.messages.set(m));
  }

  sendMessage() {
    if (!this.newMessage) return;
    const msg = this.newMessage;
    this.newMessage = '';
    this.messages.update(msgs => [...msgs, { id: 'temp', role: 'USER', content: msg }]);
    this.api.chat(msg, this.currentSessionId()).subscribe(res => {
      this.messages.update(msgs => [...msgs, res]);
      this.currentSessionId.set(res.sessionId || this.currentSessionId());
      this.api.getSessions().subscribe(s => this.sessions.set(s));
    });
  }
}
