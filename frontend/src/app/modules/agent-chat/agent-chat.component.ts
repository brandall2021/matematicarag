import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

interface AgentToolCall {
  tool_name: string;
  input: Record<string, any>;
  output?: Record<string, any>;
  duration_ms?: number;
  error?: string;
}

interface AgentStep {
  step_number: number;
  description: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  tool_calls: AgentToolCall[];
}

interface AgentLearning {
  topics_covered: string[];
  mastery_delta?: number;
  new_weak_areas?: string[];
  new_strong_areas?: string[];
}

interface AgentMsg {
  id: string;
  role: 'USER' | 'ASSISTANT';
  content: string;
  steps?: AgentStep[];
  learning?: AgentLearning;
  citations?: string[];
  sources?: { id: string; content: string; score: number; filename: string }[];
  confidence?: number;
  strategy?: string;
}

type Phase = 'idle' | 'thinking' | 'done';

@Component({
  selector: 'app-agent-chat',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule],
  template: `
    <div class="agent-container">
      <div class="agent-header">
        <div class="agent-title">
          <mat-icon class="agent-icon">smart_toy</mat-icon>
          <span>Agente Pedagógico</span>
        </div>
        <span class="agent-badge">orquestador</span>
      </div>

      <div class="messages">
        @if (messages().length === 0) {
          <div class="welcome">
            <mat-icon class="welcome-icon">auto_awesome</mat-icon>
            <p>Hacé una pregunta de matemática. El agente va a analizar tu intención, buscar recursos y armar una respuesta pedagógica.</p>
            <div class="suggestions">
              @for (s of suggestions; track s) {
                <button class="suggestion-chip" (click)="sendSuggestion(s)">{{ s }}</button>
              }
            </div>
          </div>
        }

        @for (msg of messages(); track msg.id) {
          <div class="message" [class.user]="msg.role === 'USER'" [class.assistant]="msg.role === 'ASSISTANT'">
            @if (msg.role === 'ASSISTANT') {
              <div class="msg-badge">
                <mat-icon class="msg-badge-icon">smart_toy</mat-icon>
                Agente
                @if (msg.strategy) {
                  <span class="strategy-tag">{{ msg.strategy }}</span>
                }
                @if (msg.confidence !== undefined) {
                  <span class="confidence-tag" [style.color]="confidenceColor(msg.confidence)">
                    {{ (msg.confidence * 100).toFixed(0) }}%
                  </span>
                }
              </div>
            }

            <div class="msg-content">{{ msg.content }}</div>

            @if (msg.steps && msg.steps.length > 0) {
              <div class="steps">
                @for (step of msg.steps; track step.step_number) {
                  <details class="step">
                    <summary class="step-header">
                      <mat-icon class="step-icon">check_circle</mat-icon>
                      <span class="step-num">Paso {{ step.step_number }}:</span>
                      {{ step.description }}
                    </summary>
                    @for (tc of step.tool_calls; track tc.tool_name) {
                      <div class="tool-call">
                        <div class="tool-call-header">
                          <mat-icon class="tool-icon">build</mat-icon>
                          <code>{{ tc.tool_name }}</code>
                          @if (tc.duration_ms) {
                            <span class="tool-duration">{{ tc.duration_ms }}ms</span>
                          }
                          @if (tc.error) {
                            <span class="tool-error">error</span>
                          }
                        </div>
                        @if (tc.input && !tc.error) {
                          <div class="tool-detail">
                            <span class="detail-label">input:</span>
                            <pre>{{ tc.input | json }}</pre>
                          </div>
                        }
                        @if (tc.output && !tc.error) {
                          <div class="tool-detail">
                            <span class="detail-label">output:</span>
                            <pre>{{ tc.output | json }}</pre>
                          </div>
                        }
                        @if (tc.error) {
                          <div class="tool-detail error">
                            <span class="detail-label">error:</span>
                            <pre>{{ tc.error }}</pre>
                          </div>
                        }
                      </div>
                    }
                  </details>
                }
              </div>
            }

            @if (msg.sources && msg.sources.length > 0) {
              <div class="sources">
                <span class="sources-label"><mat-icon>menu_book</mat-icon> Fuentes:</span>
                @for (src of msg.sources; track src.id) {
                  <button class="source-chip" (click)="toggleSource(msg.id + src.id)" [class.expanded]="expandedSource() === msg.id + src.id">
                    <mat-icon class="source-icon">description</mat-icon>
                    {{ src.filename }}
                    <span class="source-score" [style.color]="confidenceColor(src.score)">
                      {{ (src.score * 100).toFixed(0) }}%
                    </span>
                  </button>
                }
              </div>
              @for (src of msg.sources; track src.id) {
                @if (expandedSource() === msg.id + src.id) {
                  <div class="source-detail">
                    <div class="source-card">
                      <div class="source-card-header">{{ src.id }} — {{ src.filename }}</div>
                      <div class="source-card-content">{{ src.content }}</div>
                    </div>
                  </div>
                }
              }
            }

            @if (msg.learning) {
              <details class="learning">
                <summary class="learning-header">
                  <mat-icon>trending_up</mat-icon> Aprendizaje
                </summary>
                <div class="learning-body">
                  <div class="learning-row"><span class="detail-label">Temas:</span> {{ msg.learning.topics_covered.join(', ') }}</div>
                  @if (msg.learning.mastery_delta !== undefined) {
                    <div class="learning-row"><span class="detail-label">Delta maestría:</span> {{ msg.learning.mastery_delta > 0 ? '+' : '' }}{{ (msg.learning.mastery_delta * 100).toFixed(1) }}%</div>
                  }
                  @if (msg.learning.new_weak_areas?.length) {
                    <div class="learning-row weak"><span class="detail-label">Áreas débiles:</span> {{ msg.learning.new_weak_areas?.join(', ') }}</div>
                  }
                  @if (msg.learning.new_strong_areas?.length) {
                    <div class="learning-row strong"><span class="detail-label">Áreas fuertes:</span> {{ msg.learning.new_strong_areas?.join(', ') }}</div>
                  }
                </div>
              </details>
            }
          </div>
        }

        @if (phase() === 'thinking') {
          <div class="thinking">
            <div class="thinking-dots">
              <span class="dot"></span><span class="dot"></span><span class="dot"></span>
            </div>
            <span>Analizando consulta pedagógica...</span>
          </div>
        }
      </div>

      <div class="input-area">
        <input [(ngModel)]="newMessage" (keydown.enter)="sendMessage()" placeholder="Preguntale al agente..." class="agent-input" [disabled]="phase() === 'thinking'">
        <button mat-raised-button color="primary" (click)="sendMessage()" [disabled]="!newMessage || phase() === 'thinking'">
          <mat-icon>send</mat-icon>
        </button>
      </div>
    </div>
  `,
  styles: [`
    .agent-container { display: flex; flex-direction: column; height: 100%; }
    .agent-header { display: flex; align-items: center; gap: 0.75rem; padding: 0.75rem 1.25rem; border-bottom: 1px solid var(--border); }
    .agent-title { display: flex; align-items: center; gap: 0.5rem; font-size: 1rem; font-weight: 700; color: var(--text); }
    .agent-icon { color: #c8aa76; }
    .agent-badge { font-size: 0.65rem; padding: 0.15rem 0.5rem; border-radius: 4px; background: rgba(200, 170, 118, 0.15); color: #c8aa76; text-transform: uppercase; letter-spacing: 0.5px; font-weight: 600; }
    .messages { flex: 1; overflow-y: auto; padding: 1rem; }
    .welcome { text-align: center; padding: 2rem 1rem; color: var(--text-secondary); max-width: 480px; margin: 0 auto; }
    .welcome-icon { font-size: 2.5rem; width: 2.5rem; height: 2.5rem; color: var(--accent); margin-bottom: 0.5rem; }
    .welcome p { font-size: 0.9rem; line-height: 1.6; margin-bottom: 1.25rem; }
    .suggestions { display: flex; flex-wrap: wrap; gap: 0.5rem; justify-content: center; }
    .suggestion-chip { padding: 0.4rem 0.8rem; border-radius: 8px; border: 1px solid var(--border); background: var(--surface); color: var(--text-secondary); font-size: 0.8rem; cursor: pointer; transition: all 0.15s; }
    .suggestion-chip:hover { border-color: var(--accent); color: var(--accent); background: rgba(200, 170, 118, 0.08); }
    .message { margin-bottom: 1rem; max-width: 78%; }
    .message.user { margin-left: auto; }
    .message.assistant { margin-right: auto; }
    .msg-badge { display: flex; align-items: center; gap: 0.4rem; font-size: 0.7rem; color: #c8aa76; font-weight: 600; margin-bottom: 0.3rem; }
    .msg-badge-icon { font-size: 14px; width: 14px; height: 14px; }
    .strategy-tag { font-size: 0.65rem; padding: 0.1rem 0.4rem; border-radius: 3px; background: rgba(200, 170, 118, 0.12); color: #c8aa76; font-weight: 500; }
    .confidence-tag { margin-left: auto; font-size: 0.65rem; font-weight: 700; }
    .msg-content { padding: 0.75rem 1rem; border-radius: 12px; line-height: 1.6; white-space: pre-wrap; word-break: break-word; }
    .message.user .msg-content { background: var(--accent); color: var(--bg); }
    .message.assistant .msg-content { background: var(--border); color: var(--text); }
    .steps { margin-top: 0.5rem; display: flex; flex-direction: column; gap: 0.3rem; }
    .step { background: rgba(0,0,0,0.15); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
    .step-header { display: flex; align-items: center; gap: 0.4rem; padding: 0.4rem 0.6rem; font-size: 0.75rem; cursor: pointer; user-select: none; color: var(--text-secondary); }
    .step-header::-webkit-details-marker { display: none; }
    .step-icon { font-size: 14px; width: 14px; height: 14px; color: #4caf50; }
    .step-num { font-weight: 600; color: var(--text); }
    .tool-call { padding: 0.3rem 0.6rem 0.3rem 1.2rem; border-top: 1px solid var(--border); }
    .tool-call-header { display: flex; align-items: center; gap: 0.35rem; font-size: 0.7rem; }
    .tool-icon { font-size: 12px; width: 12px; height: 12px; color: var(--text-secondary); }
    .tool-call-header code { background: rgba(0,0,0,0.2); padding: 0.1rem 0.35rem; border-radius: 3px; font-size: 0.7rem; }
    .tool-duration { margin-left: auto; color: var(--text-secondary); font-size: 0.65rem; }
    .tool-error { margin-left: auto; color: #f44336; font-size: 0.65rem; font-weight: 600; background: rgba(244,67,54,0.1); padding: 0.05rem 0.3rem; border-radius: 3px; }
    .tool-detail { margin-top: 0.2rem; font-size: 0.65rem; }
    .tool-detail pre { background: rgba(0,0,0,0.2); padding: 0.3rem 0.5rem; border-radius: 4px; margin: 0.15rem 0 0; overflow-x: auto; max-height: 120px; font-size: 0.65rem; line-height: 1.4; }
    .tool-detail.error pre { color: #f44336; }
    .detail-label { color: var(--text-secondary); font-weight: 600; font-size: 0.65rem; }
    .sources { display: flex; flex-wrap: wrap; gap: 0.4rem; margin-top: 0.5rem; align-items: center; }
    .sources-label { display: inline-flex; align-items: center; gap: 0.2rem; font-size: 0.7rem; color: var(--text-secondary); font-weight: 600; }
    .sources-label mat-icon { font-size: 13px; width: 13px; height: 13px; }
    .source-chip { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.2rem 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface); color: var(--text-secondary); font-size: 0.72rem; cursor: pointer; transition: all 0.15s; }
    .source-chip:hover { border-color: var(--accent); color: var(--accent); }
    .source-chip.expanded { background: rgba(200,170,118,0.15); border-color: #c8aa76; color: #c8aa76; }
    .source-icon { font-size: 12px; width: 12px; height: 12px; }
    .source-score { font-weight: 700; font-size: 0.65rem; }
    .source-detail { margin-top: 6px; }
    .source-card { background: rgba(0,0,0,0.2); border: 1px solid var(--border); border-radius: 8px; padding: 8px 10px; }
    .source-card-header { font-size: 11px; color: var(--text-secondary); margin-bottom: 4px; font-weight: 600; }
    .source-card-content { font-size: 11px; color: var(--text-secondary); line-height: 1.5; font-style: italic; }
    .learning { margin-top: 0.5rem; background: rgba(76,175,80,0.08); border: 1px solid rgba(76,175,80,0.2); border-radius: 8px; overflow: hidden; }
    .learning-header { display: flex; align-items: center; gap: 0.3rem; padding: 0.4rem 0.6rem; font-size: 0.75rem; cursor: pointer; color: #4caf50; user-select: none; }
    .learning-header mat-icon { font-size: 14px; width: 14px; height: 14px; }
    .learning-body { padding: 0.3rem 0.6rem 0.5rem; font-size: 0.7rem; }
    .learning-row { margin-top: 0.2rem; }
    .learning-row.weak .detail-label { color: #ff9800; }
    .learning-row.strong .detail-label { color: #4caf50; }
    .thinking { display: flex; align-items: center; gap: 0.6rem; padding: 0.5rem 0; margin-bottom: 1rem; font-size: 0.8rem; color: var(--text-secondary); }
    .thinking-dots { display: flex; gap: 3px; }
    .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--accent); animation: pulse 1.2s infinite; }
    .dot:nth-child(2) { animation-delay: 0.2s; }
    .dot:nth-child(3) { animation-delay: 0.4s; }
    @keyframes pulse { 0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); } 40% { opacity: 1; transform: scale(1.1); } }
    .input-area { padding: 1rem; display: flex; gap: 0.5rem; border-top: 1px solid var(--border); }
    .agent-input { flex: 1; padding: 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--input-bg); color: var(--text); font-size: 1rem; outline: none; }
    .agent-input:focus { border-color: var(--accent); }
    .input-area button mat-icon { font-size: 18px; }
  `]
})
export class AgentChatComponent {
  messages = signal<AgentMsg[]>([]);
  sessionId = signal<string>('');
  phase = signal<Phase>('idle');
  newMessage = '';
  expandedSource = signal<string | null>(null);

