import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatTabsModule } from '@angular/material/tabs';
import { MatChipsModule } from '@angular/material/chips';
import { MatListModule } from '@angular/material/list';
import { AssessmentService, StudentAnalytics, RecoveryPlan, AcademicAlert } from '../../core/services/assessment.service';
import { LearningService, StudentProfile, ConceptMastery } from '../../core/services/learning.service';

@Component({
  selector: 'app-analytics',
  standalone: true,
  imports: [
    CommonModule, MatCardModule, MatButtonModule, MatIconModule,
    MatProgressBarModule, MatTabsModule, MatChipsModule, MatListModule
  ],
  template: `
    <div class="analytics-container">
      <div class="analytics-header">
        <h2>Panel de Analíticas</h2>
        <div class="analytics-actions">
          <button mat-raised-button color="primary" (click)="exportCSV()">
            <mat-icon>download</mat-icon> Exportar CSV
          </button>
        </div>
      </div>

      @if (loading()) {
        <mat-progress-bar mode="indeterminate"></mat-progress-bar>
      } @else {
        <mat-tab-group>
          <mat-tab label="Resumen">
            <div class="analytics-grid">
              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>assessment</mat-icon>
                  <mat-card-title>Evaluaciones</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value">{{ analytics()?.total_assessments || 0 }}</div>
                  <div class="stat-label">Total realizadas</div>
                  <div class="stat-detail">
                    {{ analytics()?.passed_assessments || 0 }} aprobadas
                    ({{ ((analytics()?.passed_assessments || 0) / (analytics()?.total_assessments || 1) * 100) | number:'1.0-0' }}%)
                  </div>
                </mat-card-content>
              </mat-card>

              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>grade</mat-icon>
                  <mat-card-title>Promedio</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value">{{ ((analytics()?.average_score || 0) * 100) | number:'1.0-0' }}%</div>
                  <mat-progress-bar [value]="(analytics()?.average_score || 0) * 100"></mat-progress-bar>
                </mat-card-content>
              </mat-card>

              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>emoji_events</mat-icon>
                  <mat-card-title>Competencia</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value competency-badge" [attr.data-level]="analytics()?.competency_level">
                    {{ analytics()?.competency_level | titlecase }}
                  </div>
                  <mat-progress-bar [value]="(analytics()?.competency_score || 0) * 100"></mat-progress-bar>
                </mat-card-content>
              </mat-card>

              <mat-card class="stat-card">
                <mat-card-header>
                  <mat-icon mat-card-avatar>trending_up</mat-icon>
                  <mat-card-title>Tendencia</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  <div class="stat-value" [class.positive]="(analytics()?.improvement_trend || 0) > 0" [class.negative]="(analytics()?.improvement_trend || 0) < 0">
                    {{ (analytics()?.improvement_trend || 0) > 0 ? '+' : '' }}{{ ((analytics()?.improvement_trend || 0) * 100) | number:'1.0-0' }}%
                  </div>
                  <div class="stat-label">Últimos 30 días</div>
                </mat-card-content>
              </mat-card>
            </div>
          </mat-tab>

          <mat-tab label="Conceptos">
            <div class="concepts-section">
              <mat-card>
                <mat-card-header>
                  <mat-card-title>Conceptos Débiles</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  @if (analytics()?.weakest_concepts?.length) {
                    <mat-list>
                      @for (concept of analytics()?.weakest_concepts; track concept) {
                        <mat-list-item>
                          <mat-icon matListItemIcon>warning</mat-icon>
                          <span matListItemTitle>{{ concept }}</span>
                        </mat-list-item>
                      }
                    </mat-list>
                  } @else {
                    <p>No hay conceptos débiles identificados</p>
                  }
                </mat-card-content>
              </mat-card>

              <mat-card>
                <mat-card-header>
                  <mat-card-title>Conceptos Fuertes</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  @if (analytics()?.strongest_concepts?.length) {
                    <mat-list>
                      @for (concept of analytics()?.strongest_concepts; track concept) {
                        <mat-list-item>
                          <mat-icon matListItemIcon>check_circle</mat-icon>
                          <span matListItemTitle>{{ concept }}</span>
                        </mat-list-item>
                      }
                    </mat-list>
                  } @else {
                    <p>Aún no hay conceptos dominados</p>
                  }
                </mat-card-content>
              </mat-card>
            </div>
          </mat-tab>

          <mat-tab label="Planes de Recuperación">
            <div class="recovery-section">
              @if (recoveryPlans().length) {
                @for (plan of recoveryPlans(); track plan.id) {
                  <mat-card class="recovery-card">
                    <mat-card-header>
                      <mat-card-title>Plan de Recuperación</mat-card-title>
                      <mat-card-subtitle>
                        <mat-chip [color]="plan.priority >= 4 ? 'warn' : 'accent'">
                          Prioridad: {{ plan.priority }}/5
                        </mat-chip>
                        <mat-chip>{{ plan.status | titlecase }}</mat-chip>
                      </mat-card-subtitle>
                    </mat-card-header>
                    <mat-card-content>
                      <p><strong>Conceptos a repasar:</strong></p>
                      <div class="concept-chips">
                        @for (concept of plan.concepts_to_review; track concept) {
                          <mat-chip>{{ concept }}</mat-chip>
                        }
                      </div>
                      @if (plan.target_date) {
                        <p><strong>Fecha objetivo:</strong> {{ plan.target_date | date }}</p>
                      }
                    </mat-card-content>
                    <mat-card-actions>
                      @if (plan.status === 'active') {
                        <button mat-raised-button color="primary" (click)="completePlan(plan.id)">
                          Marcar como Completado
                        </button>
                        <button mat-button (click)="cancelPlan(plan.id)">Cancelar</button>
                      }
                    </mat-card-actions>
                  </mat-card>
                }
              } @else {
                <mat-card>
                  <mat-card-content>
                    <p>No tienes planes de recuperación activos</p>
                  </mat-card-content>
                </mat-card>
              }
            </div>
          </mat-tab>

          <mat-tab label="Alertas">
            <div class="alerts-section">
              @if (alerts().length) {
                @for (alert of alerts(); track alert.id) {
                  <mat-card class="alert-card" [class.critical]="alert.severity === 'critical'" [class.warning]="alert.severity === 'warning'">
                    <mat-card-header>
                      <mat-icon mat-card-avatar [color]="alert.severity === 'critical' ? 'warn' : 'accent'">
                        {{ alert.severity === 'critical' ? 'error' : 'warning' }}
                      </mat-icon>
                      <mat-card-title>{{ alert.title }}</mat-card-title>
                      <mat-card-subtitle>{{ alert.severity | titlecase }} · {{ alert.created_at | date }}</mat-card-subtitle>
                    </mat-card-header>
                    <mat-card-content>
                      <p>{{ alert.message }}</p>
                    </mat-card-content>
                    <mat-card-actions>
                      @if (!alert.acknowledged) {
                        <button mat-button (click)="acknowledgeAlert(alert.id)">Marcar como Leído</button>
                      }
                    </mat-card-actions>
                  </mat-card>
                }
              } @else {
                <mat-card>
                  <mat-card-content>
                    <p>No hay alertas activas</p>
                  </mat-card-content>
                </mat-card>
              }
            </div>
          </mat-tab>
          <mat-tab label="Matriz de Competencias">
            <div class="competency-section">
              <mat-card>
                <mat-card-header>
                  <mat-card-title>Matriz de Competencias por Curso</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  @if (competencyMatrix().length) {
                    <table class="analytics-table">
                      <thead>
                        <tr>
                          <th>Competencia</th>
                          <th>Nivel</th>
                          <th>Puntuación</th>
                        </tr>
                      </thead>
                      <tbody>
                        @for (item of competencyMatrix(); track item.competency) {
                          <tr>
                            <td>{{ item.competency }}</td>
                            <td>{{ item.level | titlecase }}</td>
                            <td>{{ (item.score * 100) | number:'1.0-0' }}%</td>
                          </tr>
                        }
                      </tbody>
                    </table>
                  } @else {
                    <p>No hay datos de competencias disponibles</p>
                  }
                </mat-card-content>
              </mat-card>
            </div>
          </mat-tab>

          <mat-tab label="Patrones de Errores">
            <div class="errors-section">
              <mat-card>
                <mat-card-header>
                  <mat-card-title>Patrones de Errores Comunes</mat-card-title>
                </mat-card-header>
                <mat-card-content>
                  @if (errorPatterns().length) {
                    <table class="analytics-table">
                      <thead>
                        <tr>
                          <th>Concepto</th>
                          <th>Frecuencia</th>
                          <th>Descripción</th>
                        </tr>
                      </thead>
                      <tbody>
                        @for (pattern of errorPatterns(); track pattern.concept) {
                          <tr>
                            <td>{{ pattern.concept }}</td>
                            <td>{{ pattern.frequency }}</td>
                            <td>{{ pattern.description }}</td>
                          </tr>
                        }
                      </tbody>
                    </table>
                  } @else {
                    <p>No hay patrones de errores registrados</p>
                  }
                </mat-card-content>
              </mat-card>
            </div>
          </mat-tab>
        </mat-tab-group>
      }
    </div>
  `,
  styles: [`
    .analytics-container {
      max-width: 1200px;
      margin: 0 auto;
      padding: 24px;
    }
    .analytics-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    .analytics-table {
      width: 100%;
      border-collapse: collapse;
    }
    .analytics-table th,
    .analytics-table td {
      padding: 12px;
      border-bottom: 1px solid #e0e0e0;
      text-align: left;
    }
    .analytics-table th {
      font-weight: 600;
      color: #666;
    }
    .analytics-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 24px;
      margin-top: 24px;
    }
    .stat-card {
      text-align: center;
    }
    .stat-value {
      font-size: 2.5em;
      font-weight: bold;
      margin: 16px 0 8px;
    }
    .stat-label {
      color: #666;
    }
    .stat-detail {
      margin-top: 8px;
      color: #4caf50;
    }
    .competency-badge {
      text-transform: capitalize;
    }
    .competency-badge[data-level="beginner"] { color: #9e9e9e; }
    .competency-badge[data-level="developing"] { color: #ff9800; }
    .competency-badge[data-level="proficient"] { color: #2196f3; }
    .competency-badge[data-level="advanced"] { color: #4caf50; }
    .competency-badge[data-level="exceptional"] { color: #9c27b0; }
    .positive { color: #4caf50; }
    .negative { color: #f44336; }
    .concepts-section, .recovery-section, .alerts-section, .competency-section, .errors-section {
      display: flex;
      flex-direction: column;
      gap: 16px;
      margin-top: 24px;
    }
    .concept-chips {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin: 8px 0;
    }
    .recovery-card {
      border-left: 4px solid #ff9800;
    }
    .alert-card {
      margin-bottom: 12px;
    }
    .alert-card.critical {
      border-left: 4px solid #f44336;
    }
    .alert-card.warning {
      border-left: 4px solid #ff9800;
    }
  `]
})
export class AnalyticsComponent implements OnInit {
  loading = signal(true);
  analytics = signal<StudentAnalytics | null>(null);
  recoveryPlans = signal<RecoveryPlan[]>([]);
  alerts = signal<AcademicAlert[]>([]);
  competencyMatrix = signal<any[]>([]);
  errorPatterns = signal<any[]>([]);
  selectedCourseId = signal<string>('matematica-1');

