import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-chat',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule],
  template: `
    <div class="chat-container">
      <div class="sidebar">
        <div class="sidebar-header">
          <h2>MatematicaRAG</h2>
          <button mat-icon-button (click)="newSession()">+</button>
        </div>
        <div class="sessions">
          @for (session of sessions(); track session.id) {
            <div class="session-item" [class.active]="currentSessionId() === session.id" (click)="loadSession(session.id)">
              {{ session.title || 'Nueva sesion' }}
            </div>
          }
        </div>
        <div class="sidebar-footer">
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
    .sidebar-header { padding: 1rem; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #2a2a4a; }
    .sidebar-header h2 { color: #e2b714; font-size: 1.2rem; }
    .sessions { flex: 1; overflow-y: auto; padding: 0.5rem; }
    .session-item { padding: 0.75rem; border-radius: 8px; cursor: pointer; margin-bottom: 0.25rem; font-size: 0.9rem; }
    .session-item:hover { background: #2a2a4a; }
    .session-item.active { background: #e2b714; color: #1a1a2e; }
    .sidebar-footer { padding: 1rem; border-top: 1px solid #2a2a4a; }
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
