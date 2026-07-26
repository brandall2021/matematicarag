import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

interface RagSource {
  id: string;
  content: string;
  score: number;
  filename: string;
  page?: number;
  section?: string;
  url?: string;
}

interface ChatMsg {
  id: string;
  role: 'USER' | 'ASSISTANT';
  content: string;
  sources?: RagSource[];
}

@Component({
  selector: 'app-chat',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule],
  template: `
    <div class="chat-container">
      <div class="main-chat">
        <div class="messages">
          @for (msg of messages(); track msg.id) {
            <div class="message" [class.user]="msg.role === 'USER'" [class.assistant]="msg.role === 'ASSISTANT'">
              <div class="message-content">{{ msg.content }}</div>
              @if (msg.sources && msg.sources.length > 0) {
                <div class="sources">
                  <span class="sources-label"><mat-icon>menu_book</mat-icon> Fuentes:</span>
                  @for (src of msg.sources; track src.id) {
                    <button class="source-chip" (click)="openSource(src)">
                      <mat-icon class="source-icon">description</mat-icon>
                      {{ src.filename }}
                      @if (src.page) {
                        <span class="source-page">p.{{ src.page }}</span>
                      }
                      @if (src.section) {
                        <span class="source-section">{{ src.section }}</span>
                      }
                    </button>
                  }
                </div>
              }
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
    .chat-container { display: flex; height: 100%; }
    .main-chat { flex: 1; display: flex; flex-direction: column; }
    .messages { flex: 1; overflow-y: auto; padding: 1rem; }
    .message { margin-bottom: 1rem; max-width: 70%; }
    @media (max-width: 767px) { .message { max-width: 85%; } }
    .message.user { margin-left: auto; }
    .message.assistant { margin-right: auto; }
    .message-content { padding: 0.75rem 1rem; border-radius: 12px; line-height: 1.6; white-space: pre-wrap; word-break: break-word; }
    .message.user .message-content { background: var(--accent); color: var(--bg); }
    .message.assistant .message-content { background: var(--border); color: var(--text); }
    .sources { display: flex; flex-wrap: wrap; gap: 0.4rem; margin-top: 0.5rem; align-items: center; }
    .sources-label { display: inline-flex; align-items: center; gap: 0.2rem; font-size: 0.7rem; color: var(--text-secondary); font-weight: 600; }
    .sources-label mat-icon { font-size: 13px; width: 13px; height: 13px; }
    .source-chip { display: inline-flex; align-items: center; gap: 0.3rem; padding: 0.2rem 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface); color: var(--text-secondary); font-size: 0.72rem; cursor: pointer; transition: all 0.15s; }
    .source-chip:hover { border-color: var(--accent); color: var(--accent); background: var(--bg); }
    .source-icon { font-size: 12px; width: 12px; height: 12px; }
    .source-page { color: var(--accent); font-weight: 600; }
    .source-section { color: var(--text-secondary); font-style: italic; max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .input-area { padding: 1rem; display: flex; gap: 0.5rem; border-top: 1px solid var(--border); }
    .chat-input { flex: 1; padding: 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--input-bg); color: var(--text); font-size: 1rem; outline: none; }
    .chat-input:focus { border-color: var(--accent); }
  `]
})
export class ChatComponent {
  messages = signal<ChatMsg[]>([]);
  currentSessionId = signal<string>('');
  newMessage = '';

  constructor(private api: ApiService, private router: Router) {}

  sendMessage() {
    if (!this.newMessage) return;
    const msg = this.newMessage;
    this.newMessage = '';
    this.messages.update(msgs => [...msgs, { id: 'temp-user', role: 'USER', content: msg }]);
    this.api.chat(msg, this.currentSessionId()).subscribe(res => {
      const assistantMsg: ChatMsg = {
        id: res.id || 'temp-assistant',
        role: 'ASSISTANT',
        content: res.content,
        sources: res.sources ? (typeof res.sources === 'string' ? JSON.parse(res.sources) : res.sources) : undefined
      };
      this.messages.update(msgs => [...msgs, assistantMsg]);
      this.currentSessionId.set(res.sessionId || this.currentSessionId());
    });
  }

  openSource(src: RagSource) {
    if (src.url) {
      this.router.navigateByUrl(src.url);
    }
  }
}