  constructor(
    private assessmentService: AssessmentService,
    private learningService: LearningService
  ) {}

  ngOnInit() {
    this.loadData();
  }

  loadData() {
    this.loading.set(true);
    this.assessmentService.getStudentAnalytics('current').subscribe({
      next: (data) => {
        this.analytics.set(data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
      }
    });

    this.assessmentService.getRecoveryPlans().subscribe({
      next: (plans) => this.recoveryPlans.set(plans || []),
      error: () => {}
    });

    this.assessmentService.getAlerts().subscribe({
      next: (alerts) => this.alerts.set(alerts || []),
      error: () => {}
    });

    this.loadCompetencyMatrix();
    this.loadErrorPatterns();
  }

  loadCompetencyMatrix() {
    this.assessmentService.getCompetencyMatrix(this.selectedCourseId()).subscribe({
      next: (matrix) => this.competencyMatrix.set(matrix),
      error: () => this.competencyMatrix.set([])
    });
  }

  loadErrorPatterns() {
    this.assessmentService.getErrorPatterns(this.selectedCourseId()).subscribe({
      next: (patterns) => this.errorPatterns.set(patterns),
      error: () => this.errorPatterns.set([])
    });
  }

  exportCSV() {
    if (this.selectedCourseId()) {
      this.assessmentService.exportCourseCSV(this.selectedCourseId());
    }
  }

  completePlan(planId: string) {
    this.assessmentService.completeRecoveryPlan(planId).subscribe({
      next: () => {
        this.recoveryPlans.update(plans =>
          plans.map(p => p.id === planId ? { ...p, status: 'completed' as const } : p)
        );
      }
    });
  }

  cancelPlan(planId: string) {
    this.assessmentService.cancelRecoveryPlan(planId).subscribe({
      next: () => {
        this.recoveryPlans.update(plans =>
          plans.map(p => p.id === planId ? { ...p, status: 'cancelled' as const } : p)
        );
      }
    });
  }

  acknowledgeAlert(alertId: string) {
    this.assessmentService.acknowledgeAlert(alertId).subscribe({
      next: () => {
        this.alerts.update(alerts =>
          alerts.map(a => a.id === alertId ? { ...a, acknowledged: true } : a)
        );
      }
    });
  }
}
