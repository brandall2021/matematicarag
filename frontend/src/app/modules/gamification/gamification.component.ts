import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { GamificationService, GamificationSummary } from '../../core/services/gamification.service';

@Component({
  selector: 'app-gamification',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatProgressBarModule],
  template: `
    <div class="gamification-container">
      <h1>Logros</h1>

      @if (summary(); as s) {
        <div class="stats-grid">
          <div class="stat-card">
            <mat-icon>stars</mat-icon>
            <div class="stat-value">{{ s.points }}</div>
            <div class="stat-label">Puntos</div>
          </div>
          <div class="stat-card">
            <mat-icon>emoji_events</mat-icon>
            <div class="stat-value">{{ s.level }} — {{ s.level_name }}</div>
            <div class="stat-label">Nivel</div>
          </div>
          <div class="stat-card">
            <mat-icon>local_fire_department</mat-icon>
            <div class="stat-value">{{ s.current_streak }}</div>
            <div class="stat-label">Racha actual</div>
          </div>
          <div class="stat-card">
            <mat-icon>trending_up</mat-icon>
            <div class="stat-value">{{ s.best_streak }}</div>
            <div class="stat-label">Mejor racha</div>
          </div>
        </div>

        <div class="level-bar">
          <div class="level-label">Progreso al nivel {{ s.level + 1 }}</div>
          <mat-progress-bar mode="determinate"
            [value]="levelProgress(s)"
            max="100"></mat-progress-bar>
          <div class="level-hint">{{ s.points }} / {{ s.next_level_points }} puntos</div>
        </div>

        <h2>Medallas</h2>
        <div class="achievements-grid">
          @for (a of s.achievements; track a.id) {
            <div class="achievement-card" [class.locked]="!a.unlocked">
              <mat-icon>{{ a.icon }}</mat-icon>
              <div class="ach-title">{{ a.title }}</div>
              <div class="ach-desc">{{ a.description }}</div>
              @if (a.unlocked) {
                <div class="ach-points">+{{ a.points }} pts</div>
              } @else {
                <div class="ach-locked-label">Bloqueado</div>
              }
            </div>
          }
        </div>

        @if (s.recent_activities.length > 0) {
          <h2>Actividad reciente</h2>
          <div class="activity-list">
            @for (act of s.recent_activities; track act.created_at) {
              <div class="activity-item">
                <span>{{ act.source.replace('_', ' ') }}</span>
                <span class="activity-points">+{{ act.points }}</span>
              </div>
            }
          </div>
        }
      } @else {
        <div class="empty">Cargando tus logros...</div>
      }
    </div>
  `,
  styles: [`
    .gamification-container { padding: 1.5rem; max-width: 900px; margin: 0 auto; }
    .gamification-container h1 { color: var(--accent); font-family: var(--font-serif); margin: 0 0 1rem 0; }
    .gamification-container h2 { color: var(--accent); font-family: var(--font-serif); margin: 1.5rem 0 0.75rem; }
    .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 1rem; margin: 1rem 0; }
    .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 1rem; text-align: center; }
    .stat-card mat-icon { color: var(--accent); font-size: 28px; width: 28px; height: 28px; }
    .stat-value { font-size: 1.4rem; font-weight: 700; margin-top: 0.25rem; }
    .stat-label { color: var(--text-secondary); font-size: 0.8rem; }
    .level-bar { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 1rem; margin-bottom: 1rem; }
    .level-label { font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 0.5rem; }
    .level-hint { font-size: 0.75rem; color: var(--text-tertiary); margin-top: 0.35rem; text-align: right; }
    .achievements-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 1rem; }
    .achievement-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 1rem; text-align: center; }
    .achievement-card mat-icon { font-size: 40px; width: 40px; height: 40px; color: var(--accent); }
    .achievement-card.locked { opacity: 0.45; filter: grayscale(1); }
    .ach-title { font-weight: 600; margin-top: 0.5rem; }
    .ach-desc { font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem; }
    .ach-points { font-size: 0.8rem; color: var(--accent-text); font-weight: 600; margin-top: 0.5rem; }
    .ach-locked-label { font-size: 0.7rem; color: var(--text-tertiary); margin-top: 0.5rem; }
    .activity-list { display: flex; flex-direction: column; gap: 0.4rem; }
    .activity-item { display: flex; justify-content: space-between; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 0.5rem 0.75rem; font-size: 0.85rem; }
    .activity-points { color: var(--accent-text); font-weight: 600; }
    .empty { color: var(--text-secondary); padding: 2rem; text-align: center; }
  `]
})
export class GamificationComponent implements OnInit {
  summary = signal<GamificationSummary | null>(null);

  constructor(private gamificationService: GamificationService) {}

  ngOnInit(): void {
    this.gamificationService.getSummary().subscribe({
      next: s => this.summary.set(s),
      error: () => this.summary.set(null),
    });
  }

  levelProgress(s: GamificationSummary): number {
    const raw = s.points - (s.level - 1) * 100;
    return Math.min(100, Math.max(0, raw));
  }
}