  suggestions = [
    'Explicame cómo derivar x² + 3x',
    'Ayudame con la regla de la cadena',
    'Qué es una integral definida?',
    'Tenemos ejercicios de límites?',
  ];

  constructor(private api: ApiService, private router: Router) {}

  confidenceColor(score: number): string {
    if (score >= 0.9) return '#4caf50';
    if (score >= 0.7) return '#ff9800';
    return '#9e9e9e';
  }

  toggleSource(id: string) {
    this.expandedSource.update(v => v === id ? null : id);
  }

  sendSuggestion(text: string) {
    this.newMessage = text;
    this.sendMessage();
  }

  sendMessage() {
    if (!this.newMessage || this.phase() === 'thinking') return;
    const query = this.newMessage;
    this.newMessage = '';
    this.phase.set('thinking');
    this.messages.update(msgs => [...msgs, { id: 'temp-user', role: 'USER', content: query }]);
    this.api.agentChat(query, this.sessionId()).subscribe({
      next: res => {
        const msg: AgentMsg = {
          id: res.session_id || 'agent-' + Date.now(),
          role: 'ASSISTANT',
          content: res.response || res.content || '',
          steps: res.steps || [],
          learning: res.learning,
          citations: res.citations || [],
          sources: res.sources || [],
          confidence: res.confidence,
          strategy: res.strategy,
        };
        this.messages.update(msgs => [...msgs, msg]);
        this.sessionId.set(res.session_id || this.sessionId());
        this.phase.set('done');
      },
      error: err => {
        this.messages.update(msgs => [...msgs, {
          id: 'error-' + Date.now(),
          role: 'ASSISTANT',
          content: err.error?.error || 'Error al conectar con el agente pedagógico. Intentalo de nuevo.',
        }]);
        this.phase.set('done');
      }
    });
  }
}
