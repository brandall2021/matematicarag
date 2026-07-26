import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-chat',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule],
  template: `
    <div class="chat-container">
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
    .chat-container { display: flex; height: 100%; }
    .main-chat { flex: 1; display: flex; flex-direction: column; }
    .messages { flex: 1; overflow-y: auto; padding: 1rem; }
    .message { margin-bottom: 1rem; max-width: 70%; }
    @media (max-width: 767px) { .message { max-width: 85%; } }
    .message.user { margin-left: auto; }
    .message.assistant { margin-right: auto; }
    .message-content { padding: 0.75rem 1rem; border-radius: 12px; line-height: 1.5; }
    .message.user .message-content { background: var(--accent); color: var(--bg); }
    .message.assistant .message-content { background: var(--border); color: var(--text); }
    .input-area { padding: 1rem; display: flex; gap: 0.5rem; border-top: 1px solid var(--border); }
    .chat-input { flex: 1; padding: 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--input-bg); color: var(--text); font-size: 1rem; outline: none; }
    .chat-input:focus { border-color: var(--accent); }
  `]
})
export class ChatComponent implements OnInit {
  messages = signal<any[]>([]);
  currentSessionId = signal<string>('');
  newMessage = '';

  constructor(private api: ApiService) {}

  ngOnInit() {}

  sendMessage() {
    if (!this.newMessage) return;
    const msg = this.newMessage;
    this.newMessage = '';
    this.messages.update(msgs => [...msgs, { id: 'temp', role: 'USER', content: msg }]);
    this.api.chat(msg, this.currentSessionId()).subscribe(res => {
      this.messages.update(msgs => [...msgs, res]);
      this.currentSessionId.set(res.sessionId || this.currentSessionId());
    });
  }
}
