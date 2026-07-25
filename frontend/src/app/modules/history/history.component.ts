import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';

@Component({
  selector: 'app-history',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="container">
      <h1>Historial</h1>
      @for (entry of history(); track entry.id) {
        <div class="history-item">
          <h3>{{ entry.title }}</h3>
          <p>{{ entry.messages }} mensajes - {{ entry.createdAt | date:'short' }}</p>
        </div>
      } @empty {
        <div class="empty-state"><p>No hay historial aun.</p></div>
      }
    </div>
  `,
  styles: [`
    .container { padding: 2rem; max-width: 800px; margin: 0 auto; background: var(--bg); color: var(--text); min-height: 100vh; }
    h1 { color: var(--accent); }
    .history-item { background: var(--surface); padding: 1rem; border-radius: 8px; margin-bottom: 0.5rem; }
    .empty-state { text-align: center; margin-top: 3rem; }
  `]
})
export class HistoryComponent implements OnInit {
  history = signal<any[]>([]);
  constructor(private api: ApiService) {}
  ngOnInit() { this.api.getHistory().subscribe(h => this.history.set(h)); }
}
