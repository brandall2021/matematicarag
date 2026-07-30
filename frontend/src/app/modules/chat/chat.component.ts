import { Component, signal, ViewChild, ElementRef, AfterViewInit, OnDestroy, NgZone, CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MathfieldElement } from 'mathlive';

MathfieldElement.fontsDirectory = 'https://cdn.jsdelivr.net/npm/mathlive@0.110.0/fonts';

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
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
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
                    <button class="source-chip" [class.expanded]="expandedCitation === src.id" (click)="toggleCitation(src.id)">
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
                @if (expandedCitation) {
                  <div class="citation-detail">
                    @for (src of msg.sources; track src.id) {
                      @if (expandedCitation === src.id) {
                        <div class="citation-card">
                          <div class="citation-card-header">
                            <strong>{{ src.id }}</strong> — {{ src.filename }}
                            @if (src.section) {
                              <span> / {{ src.section }}</span>
                            }
                            @if (src.page) {
                              <span> (página {{ src.page }})</span>
                            }
                            <span class="citation-score" [style.color]="getSourceColor(src.score)">
                              {{ (src.score * 100).toFixed(0) }}%
                            </span>
                          </div>
                          <div class="citation-card-content">{{ src.content }}</div>
                        </div>
                      }
                    }
                  </div>
                }
              }
            </div>
          }
        </div>
        <div class="input-area">
          <math-field
            #mathField
            id="chat-mathfield"
            virtual-keyboard-mode="auto"
            smart-fence
            smart-superscript
            (keydown.enter)="sendMessage()"
            (input)="onMathInput()"
            class="chat-mathfield"
            placeholder="Escribi tu pregunta de matematica..."
          ></math-field>
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
    .source-chip.expanded { background: rgba(200, 170, 118, 0.15); border-color: #c8aa76; color: #c8aa76; }
    .source-page { color: var(--accent); font-weight: 600; }
    .source-section { color: var(--text-secondary); font-style: italic; max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .citation-detail { margin-top: 8px; }
    .citation-card { background: rgba(0, 0, 0, 0.2); border: 1px solid var(--border); border-radius: 8px; padding: 10px 12px; margin-top: 6px; }
    .citation-card-header { font-size: 11px; color: var(--text-secondary); margin-bottom: 6px; display: flex; align-items: center; flex-wrap: wrap; }
    .citation-score { margin-left: auto; font-weight: 700; font-size: 12px; }
    .citation-card-content { font-size: 12px; color: var(--text-secondary); line-height: 1.5; font-style: italic; }
    .input-area { padding: var(--space-md); display: flex; gap: var(--space-sm); border-top: 1px solid var(--border); align-items: center; }
    .chat-mathfield { flex: 1; min-height: 48px; padding: 0.5rem; border-radius: var(--radius-md); border: 1px solid var(--border); background: var(--input-bg); font-size: 1.1rem; }
    .input-area:focus-within .chat-mathfield { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-muted); }
    :host ::ng-deep #chat-mathfield {
      --math-field-border: none;
      --math-field-border-radius: 0;
      --math-field-background: transparent;
      --math-field-color: var(--text);
      --math-field-placeholder-color: var(--text-tertiary);
      font-size: 1.1rem;
    }
    :host ::ng-deep .ML__virtual-keyboard {
      background: var(--surface) !important;
      border-radius: var(--radius-md) !important;
    }
  `]
})
export class ChatComponent implements AfterViewInit, OnDestroy {
  @ViewChild('mathField') mathFieldRef!: ElementRef<any>;

  messages = signal<ChatMsg[]>([]);
  currentSessionId = signal<string>('');
  newMessage = '';
  expandedCitation: string | null = null;

  private mf: any = null;

  constructor(private api: ApiService, private router: Router, private zone: NgZone) {}

  ngAfterViewInit() {
    setTimeout(() => {
      this.mf = this.mathFieldRef?.nativeElement;
      if (this.mf) {
        this.mf.addEventListener('input', this.handleInput);
      }
    }, 100);
  }

  ngOnDestroy() {
    if (this.mf) {
      this.mf.removeEventListener('input', this.handleInput);
    }
  }

  private handleInput = () => {
    this.zone.run(() => {
      if (this.mf) {
        this.newMessage = this.mf.value || '';
      }
    });
  };

  onMathInput() {
    if (this.mf) {
      this.newMessage = this.mf.value || '';
    }
  }

  toggleCitation(id: string) {
    this.expandedCitation = this.expandedCitation === id ? null : id;
  }

  getSourceColor(score: number): string {
    if (score >= 0.9) return '#4caf50';
    if (score >= 0.7) return '#ff9800';
    return '#9e9e9e';
  }

  sendMessage() {
    if (!this.newMessage) return;
    const msg = this.newMessage;
    this.newMessage = '';
    if (this.mf) {
      this.mf.value = '';
    }
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
