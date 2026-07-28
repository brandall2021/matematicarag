import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LearningService, StudentDashboard, ConceptMastery, AdaptiveRecommendation } from '../../core/services/learning.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';

@Component({
  selector: 'app-student-progress',
  standalone: true,
  imports: [CommonModule, MatButtonModule, MatIconModule, MatProgressBarModule],
  template: `
    <div class="progress-container">
      <div class="progress-header">
        <h1><mat-icon>trending_up</mat-icon> Mi Progreso</h1>
      </div>

      @if (loading()) {
        <div class="loading-box"><mat-icon>sync</mat-icon> Cargando...</div>
      }

      @if (error()) {
        <div class="error-box"><mat-icon>error</mat-icon> {{ error() }}</div>
      }

      @if (dashboard()) {
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-value">{{ (dashboard()!.profile.overall_level * 100).toFixed(0) }}%</div>
            <div class="stat-label">Nivel General</div>
            <mat-progress-bar mode="determinate" [value]="dashboard()!.profile.overall_level * 100" class="mastery-bar"></mat-progress-bar>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ dashboard()!.sessions_summary.total_exercises }}</div>
            <div class="stat-label">Ejercicios Resueltos</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ (dashboard()!.sessions_summary.correct_rate * 100).toFixed(0) }}%</div>
            <div class="stat-label">Precision</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ dashboard()!.sessions_summary.study_time_hours.toFixed(1) }}h</div>
            <div class="stat-label">Tiempo de Estudio</div>
          </div>
        </div>

        @if (recommendation()) {
          <div class="recommendation-card">
            <mat-icon>auto_awesome</mat-icon>
            <div>
              <strong>Recomendacion:</strong> {{ recommendation()!.concept_name }}
              <span class="rec-reason">{{ recommendation()!.reason }}</span>
            </div>
          </div>
        }

        <div class="section">
          <h2><mat-icon>psychology</mat-icon> Dominio por Concepto</h2>
          <div class="mastery-list">
            @for (entry of masteryList(); track entry.concept_id) {
              <div class="mastery-item">
                <div class="mastery-info">
                  <span class="mastery-name">{{ entry.concept_id }}</span>
                  <span class="mastery-status" [attr.data-status]="entry.status">{{ formatStatus(entry.status) }}</span>
                </div>
                <mat-progress-bar mode="determinate" [value]="entry.mastery * 100" [class]="getMasteryClass(entry.mastery)"></mat-progress-bar>
                <div class="mastery-detail">{{ (entry.mastery * 100).toFixed(0) }}% · {{ entry.correct }}/{{ entry.attempts }} correctos</div>
              </div>
            }
            @if (masteryList().length === 0) {
              <p class="empty-text">Aun no hay datos de dominio. Comienza a practicar!</p>
            }
          </div>
        </div>

        @if (dashboard()!.recent_errors && dashboard()!.recent_errors.length > 0) {
          <div class="section">
            <h2><mat-icon>bug_report</mat-icon> Errores Recientes</h2>
            <div class="errors-list">
              @for (err of dashboard()!.recent_errors.slice(0, 5); track err.id) {
                <div class="error-item">
                  <span class="error-type">{{ err.error_type }}</span>
                  <span class="error-concept">{{ err.concept_id }}</span>
                  <span class="error-count">x{{ err.count }}</span>
                </div>
              }
            </div>
          </div>
        }

        @if (dashboard()!.recommendations && dashboard()!.recommendations.length > 0) {
          <div class="section">
            <h2><mat-icon>tips_and_updates</mat-icon> Sugerencias</h2>
            <div class="rec-list">
              @for (rec of dashboard()!.recommendations; track $index) {
                <div class="rec-item">{{ rec }}</div>
              }
            </div>
          </div>
        }
      }
    </div>
  `,
  styles: [`
    .progress-container { padding: 1.5rem; max-width: 900px; margin: 0 auto; }
    .progress-header h1 { color: var(--accent); font-family: 'Newsreader', serif; margin: 0 0 1.5rem 0; display: flex; align-items: center; gap: 0.5rem; }
    .progress-header h1 mat-icon { font-size: 28px; width: 28px; height: 28px; }

    .loading-box, .error-box { display: flex; align-items: center; gap: 0.5rem; padding: 1rem; border-radius: 8px; margin-bottom: 1rem; }
    .loading-box { background: var(--surface); color: var(--text-secondary); }
    .error-box { background: rgba(244, 67, 54, 0.1); color: #ef5350; border: 1px solid rgba(244, 67, 54, 0.3); }

    .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem; margin-bottom: 1.5rem; }
    .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; text-align: center; }
    .stat-value { font-size: 1.8rem; font-weight: 700; color: var(--accent); }
    .stat-label { font-size: 0.78rem; color: var(--text-secondary); margin-top: 0.25rem; text-transform: uppercase; letter-spacing: 0.05em; }
    .mastery-bar { margin-top: 0.75rem; border-radius: 4px; }

    .recommendation-card { display: flex; align-items: flex-start; gap: 0.75rem; background: rgba(200, 170, 118, 0.1); border: 1px solid rgba(200, 170, 118, 0.25); border-radius: 10px; padding: 1rem; margin-bottom: 1.5rem; color: var(--text); }
    .recommendation-card mat-icon { color: #c8aa76; margin-top: 2px; }
    .rec-reason { display: block; font-size: 0.85rem; color: var(--text-secondary); margin-top: 0.25rem; }

    .section { margin-bottom: 1.5rem; }
    .section h2 { color: var(--accent); font-size: 1rem; margin: 0 0 0.75rem 0; display: flex; align-items: center; gap: 0.4rem; }
    .section h2 mat-icon { font-size: 20px; width: 20px; height: 20px; }

    .mastery-list { display: flex; flex-direction: column; gap: 0.75rem; }
    .mastery-item { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 0.75rem 1rem; }
    .mastery-info { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.4rem; }
    .mastery-name { font-weight: 600; font-size: 0.85rem; }
    .mastery-status { font-size: 0.72rem; padding: 0.15rem 0.5rem; border-radius: 4px; font-weight: 600; text-transform: uppercase; }
    .mastery-status[data-status="mastered"] { background: rgba(76, 175, 80, 0.15); color: #4caf50; }
    .mastery-status[data-status="developing"] { background: rgba(255, 193, 7, 0.15); color: #ffc107; }
    .mastery-status[data-status="learning"] { background: rgba(33, 150, 243, 0.15); color: #2196f3; }
    .mastery-status[data-status="not_started"] { background: rgba(158, 158, 158, 0.15); color: #9e9e9e; }
    .mastery-detail { font-size: 0.72rem; color: var(--text-secondary); margin-top: 0.3rem; }
    :host ::ng-deep .mastery-list mat-progress-bar { border-radius: 4px; }
    :host ::ng-deep mat-progress-bar.bar-green .mat-mdc-progress-bar-fill::after { background: #4caf50; }
    :host ::ng-deep mat-progress-bar.bar-yellow .mat-mdc-progress-bar-fill::after { background: #ffc107; }
    :host ::ng-deep mat-progress-bar.bar-red .mat-mdc-progress-bar-fill::after { background: #ef5350; }

    .errors-list { display: flex; flex-direction: column; gap: 0.4rem; }
    .error-item { display: flex; align-items: center; gap: 0.75rem; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.6rem 0.85rem; font-size: 0.85rem; }
    .error-type { font-weight: 600; color: #ef5350; }
    .error-concept { color: var(--text-secondary); flex: 1; }
    .error-count { color: var(--text-secondary); font-size: 0.78rem; }

    .rec-list { display: flex; flex-direction: column; gap: 0.4rem; }
    .rec-item { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.6rem 0.85rem; font-size: 0.85rem; color: var(--text-secondary); }

    .empty-text { color: var(--text-secondary); font-size: 0.85rem; text-align: center; padding: 1rem; }
  `]
})
export class StudentProgressComponent implements OnInit {
  dashboard = signal<StudentDashboard | null>(null);
  recommendation = signal<AdaptiveRecommendation | null>(null);
  masteryList = signal<ConceptMastery[]>([]);
  loading = signal(true);
  error = signal('');

  constructor(private learning: LearningService) {}

  ngOnInit() {
    this.learning.getMyProgress().subscribe({
      next: (d) => {
        this.dashboard.set(d);
        const list = Object.values(d.mastery_map || {});
        list.sort((a, b) => b.mastery - a.mastery);
        this.masteryList.set(list);
        this.loading.set(false);
      },
      error: (e) => {
        this.error.set('Error al cargar progreso');
        this.loading.set(false);
      }
    });

    this.learning.getRecommendation().subscribe({
      next: (r) => this.recommendation.set(r),
      error: () => {}
    });
  }

  formatStatus(status: string): string {
    const map: Record<string, string> = {
      mastered: 'Dominado', developing: 'En progreso', learning: 'Aprendiendo', not_started: 'Sin iniciar'
    };
    return map[status] || status;
  }

  getMasteryClass(mastery: number): string {
    if (mastery >= 0.7) return 'bar-green';
    if (mastery >= 0.3) return 'bar-yellow';
    return 'bar-red';
  }
}
