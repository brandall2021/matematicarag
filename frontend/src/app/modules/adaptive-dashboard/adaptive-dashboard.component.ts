import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

interface ConceptMastery {
  concept_id: string;
  concept_name: string;
  mastery: number;
  status: string;
}

interface LearnerProfile {
  overall_mastery: number;
  concepts: ConceptMastery[];
  weak_concepts: string[];
  strong_concepts: string[];
}

interface LearningRecommendation {
  action: string;
  concept_id: string;
  concept_name: string;
  reason: string;
  difficulty: number;
}

interface LearningPathStep {
  step: number;
  concept_id: string;
  concept_name: string;
  status: 'pending' | 'in_progress' | 'completed';
  prerequisites: string[];
}

@Component({
  selector: 'app-adaptive-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule],
  template: `
    <div class="dashboard">
      <div class="header">
        <h1><mat-icon>insights</mat-icon> Panel de Aprendizaje</h1>
        <span class="subtitle">Seguimiento adaptativo personalizado</span>
      </div>

      @if (loading()) {
        <div class="loading-box"><mat-icon>sync</mat-icon> Cargando tu perfil de aprendizaje...</div>
      }

      @if (error()) {
        <div class="error-box"><mat-icon>error</mat-icon> {{ error() }}</div>
      }

      @if (profile()) {
        <div class="top-row">
          <div class="mastery-card">
            <div class="circular-progress">
              <svg viewBox="0 0 120 120" class="progress-ring">
                <circle cx="60" cy="60" r="52" class="ring-bg" />
                <circle cx="60" cy="60" r="52" class="ring-fill"
                  [style.stroke-dasharray]="326.73"
                  [style.stroke-dashoffset]="326.73 - (326.73 * profile()!.overall_mastery) / 100"
                />
              </svg>
              <span class="progress-text">{{ profile()!.overall_mastery.toFixed(0) }}%</span>
            </div>
            <div class="mastery-label">Dominio General</div>
          </div>

          <div class="stats-row">
            <div class="stat-chip">
              <mat-icon class="stat-icon strong">check_circle</mat-icon>
              <span>{{ profile()!.strong_concepts.length }} fuertes</span>
            </div>
            <div class="stat-chip">
              <mat-icon class="stat-icon weak">warning</mat-icon>
              <span>{{ profile()!.weak_concepts.length }} debiles</span>
            </div>
            <div class="stat-chip">
              <mat-icon class="stat-icon total">psychology</mat-icon>
              <span>{{ profile()!.concepts.length }} conceptos</span>
            </div>
          </div>
        </div>

        <div class="grid-2col">
          <div class="section">
            <h2><mat-icon>psychology</mat-icon> Dominio por Concepto</h2>
            <div class="concept-list">
              @for (c of profile()!.concepts; track c.concept_id) {
                <div class="concept-row">
                  <div class="concept-info">
                    <span class="concept-name">{{ c.concept_name }}</span>
                    <span class="concept-badge" [style.background]="confidenceColorBg(c.mastery)" [style.color]="confidenceColor(c.mastery)">
                      {{ (c.mastery * 100).toFixed(0) }}%
                    </span>
                  </div>
                  <div class="concept-bar-track">
                    <div class="concept-bar-fill" [style.width.%]="c.mastery * 100" [style.background]="confidenceColor(c.mastery)"></div>
                  </div>
                </div>
              }
              @if (profile()!.concepts.length === 0) {
                <p class="empty">Aun no hay datos de concepto. Practica para ver tu progreso.</p>
              }
            </div>
          </div>

          <div class="section">
            <h2><mat-icon>auto_awesome</mat-icon> Accion Recomendada</h2>
            @if (recommendation()) {
              <div class="rec-card">
                <div class="rec-header">
                  <mat-icon class="rec-icon">touch_app</mat-icon>
                  <span class="rec-action">{{ recommendation()!.action }}</span>
                </div>
                <p class="rec-concept">{{ recommendation()!.concept_name }}</p>
                <p class="rec-reason">{{ recommendation()!.reason }}</p>
                <button mat-raised-button color="primary" class="rec-button" (click)="startLearning(recommendation()!.concept_id)">
                  <mat-icon>play_arrow</mat-icon> Practicar {{ recommendation()!.concept_name }}
                </button>
              </div>
            } @else {
              <div class="rec-card empty-rec">
                <mat-icon>check_circle</mat-icon>
                <p>No hay recomendaciones pendientes. Sigue practicando!</p>
              </div>
            }
          </div>
        </div>

        @if (profile()!.weak_concepts.length > 0) {
          <div class="section">
            <h2><mat-icon>error_outline</mat-icon> Conceptos Debiles</h2>
            <div class="chips">
              @for (w of profile()!.weak_concepts; track w) {
                <span class="chip weak-chip">{{ w }}</span>
              }
            </div>
          </div>
        }

        @if (learningPath().length > 0) {
          <div class="section">
            <h2><mat-icon>route</mat-icon> Ruta de Aprendizaje</h2>
            <div class="path-list">
              @for (step of learningPath(); track step.step) {
                <div class="path-step" [class.completed]="step.status === 'completed'" [class.active]="step.status === 'in_progress'">
                  <div class="step-indicator">
                    @if (step.status === 'completed') {
                      <mat-icon class="step-icon done">check_circle</mat-icon>
                    } @else if (step.status === 'in_progress') {
                      <mat-icon class="step-icon current">play_circle</mat-icon>
                    } @else {
                      <span class="step-num">{{ step.step }}</span>
                    }
                  </div>
                  <div class="step-body">
                    <span class="step-name">{{ step.concept_name }}</span>
                    @if (step.prerequisites.length > 0) {
                      <span class="step-prereqs">Prerrequisitos: {{ step.prerequisites.join(', ') }}</span>
                    }
                  </div>
                </div>
              }
            </div>
          </div>
        }
      }
    </div>
  `,
  styles: [`
    .dashboard { padding: 1.5rem; max-width: 960px; margin: 0 auto; }
    .header { margin-bottom: 1.5rem; }
    .header h1 { color: var(--accent); font-family: 'Newsreader', serif; margin: 0; display: flex; align-items: center; gap: 0.5rem; }
    .header h1 mat-icon { font-size: 28px; width: 28px; height: 28px; }
    .subtitle { font-size: 0.85rem; color: var(--text-secondary); display: block; margin-top: 0.25rem; }

    .loading-box, .error-box { display: flex; align-items: center; gap: 0.5rem; padding: 1rem; border-radius: 8px; margin-bottom: 1rem; }
    .loading-box { background: var(--surface); color: var(--text-secondary); }
    .error-box { background: rgba(244, 67, 54, 0.1); color: #ef5350; border: 1px solid rgba(244, 67, 54, 0.3); }

    .top-row { display: flex; align-items: center; gap: 1.5rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
    .mastery-card { display: flex; flex-direction: column; align-items: center; gap: 0.5rem; }
    .circular-progress { position: relative; width: 120px; height: 120px; }
    .progress-ring { width: 100%; height: 100%; transform: rotate(-90deg); }
    .ring-bg { fill: none; stroke: var(--border); stroke-width: 8; }
    .ring-fill { fill: none; stroke: var(--accent); stroke-width: 8; stroke-linecap: round; transition: stroke-dashoffset 0.5s ease; }
    .progress-text { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; font-size: 1.5rem; font-weight: 700; color: var(--accent); }
    .mastery-label { font-size: 0.78rem; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.05em; }
    .stats-row { display: flex; gap: 0.75rem; flex-wrap: wrap; }
    .stat-chip { display: flex; align-items: center; gap: 0.35rem; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.5rem 0.75rem; font-size: 0.82rem; color: var(--text); }
    .stat-icon { font-size: 18px; width: 18px; height: 18px; }
    .stat-icon.strong { color: #4caf50; }
    .stat-icon.weak { color: #ff9800; }
    .stat-icon.total { color: #2196f3; }

    .grid-2col { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-bottom: 1.5rem; }
    @media (max-width: 700px) { .grid-2col { grid-template-columns: 1fr; } }

    .section { margin-bottom: 1.5rem; }
    .section h2 { color: var(--accent); font-size: 1rem; margin: 0 0 0.75rem 0; display: flex; align-items: center; gap: 0.4rem; }
    .section h2 mat-icon { font-size: 20px; width: 20px; height: 20px; }

    .concept-list { display: flex; flex-direction: column; gap: 0.6rem; }
    .concept-row { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 0.65rem 1rem; }
    .concept-info { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.35rem; }
    .concept-name { font-weight: 600; font-size: 0.85rem; color: var(--text); }
    .concept-badge { font-size: 0.7rem; font-weight: 700; padding: 0.1rem 0.5rem; border-radius: 4px; }
    .concept-bar-track { height: 6px; background: var(--border); border-radius: 3px; overflow: hidden; }
    .concept-bar-fill { height: 100%; border-radius: 3px; transition: width 0.4s ease; }

    .rec-card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; }
    .rec-card.empty-rec { display: flex; align-items: center; gap: 0.5rem; color: var(--text-secondary); font-size: 0.85rem; }
    .rec-card.empty-rec mat-icon { color: #4caf50; }
    .rec-header { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem; }
    .rec-icon { color: var(--accent); }
    .rec-action { font-size: 0.8rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: var(--accent); }
    .rec-concept { font-size: 1.1rem; font-weight: 600; color: var(--text); margin: 0 0 0.35rem; }
    .rec-reason { font-size: 0.85rem; color: var(--text-secondary); margin: 0 0 1rem; line-height: 1.5; }
    .rec-button { width: 100%; }

    .chips { display: flex; flex-wrap: wrap; gap: 0.5rem; }
    .chip { padding: 0.35rem 0.8rem; border-radius: 20px; font-size: 0.82rem; font-weight: 500; }
    .weak-chip { background: rgba(255, 152, 0, 0.12); color: #ff9800; border: 1px solid rgba(255, 152, 0, 0.25); }

    .path-list { display: flex; flex-direction: column; gap: 0.5rem; }
    .path-step { display: flex; align-items: flex-start; gap: 0.75rem; background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 0.75rem 1rem; }
    .path-step.completed { opacity: 0.65; }
    .path-step.active { border-color: var(--accent); }
    .step-indicator { flex-shrink: 0; width: 28px; height: 28px; display: flex; align-items: center; justify-content: center; margin-top: 1px; }
    .step-icon { font-size: 22px; width: 22px; height: 22px; }
    .step-icon.done { color: #4caf50; }
    .step-icon.current { color: var(--accent); }
    .step-num { width: 24px; height: 24px; border-radius: 50%; background: var(--border); color: var(--text-secondary); display: flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 700; }
    .step-body { display: flex; flex-direction: column; }
    .step-name { font-size: 0.9rem; font-weight: 600; color: var(--text); }
    .step-prereqs { font-size: 0.72rem; color: var(--text-secondary); margin-top: 0.15rem; }

    .empty { color: var(--text-secondary); font-size: 0.85rem; text-align: center; padding: 1rem; }
  `]
})
export class AdaptiveDashboardComponent implements OnInit {
  profile = signal<LearnerProfile | null>(null);
  recommendation = signal<LearningRecommendation | null>(null);
  learningPath = signal<LearningPathStep[]>([]);
  loading = signal(true);
  error = signal('');

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.api.getLearnerProfile().subscribe({
      next: (data) => {
        this.profile.set({
          overall_mastery: data.overall_mastery ?? 0,
          concepts: (data.concepts || []).sort((a: ConceptMastery, b: ConceptMastery) => b.mastery - a.mastery),
          weak_concepts: data.weak_concepts || [],
          strong_concepts: data.strong_concepts || [],
        });
        this.loading.set(false);
      },
      error: () => {
        this.error.set('Error al cargar el perfil de aprendizaje.');
        this.loading.set(false);
      }
    });

    this.api.getLearningRecommendation().subscribe({
      next: (data) => this.recommendation.set(data),
      error: () => {}
    });

    this.api.getLearningPath().subscribe({
      next: (data) => this.learningPath.set(Array.isArray(data) ? data : data.steps || []),
      error: () => {}
    });
  }

  confidenceColor(mastery: number): string {
    if (mastery >= 0.7) return '#4caf50';
    if (mastery >= 0.4) return '#ff9800';
    return '#f44336';
  }

  confidenceColorBg(mastery: number): string {
    if (mastery >= 0.7) return 'rgba(76, 175, 80, 0.12)';
    if (mastery >= 0.4) return 'rgba(255, 152, 0, 0.12)';
    return 'rgba(244, 67, 54, 0.12)';
  }

  startLearning(conceptId: string) {
  }
}
