import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LearningService, TeacherCourseProgress, TopicMastery, CommonError, StudentProgress } from '../../core/services/learning.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';

@Component({
  selector: 'app-teacher-dashboard',
  standalone: true,
  imports: [CommonModule, MatButtonModule, MatIconModule, MatProgressBarModule],
  template: `
    <div class="teacher-container">
      <div class="teacher-header">
        <h1><mat-icon>analytics</mat-icon> Panel del Profesor</h1>
      </div>

      @if (loading()) {
        <div class="loading-box"><mat-icon>sync</mat-icon> Cargando...</div>
      }

      @if (courseProgress()) {
        <div class="overview-card">
          <div class="overview-stat">
            <div class="overview-value">{{ courseProgress()!.total_students }}</div>
            <div class="overview-label">Estudiantes</div>
          </div>
          <div class="overview-stat">
            <div class="overview-value">{{ (courseProgress()!.average_mastery * 100).toFixed(0) }}%</div>
            <div class="overview-label">Dominio Promedio</div>
            <mat-progress-bar mode="determinate" [value]="courseProgress()!.average_mastery * 100" class="overview-bar"></mat-progress-bar>
          </div>
        </div>
      }

      @if (topicMastery().length > 0) {
        <div class="section">
          <h2><mat-icon>school</mat-icon> Rendimiento del Curso</h2>
          <div class="topic-table">
            <div class="topic-header">
              <span>Tema</span><span>Promedio</span><span>Estudiantes</span><span>En dificultad</span>
            </div>
            @for (t of topicMastery(); track t.concept_id) {
              <div class="topic-row">
                <span class="topic-name">{{ t.concept_name }}</span>
                <span class="topic-mastery">
                  <mat-progress-bar mode="determinate" [value]="t.average_mastery * 100" [class]="getMasteryClass(t.average_mastery)"></mat-progress-bar>
                  <span class="mastery-pct">{{ (t.average_mastery * 100).toFixed(0) }}%</span>
                </span>
                <span>{{ t.student_count }}</span>
                <span class="struggling" [class.has-struggling]="t.struggling_count > 0">{{ t.struggling_count }}</span>
              </div>
            }
          </div>
        </div>
      }

      @if (commonErrors().length > 0) {
        <div class="section">
          <h2><mat-icon>bug_report</mat-icon> Errores Comunes</h2>
          <div class="errors-table">
            @for (e of commonErrors(); track $index) {
              <div class="error-row">
                <span class="error-type-badge">{{ e.error_type }}</span>
                <span class="error-subtype">{{ e.error_subtype || '-' }}</span>
                <span class="error-count">{{ e.count }} ocurrencias</span>
                <span class="error-affected">{{ e.affected_students }} estudiantes</span>
              </div>
            }
          </div>
        </div>
      }

      @if (students().length > 0) {
        <div class="section">
          <h2><mat-icon>people</mat-icon> Progreso por Estudiante</h2>
          <div class="student-table">
            <div class="student-header">
              <span>Nombre</span><span>Email</span><span>Nivel</span><span>Intentos</span><span>Última vez</span>
            </div>
            @for (s of students(); track s.student_id) {
              <div class="student-row">
                <span class="student-name">{{ s.student_name }}</span>
                <span class="student-email">{{ s.email }}</span>
                <span class="student-level">
                  <mat-progress-bar mode="determinate" [value]="s.overall_level * 100" [class]="getMasteryClass(s.overall_level)"></mat-progress-bar>
                  {{ (s.overall_level * 100).toFixed(0) }}%
                </span>
                <span>{{ s.total_attempts }}</span>
                <span class="student-date">{{ formatDate(s.last_active) }}</span>
              </div>
            }
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .teacher-container { padding: 1.5rem; max-width: 1000px; margin: 0 auto; }
    .teacher-header h1 { color: var(--accent); font-family: 'Newsreader', serif; margin: 0 0 1.5rem 0; display: flex; align-items: center; gap: 0.5rem; }
    .teacher-header h1 mat-icon { font-size: 28px; width: 28px; height: 28px; }

    .loading-box { display: flex; align-items: center; gap: 0.5rem; padding: 1rem; border-radius: 8px; background: var(--surface); color: var(--text-secondary); }

    .overview-card { display: flex; gap: 2rem; background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; margin-bottom: 1.5rem; }
    .overview-value { font-size: 2rem; font-weight: 700; color: var(--accent); }
    .overview-label { font-size: 0.78rem; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.05em; }
    .overview-bar { margin-top: 0.5rem; max-width: 200px; border-radius: 4px; }

    .section { margin-bottom: 1.5rem; }
    .section h2 { color: var(--accent); font-size: 1rem; margin: 0 0 0.75rem 0; display: flex; align-items: center; gap: 0.4rem; }
    .section h2 mat-icon { font-size: 20px; width: 20px; height: 20px; }

    .topic-table, .student-table, .errors-table { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; }
    .topic-header, .student-header { display: grid; grid-template-columns: 2fr 1fr 1fr 1fr; padding: 0.6rem 1rem; background: var(--border); font-size: 0.72rem; font-weight: 600; text-transform: uppercase; color: var(--text-secondary); letter-spacing: 0.05em; }
    .topic-row, .student-row { display: grid; grid-template-columns: 2fr 1fr 1fr 1fr; padding: 0.6rem 1rem; border-top: 1px solid var(--border); font-size: 0.85rem; align-items: center; }
    .topic-row:hover, .student-row:hover { background: rgba(200, 170, 118, 0.05); }
    .topic-name, .student-name { font-weight: 600; }
    .student-email { color: var(--text-secondary); font-size: 0.78rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .topic-mastery { display: flex; align-items: center; gap: 0.5rem; }
    .topic-mastery mat-progress-bar { flex: 1; border-radius: 4px; }
    .mastery-pct { font-size: 0.78rem; min-width: 35px; }
    .struggling { color: var(--text-secondary); }
    .struggling.has-struggling { color: #ef5350; font-weight: 700; }
    .student-level { display: flex; align-items: center; gap: 0.5rem; }
    .student-level mat-progress-bar { flex: 1; border-radius: 4px; max-width: 80px; }
    .student-date { font-size: 0.78rem; color: var(--text-secondary); }

    .error-row { display: flex; align-items: center; gap: 0.75rem; padding: 0.6rem 1rem; border-top: 1px solid var(--border); font-size: 0.85rem; }
    .error-type-badge { background: rgba(244, 67, 54, 0.1); color: #ef5350; padding: 0.15rem 0.5rem; border-radius: 4px; font-weight: 600; font-size: 0.78rem; }
    .error-subtype { color: var(--text-secondary); flex: 1; }
    .error-count { font-size: 0.78rem; }
    .error-affected { font-size: 0.78rem; color: var(--text-secondary); }

    :host ::ng-deep mat-progress-bar.bar-green .mat-mdc-progress-bar-fill::after { background: #4caf50; }
    :host ::ng-deep mat-progress-bar.bar-yellow .mat-mdc-progress-bar-fill::after { background: #ffc107; }
    :host ::ng-deep mat-progress-bar.bar-red .mat-mdc-progress-bar-fill::after { background: #ef5350; }

    @media (max-width: 768px) {
      .overview-card { flex-direction: column; gap: 1rem; }
      .topic-header, .topic-row, .student-header, .student-row { grid-template-columns: 1.5fr 1fr 0.8fr 0.8fr; font-size: 0.78rem; }
      .error-row { flex-wrap: wrap; }
    }
  `]
})
export class TeacherDashboardComponent implements OnInit {
  courseProgress = signal<TeacherCourseProgress | null>(null);
  topicMastery = signal<TopicMastery[]>([]);
  commonErrors = signal<CommonError[]>([]);
  students = signal<StudentProgress[]>([]);
  loading = signal(true);

  constructor(private learning: LearningService) {}

  ngOnInit() {
    this.learning.getTeacherCourseProgress().subscribe({
      next: (p) => { this.courseProgress.set(p); this.loading.set(false); },
      error: () => this.loading.set(false)
    });
    this.learning.getTeacherTopicMastery().subscribe({
      next: (t) => this.topicMastery.set(t),
      error: () => {}
    });
    this.learning.getTeacherCommonErrors().subscribe({
      next: (e) => this.commonErrors.set(e),
      error: () => {}
    });
    this.learning.getTeacherStudentProgress().subscribe({
      next: (s) => this.students.set(s),
      error: () => {}
    });
  }

  getMasteryClass(mastery: number): string {
    if (mastery >= 0.7) return 'bar-green';
    if (mastery >= 0.3) return 'bar-yellow';
    return 'bar-red';
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return '-';
    try { return new Date(dateStr).toLocaleDateString('es-AR'); } catch { return dateStr; }
  }
}
