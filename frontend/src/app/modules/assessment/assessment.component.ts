import { Component, OnInit, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatRadioModule } from '@angular/material/radio';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatTabsModule } from '@angular/material/tabs';
import { MatChipsModule } from '@angular/material/chips';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { AuthService } from '../../core/services/auth.service';
import {
  AssessmentService, Assessment, AssessmentQuestion,
  StudentAssessment, StudentAnswer
} from '../../core/services/assessment.service';

type PageMode = 'list' | 'taking' | 'results' | 'teacher-list' | 'create-form' | 'edit-form' | 'teacher-results';

@Component({
  selector: 'app-assessment',
  standalone: true,
  imports: [
    CommonModule, FormsModule, MatCardModule, MatButtonModule, MatIconModule,
    MatProgressBarModule, MatRadioModule, MatInputModule, MatFormFieldModule,
    MatTabsModule, MatChipsModule, MatSnackBarModule, MatSelectModule,
    MatSlideToggleModule
  ],
  template: `
    @if (loading()) {
      <div class="loading-container">
        <mat-progress-bar mode="indeterminate"></mat-progress-bar>
        <p>Cargando...</p>
      </div>
    } @else if (error()) {
      <div class="error-container">
        <mat-icon color="warn">error</mat-icon>
        <p>{{ error() }}</p>
        <button mat-raised-button color="primary" (click)="goBack()">Volver</button>
      </div>
    } @else if (mode() === 'teacher-list') {
      <div class="page">
        <div class="page-header">
          <div>
            <h2>Gestión de Evaluaciones</h2>
            <p class="page-subtitle">Creá, editá y publicá evaluaciones para tus cursos</p>
          </div>
          <button mat-raised-button color="primary" (click)="showCreateForm()">
            <mat-icon>add</mat-icon> Nueva Evaluación
          </button>
        </div>

        <div class="tab-bar">
          <button class="tab-btn" [class.active]="teacherTab() === 'draft'" (click)="teacherTab.set('draft')">Borradores</button>
          <button class="tab-btn" [class.active]="teacherTab() === 'published'" (click)="teacherTab.set('published')">Publicadas</button>
          <button class="tab-btn" [class.active]="teacherTab() === 'all'" (click)="teacherTab.set('all')">Todas</button>
        </div>

        @if (filteredAssessments().length === 0) {
          <div class="empty-state">
            <mat-icon>quiz</mat-icon>
            <p>No hay evaluaciones {{ teacherTab() === 'draft' ? 'en borrador' : teacherTab() === 'published' ? 'publicadas' : '' }}</p>
          </div>
        }
        @for (a of filteredAssessments(); track a.id) {
          <mat-card class="teacher-card">
            <mat-card-header>
              <mat-card-title>{{ a.title }}</mat-card-title>
              <mat-card-subtitle>
                {{ a.assessment_type | titlecase }} · {{ a.total_points }} pts · {{ a.mode }}
                <span class="status-badge" [class.draft]="a.status === 'draft'" [class.published]="a.status === 'published'">
                  {{ a.status === 'draft' ? 'Borrador' : 'Publicada' }}
                </span>
              </mat-card-subtitle>
            </mat-card-header>
            <mat-card-content>
              <p>{{ a.description }}</p>
              <div class="card-meta">
                <span>⏱ {{ a.time_limit_minutes > 0 ? a.time_limit_minutes + ' min' : 'Sin límite' }}</span>
                <span>🎯 Aprobar: {{ (a.passing_score * 100) | number:'1.0-0' }}%</span>
                <span>🔄 {{ a.max_attempts }} intento(s)</span>
              </div>
            </mat-card-content>
            <mat-card-actions>
              @if (a.status === 'draft') {
                <button mat-button color="primary" (click)="editAssessment(a)"><mat-icon>edit</mat-icon> Editar</button>
                <button mat-button color="accent" (click)="publishAssessment(a)"><mat-icon>publish</mat-icon> Publicar</button>
              }
              <button mat-button (click)="viewResults(a)"><mat-icon>people</mat-icon> Resultados</button>
              <button mat-button color="warn" (click)="deleteAssessment(a)"><mat-icon>delete</mat-icon> Eliminar</button>
            </mat-card-actions>
          </mat-card>
        }
      </div>

    } @else if (mode() === 'create-form' || mode() === 'edit-form') {
      <div class="page form-page">
        <div class="page-header">
          <div>
            <h2>{{ mode() === 'create-form' ? 'Nueva Evaluación' : 'Editar Evaluación' }}</h2>
            <p class="page-subtitle">Completá los datos de la evaluación</p>
          </div>
        </div>

        <div class="form-grid">
          <mat-form-field appearance="outline">
            <mat-label>Título</mat-label>
            <input matInput [(ngModel)]="formData.title" placeholder="Ej: Parcial 1 - Análisis Matemático">
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Tipo</mat-label>
            <mat-select [(ngModel)]="formData.assessment_type">
              <mat-option value="diagnostic">Diagnóstico</mat-option>
              <mat-option value="formative">Formativo</mat-option>
              <mat-option value="summative">Sumativo</mat-option>
              <mat-option value="recovery">Recuperación</mat-option>
              <mat-option value="practice">Práctica</mat-option>
            </mat-select>
          </mat-form-field>

          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Descripción</mat-label>
            <textarea matInput [(ngModel)]="formData.description" rows="3" placeholder="Instrucciones para los estudiantes..."></textarea>
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Modalidad</mat-label>
            <mat-select [(ngModel)]="formData.mode">
              <mat-option value="fixed">Fijo (elegís las preguntas)</mat-option>
              <mat-option value="generated">Generado por IA</mat-option>
              <mat-option value="adaptive">Adaptativo (según desempeño)</mat-option>
            </mat-select>
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Puntaje total</mat-label>
            <input matInput type="number" [(ngModel)]="formData.total_points" min="1">
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Porcentaje para aprobar</mat-label>
            <input matInput type="number" [(ngModel)]="formData.passing_score_display" min="1" max="100">
            <span matTextSuffix>%</span>
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Límite de tiempo (minutos)</mat-label>
            <input matInput type="number" [(ngModel)]="formData.time_limit_minutes" min="0">
            <span matTextSuffix>0 = sin límite</span>
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Intentos máximos</mat-label>
            <input matInput type="number" [(ngModel)]="formData.max_attempts" min="1" value="1">
          </mat-form-field>

          @if (formData.mode === 'fixed') {
            <div class="full-width">
              <div class="section-label">Seleccionar preguntas del banco</div>
              @if (availableQuestions().length === 0) {
                <p class="text-secondary">No hay preguntas disponibles. Creá preguntas primero o usá el modo "Generado por IA".</p>
              }
              <div class="question-bank">
                @for (q of availableQuestions(); track q.id) {
                  <label class="question-option" [class.selected]="formData.question_ids.includes(q.id)">
                    <input type="checkbox" [checked]="formData.question_ids.includes(q.id)" (change)="toggleQuestion(q.id)">
                    <div class="question-option-content">
                      <strong>{{ q.statement || q.latex || 'Pregunta' }}</strong>
                      @if (q.difficulty) {
                        <span class="diff-badge">Dificultad: {{ q.difficulty }}</span>
                      }
                    </div>
                  </label>
                }
              </div>
            </div>
          }
        </div>

        <div class="form-actions">
          <button mat-button (click)="goBack()">Cancelar</button>
          <button mat-raised-button color="primary" (click)="saveAssessment()" [disabled]="!formData.title">
            <mat-icon>save</mat-icon>
            {{ mode() === 'create-form' ? 'Crear Evaluación' : 'Guardar Cambios' }}
          </button>
        </div>
      </div>

    } @else if (mode() === 'teacher-results') {
      <div class="page">
        <div class="page-header">
          <div>
            <h2>Resultados: {{ currentAssessment()?.title }}</h2>
            <p class="page-subtitle">{{ studentResults().length }} estudiante(s) presentaron</p>
          </div>
          <div class="header-actions">
            <button mat-button (click)="exportResultsCSV()"><mat-icon>download</mat-icon> Exportar CSV</button>
            <button mat-button (click)="goBack()"><mat-icon>arrow_back</mat-icon> Volver</button>
          </div>
        </div>

        @if (studentResults().length === 0) {
          <div class="empty-state">
            <mat-icon>people_outline</mat-icon>
            <p>Ningún estudiante ha rendido esta evaluación todavía</p>
          </div>
        }

        <div class="results-summary">
          <div class="stat-card">
            <div class="stat-value">{{ avgScore() }}%</div>
            <div class="stat-label">Promedio</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ passRate() }}%</div>
            <div class="stat-label">Aprobación</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ studentResults().length }}</div>
            <div class="stat-label">Entregas</div>
          </div>
        </div>

        <table class="results-table">
          <thead>
            <tr>
              <th>Estudiante</th>
              <th>Puntaje</th>
              <th>Porcentaje</th>
              <th>Estado</th>
              <th>Tiempo</th>
              <th>Acción</th>
            </tr>
          </thead>
          <tbody>
            @for (r of studentResults(); track r.id) {
              <tr>
                <td>{{ r.student_name || r.student_id }}</td>
                <td>{{ r.total_score }}/{{ r.max_score }}</td>
                <td>
                  <span class="score-pill" [class.pass]="r.passed" [class.fail]="!r.passed">
                    {{ (r.percentage * 100) | number:'1.0-0' }}%
                  </span>
                </td>
                <td>{{ r.passed ? '✅ Aprobado' : '❌ No aprobado' }}</td>
                <td>{{ formatTime(r.time_spent_seconds || 0) }}</td>
                <td>
                  <button mat-icon-button (click)="viewStudentDetail(r)" aria-label="Ver detalle">
                    <mat-icon>visibility</mat-icon>
                  </button>
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>

    } @else if (mode() === 'list') {
      <div class="page">
        <div class="page-header">
          <div>
            <h2>Evaluaciones Disponibles</h2>
            <p class="page-subtitle">Seleccioná una evaluación para rendir</p>
          </div>
          @if (auth.hasRole('ADMIN', 'TEACHER')) {
            <button mat-stroked-button (click)="switchToTeacher()">
              <mat-icon>manage_accounts</mat-icon> Gestión Docente
            </button>
          }
        </div>

        @if (studentAssessments().length === 0) {
          <div class="empty-state">
            <mat-icon>quiz</mat-icon>
            <p>No hay evaluaciones publicadas disponibles</p>
          </div>
        }
        @for (assessment of studentAssessments(); track assessment.id) {
          <mat-card class="assessment-card" (click)="selectAssessment(assessment)">
            <mat-card-header>
              <mat-card-title>{{ assessment.title }}</mat-card-title>
              <mat-card-subtitle>
                {{ assessment.assessment_type | titlecase }} · {{ assessment.total_points }} puntos
              </mat-card-subtitle>
            </mat-card-header>
            <mat-card-content>
              <p>{{ assessment.description }}</p>
              <div class="assessment-meta">
                @if (assessment.time_limit_minutes > 0) {
                  <mat-chip>{{ assessment.time_limit_minutes }} min</mat-chip>
                }
                <mat-chip>{{ assessment.max_attempts }} intento(s)</mat-chip>
                <mat-chip>Aprobar: {{ (assessment.passing_score * 100) | number:'1.0-0' }}%</mat-chip>
              </div>
            </mat-card-content>
          </mat-card>
        }
      </div>

    } @else if (mode() === 'taking') {
      <div class="page taking-page">
        <div class="assessment-header">
          <h2>{{ currentAssessment()?.title }}</h2>
          @if (timeRemaining() !== null) {
            <div class="timer" [class.warning]="timeRemaining()! < 300">
              <mat-icon>timer</mat-icon>
              {{ formatTime(timeRemaining()!) }}
            </div>
          }
          <div class="progress-label">
            Pregunta {{ currentIndex() + 1 }} de {{ questions().length }}
          </div>
        </div>

        <div class="autosave-status">
          @if (isSaving()) {
            <span class="saving"><mat-icon class="spin">sync</mat-icon> Guardando...</span>
          } @else if (lastSaved()) {
            <span class="saved"><mat-icon>check_circle</mat-icon> Guardado {{ lastSaved() }}</span>
          }
        </div>

        <mat-progress-bar [value]="((currentIndex() + 1) / questions().length) * 100"></mat-progress-bar>

        @if (currentQuestion()) {
          <mat-card class="question-card">
            <mat-card-header>
              <mat-card-subtitle>
                Pregunta {{ currentIndex() + 1 }} · {{ currentQuestion()!.points }} puntos
              </mat-card-subtitle>
            </mat-card-header>
            <mat-card-content>
              @if (currentQuestion()!.latex) {
                <div class="question-latex" [innerHTML]="currentQuestion()!.latex"></div>
              } @else {
                <p class="question-statement">{{ currentQuestion()!.statement }}</p>
              }

              <mat-form-field appearance="outline" class="answer-field">
                <mat-label>Tu respuesta</mat-label>
                <input matInput [(ngModel)]="currentAnswer" placeholder="Escribí tu respuesta...">
              </mat-form-field>
            </mat-card-content>
            <mat-card-actions>
              <button mat-button (click)="previousQuestion()" [disabled]="currentIndex() === 0">
                <mat-icon>chevron_left</mat-icon> Anterior
              </button>
              <span class="spacer"></span>
              @if (currentIndex() < questions().length - 1) {
                <button mat-raised-button color="primary" (click)="nextQuestion()">
                  Siguiente <mat-icon>chevron_right</mat-icon>
                </button>
              } @else {
                <button mat-raised-button color="accent" (click)="submitAssessment()">
                  <mat-icon>send</mat-icon> Enviar Evaluación
                </button>
              }
            </mat-card-actions>
          </mat-card>
        }

        <div class="question-nav">
          @for (q of questions(); track q.id; let i = $index) {
            <button mat-mini-fab
              [color]="i === currentIndex() ? 'primary' : (answers()[q.id] ? 'accent' : '')"
              (click)="goToQuestion(i)">
              {{ i + 1 }}
            </button>
          }
        </div>
      </div>

    } @else if (mode() === 'results') {
      <div class="page results-page">
        <mat-card class="results-header-card">
          <mat-card-header>
            <mat-card-title>Resultados</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <div class="score-display">
              <div class="score-circle" [class.passed]="result()!.passed">
                {{ (result()!.percentage * 100) | number:'1.0-0' }}%
              </div>
              <div class="score-details">
                <p class="score-main">{{ result()!.total_score }} / {{ result()!.max_score }} puntos</p>
                <p [class]="result()!.passed ? 'passed-text' : 'failed-text'">
                  {{ result()!.passed ? 'Aprobado' : 'No aprobado' }}
                </p>
              </div>
            </div>
          </mat-card-content>
        </mat-card>

        <div class="answers-breakdown">
          <h3>Detalle de Respuestas</h3>
          @for (answer of resultAnswers(); track answer.id; let i = $index) {
            <mat-card class="answer-card" [class.correct]="answer.is_correct" [class.incorrect]="!answer.is_correct">
              <mat-card-header>
                <mat-card-subtitle>
                  Pregunta {{ i + 1 }} · {{ answer.points_earned }}/{{ answer.points_possible }} puntos
                </mat-card-subtitle>
              </mat-card-header>
              <mat-card-content>
                <p><strong>Tu respuesta:</strong> {{ answer.answer }}</p>
                <p><strong>Resultado:</strong>
                  <span [class]="answer.is_correct ? 'correct-text' : 'incorrect-text'">
                    {{ answer.is_correct ? 'Correcto' : 'Incorrecto' }}
                  </span>
                </p>
                @if (answer.feedback) {
                  <p class="feedback-text">{{ answer.feedback }}</p>
                }
              </mat-card-content>
            </mat-card>
          }
        </div>

        <button mat-raised-button color="primary" (click)="goBack()">Volver a Evaluaciones</button>
      </div>
    }
  `,
  styles: [`
    .loading-container, .error-container {
      display: flex; flex-direction: column; align-items: center;
      justify-content: center; min-height: 400px; gap: var(--space-md);
    }
    .page { max-width: 860px; margin: 0 auto; padding: var(--space-xl); }
    .page-header {
      display: flex; justify-content: space-between; align-items: flex-start;
      margin-bottom: var(--space-lg); gap: var(--space-md);
      flex-wrap: wrap;
    }
    .page-header h2 { margin: 0; font-family: var(--font-serif); font-size: 1.35rem; font-weight: 600; }
    .page-subtitle { margin: var(--space-xs) 0 0; color: var(--text-secondary); font-size: 0.85rem; }
    .header-actions { display: flex; gap: var(--space-sm); }

    .tab-bar { display: flex; gap: var(--space-xs); margin-bottom: var(--space-lg); }
    .tab-btn {
      padding: var(--space-sm) var(--space-md); border-radius: var(--radius-sm);
      border: 1px solid var(--border); background: transparent; color: var(--text-secondary);
      cursor: pointer; font-size: 0.85rem; font-family: var(--font-sans);
      transition: all 0.12s;
    }
    .tab-btn:hover { border-color: var(--accent); color: var(--text); }
    .tab-btn.active { background: var(--accent-muted); border-color: var(--accent); color: var(--accent-text); font-weight: 500; }

    .empty-state {
      text-align: center; padding: var(--space-2xl); color: var(--text-secondary);
    }
    .empty-state mat-icon { font-size: 3rem; width: 3rem; height: 3rem; margin-bottom: var(--space-md); opacity: 0.4; }

    .teacher-card, .assessment-card { margin-bottom: var(--space-md); cursor: pointer; transition: transform 0.15s; }
    .teacher-card:hover, .assessment-card:hover { transform: translateY(-2px); }
    .teacher-card mat-card-actions { padding: 0 var(--space-md) var(--space-sm); }
    .status-badge {
      font-size: 0.7rem; padding: 0.1rem 0.45rem; border-radius: 3px; margin-left: var(--space-sm);
      font-weight: 600; text-transform: uppercase;
    }
    .status-badge.draft { background: var(--accent-muted); color: var(--accent-text); }
    .status-badge.published { background: var(--success-muted); color: var(--success); }
    .card-meta { display: flex; gap: var(--space-md); margin-top: var(--space-sm); font-size: 0.8rem; color: var(--text-secondary); }
    .assessment-meta { display: flex; gap: var(--space-sm); margin-top: var(--space-sm); flex-wrap: wrap; }

    .form-grid {
      display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md);
      margin-bottom: var(--space-xl);
    }
    .form-grid .full-width { grid-column: 1 / -1; }
    .section-label {
      font-size: 0.8rem; font-weight: 600; color: var(--text-secondary);
      margin-bottom: var(--space-sm); text-transform: uppercase; letter-spacing: 0.05em;
    }
    .text-secondary { color: var(--text-secondary); font-size: 0.85rem; }
    .question-bank { display: flex; flex-direction: column; gap: var(--space-xs); max-height: 320px; overflow-y: auto; }
    .question-option {
      display: flex; align-items: flex-start; gap: var(--space-sm);
      padding: var(--space-sm) var(--space-md); border: 1px solid var(--border);
      border-radius: var(--radius-sm); cursor: pointer; transition: all 0.12s;
    }
    .question-option:hover { border-color: var(--accent); }
    .question-option.selected { background: var(--accent-muted); border-color: var(--accent); }
    .question-option input[type="checkbox"] { margin-top: 3px; accent-color: var(--accent); }
    .question-option-content { font-size: 0.85rem; }
    .diff-badge { font-size: 0.7rem; color: var(--text-tertiary); margin-left: var(--space-sm); }
    .form-actions { display: flex; justify-content: flex-end; gap: var(--space-sm); }

    .results-summary { display: flex; gap: var(--space-md); margin-bottom: var(--space-xl); }
    .stat-card {
      flex: 1; background: var(--surface); border: 1px solid var(--border);
      border-radius: var(--radius-md); padding: var(--space-lg); text-align: center;
    }
    .stat-value { font-size: 1.8rem; font-weight: 700; color: var(--accent-text); }
    .stat-label { font-size: 0.8rem; color: var(--text-secondary); margin-top: var(--space-xs); }

    .results-table { width: 100%; border-collapse: collapse; }
    .results-table th {
      text-align: left; padding: var(--space-sm) var(--space-md);
      border-bottom: 2px solid var(--border); font-size: 0.75rem;
      text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-tertiary);
    }
    .results-table td { padding: var(--space-sm) var(--space-md); border-bottom: 1px solid var(--border-light); font-size: 0.85rem; }
    .results-table tr:hover td { background: var(--accent-muted); }
    .score-pill {
      padding: 2px 10px; border-radius: 12px; font-weight: 600; font-size: 0.8rem;
    }
    .score-pill.pass { background: var(--success-muted); color: var(--success); }
    .score-pill.fail { background: var(--danger-muted); color: var(--danger); }

    .taking-page .assessment-header {
      display: flex; justify-content: space-between; align-items: center;
      margin-bottom: var(--space-md); flex-wrap: wrap; gap: var(--space-sm);
    }
    .taking-page .assessment-header h2 { margin: 0; font-size: 1.2rem; }
    .timer {
      display: flex; align-items: center; gap: var(--space-xs);
      font-size: 1.1rem; font-weight: 700; color: var(--text);
    }
    .timer.warning { color: var(--danger); }
    .progress-label { color: var(--text-secondary); font-size: 0.85rem; }
    .autosave-status { text-align: right; margin-bottom: var(--space-xs); font-size: 0.8rem; }
    .autosave-status .saving { display: inline-flex; align-items: center; gap: 4px; color: var(--text-secondary); }
    .autosave-status .saved { display: inline-flex; align-items: center; gap: 4px; color: var(--success); }
    .spin { animation: spin 1s linear infinite; }
    @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
    .question-card { margin: var(--space-md) 0; }
    .question-latex, .question-statement { font-size: 1.05rem; margin-bottom: var(--space-lg); line-height: 1.6; }
    .answer-field { width: 100%; }
    mat-card-actions { display: flex; align-items: center; }
    .spacer { flex: 1; }
    .question-nav { display: flex; flex-wrap: wrap; gap: var(--space-sm); margin-top: var(--space-lg); justify-content: center; }

    .results-page .results-header-card { text-align: center; margin-bottom: var(--space-xl); }
    .score-display { display: flex; align-items: center; justify-content: center; gap: var(--space-xl); padding: var(--space-xl); }
    .score-circle {
      font-size: 2.5rem; font-weight: 700; width: 130px; height: 130px;
      border-radius: 50%; display: flex; align-items: center; justify-content: center;
      border: 6px solid var(--danger); color: var(--text);
    }
    .score-circle.passed { border-color: var(--success); }
    .score-main { font-size: 1.2rem; font-weight: 600; margin: 0 0 var(--space-xs); }
    .passed-text { color: var(--success); font-weight: 600; }
    .failed-text { color: var(--danger); font-weight: 600; }
    .correct-text { color: var(--success); font-weight: 600; }
    .incorrect-text { color: var(--danger); font-weight: 600; }
    .answers-breakdown { margin-bottom: var(--space-xl); }
    .answers-breakdown h3 { font-family: var(--font-serif); margin-bottom: var(--space-md); }
    .answer-card { margin-bottom: var(--space-sm); border-left: 4px solid; }
    .answer-card.correct { border-left-color: var(--success); }
    .answer-card.incorrect { border-left-color: var(--danger); }
    .feedback-text { color: var(--text-secondary); font-style: italic; margin-top: var(--space-sm); }

    @media (max-width: 640px) {
      .page { padding: var(--space-md); }
      .form-grid { grid-template-columns: 1fr; }
      .results-summary { flex-direction: column; }
      .results-table { font-size: 0.8rem; }
      .results-table th, .results-table td { padding: var(--space-sm); }
      .score-display { flex-direction: column; }
      .score-circle { width: 100px; height: 100px; font-size: 2rem; }
    }
  `]
})
export class AssessmentComponent implements OnInit {
  loading = signal(true);
  error = signal<string | null>(null);
  mode = signal<PageMode>('list');

  studentAssessments = signal<Assessment[]>([]);
  teacherAssessments = signal<Assessment[]>([]);
  teacherTab = signal<'draft' | 'published' | 'all'>('draft');

  currentAssessment = signal<Assessment | null>(null);
  questions = signal<AssessmentQuestion[]>([]);
  currentIndex = signal(0);
  currentAnswer = '';
  answers = signal<Record<string, string>>({});
  studentAssessment = signal<StudentAssessment | null>(null);
  timeRemaining = signal<number | null>(null);
  timeSpentSeconds = signal(0);
  result = signal<StudentAssessment | null>(null);
  resultAnswers = signal<StudentAnswer[]>([]);
  isSaving = signal(false);
  lastSaved = signal<string | null>(null);

  formData: {
    title: string;
    description: string;
    assessment_type: 'diagnostic' | 'formative' | 'summative' | 'recovery' | 'practice';
    mode: 'fixed' | 'generated' | 'adaptive';
    total_points: number;
    passing_score_display: number;
    time_limit_minutes: number;
    max_attempts: number;
    question_ids: string[];
  } = {
    title: '',
    description: '',
    assessment_type: 'formative',
    mode: 'fixed',
    total_points: 100,
    passing_score_display: 60,
    time_limit_minutes: 0,
    max_attempts: 1,
    question_ids: []
  };
  editId: string | null = null;

  availableQuestions = signal<any[]>([]);
  studentResults = signal<any[]>([]);

  private timerInterval: any;
  private autosaveInterval: any;

  currentQuestion = computed(() => {
    const qs = this.questions();
    const idx = this.currentIndex();
    return qs.length > idx ? qs[idx] : null;
  });

  filteredAssessments = computed(() => {
    const all = this.teacherAssessments();
    const tab = this.teacherTab();
    if (tab === 'draft') return all.filter(a => a.status === 'draft');
    if (tab === 'published') return all.filter(a => a.status === 'published');
    return all;
  });

  avgScore = computed(() => {
    const results = this.studentResults();
    if (results.length === 0) return 0;
    const sum = results.reduce((acc, r) => acc + (r.percentage || 0), 0);
    return Math.round((sum / results.length) * 100);
  });

  passRate = computed(() => {
    const results = this.studentResults();
    if (results.length === 0) return 0;
    const passed = results.filter(r => r.passed).length;
    return Math.round((passed / results.length) * 100);
  });

  constructor(
    private assessmentService: AssessmentService,
    protected auth: AuthService,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit() {
    if (this.auth.hasRole('ADMIN', 'TEACHER')) {
      this.loadTeacherAssessments();
    } else {
      this.loadStudentAssessments();
    }
  }

  loadStudentAssessments() {
    this.loading.set(true);
    this.assessmentService.listAssessments(undefined, undefined, 'published').subscribe({
      next: (assessments) => {
        this.studentAssessments.set(assessments || []);
        this.loading.set(false);
      },
      error: () => {
        this.error.set('Error al cargar evaluaciones');
        this.loading.set(false);
      }
    });
  }

  loadTeacherAssessments() {
    this.loading.set(true);
    this.mode.set('teacher-list');
    this.assessmentService.listAssessments().subscribe({
      next: (assessments) => {
        this.teacherAssessments.set(assessments || []);
        this.loading.set(false);
      },
      error: () => {
        this.error.set('Error al cargar evaluaciones');
        this.loading.set(false);
      }
    });
  }

  switchToTeacher() {
    this.loadTeacherAssessments();
  }

  showCreateForm() {
    this.formData = {
      title: '',
      description: '',
      assessment_type: 'formative',
      mode: 'fixed',
      total_points: 100,
      passing_score_display: 60,
      time_limit_minutes: 0,
      max_attempts: 1,
      question_ids: []
    };
    this.editId = null;
    this.mode.set('create-form');
    this.loadQuestions();
  }

  editAssessment(a: Assessment) {
    this.formData = {
      title: a.title || '',
      description: a.description || '',
      assessment_type: a.assessment_type || 'formative',
      mode: a.mode || 'fixed',
      total_points: a.total_points || 100,
      passing_score_display: (a.passing_score || 0.6) * 100,
      time_limit_minutes: a.time_limit_minutes || 0,
      max_attempts: a.max_attempts || 1,
      question_ids: []
    };
    this.editId = a.id;
    this.currentAssessment.set(a);
    this.mode.set('edit-form');
    this.loadQuestions();
  }

  loadQuestions() {
    this.assessmentService.getQuestions({}).subscribe({
      next: (qs) => this.availableQuestions.set(qs || []),
      error: () => this.availableQuestions.set([])
    });
  }

  toggleQuestion(qId: string) {
    const ids = this.formData.question_ids;
    const idx = ids.indexOf(qId);
    if (idx >= 0) {
      ids.splice(idx, 1);
    } else {
      ids.push(qId);
    }
    this.formData.question_ids = [...ids];
  }

  saveAssessment() {
    if (!this.formData.title) return;
    this.loading.set(true);

    const payload = {
      title: this.formData.title,
      description: this.formData.description,
      course_id: 'matematica-1',
      assessment_type: this.formData.assessment_type,
      mode: this.formData.mode,
      total_points: this.formData.total_points,
      passing_score: this.formData.passing_score_display / 100,
      time_limit_minutes: this.formData.time_limit_minutes,
      max_attempts: this.formData.max_attempts,
      question_ids: this.formData.mode === 'fixed' ? this.formData.question_ids : []
    };

    if (this.mode() === 'create-form') {
      this.assessmentService.createAssessment(payload).subscribe({
        next: () => {
          this.snackBar.open('Evaluación creada', 'Cerrar', { duration: 2000 });
          this.loadTeacherAssessments();
        },
        error: (err) => {
          this.snackBar.open(err.error?.error || 'Error al crear', 'Cerrar', { duration: 3000 });
          this.loading.set(false);
        }
      });
    } else if (this.editId) {
      this.assessmentService.updateAssessment(this.editId, payload).subscribe({
        next: () => {
          this.snackBar.open('Evaluación actualizada', 'Cerrar', { duration: 2000 });
          this.loadTeacherAssessments();
        },
        error: (err) => {
          this.snackBar.open(err.error?.error || 'Error al actualizar', 'Cerrar', { duration: 3000 });
          this.loading.set(false);
        }
      });
    }
  }

  publishAssessment(a: Assessment) {
    this.assessmentService.publishAssessment(a.id).subscribe({
      next: () => {
        this.snackBar.open('Evaluación publicada', 'Cerrar', { duration: 2000 });
        this.loadTeacherAssessments();
      },
      error: () => this.snackBar.open('Error al publicar', 'Cerrar', { duration: 3000 })
    });
  }

  deleteAssessment(a: Assessment) {
    if (!confirm(`¿Eliminar "${a.title}"?`)) return;
    this.assessmentService.deleteAssessment(a.id).subscribe({
      next: () => {
        this.snackBar.open('Evaluación eliminada', 'Cerrar', { duration: 2000 });
        this.loadTeacherAssessments();
      },
      error: () => this.snackBar.open('Error al eliminar', 'Cerrar', { duration: 3000 })
    });
  }

  viewResults(a: Assessment) {
    this.currentAssessment.set(a);
    this.loading.set(true);
    this.mode.set('teacher-results');
    this.assessmentService.getAssessmentStudentResults(a.id).subscribe({
      next: (results) => {
        this.studentResults.set(results || []);
        this.loading.set(false);
      },
      error: () => {
        this.studentResults.set([]);
        this.loading.set(false);
      }
    });
  }

  viewStudentDetail(r: any) {
    this.snackBar.open(`Estudiante: ${r.student_name || r.student_id} - ${r.total_score}/${r.max_score} pts`, 'Cerrar', { duration: 4000 });
  }

  exportResultsCSV() {
    const a = this.currentAssessment();
    if (a) this.assessmentService.exportAssessmentCSV(a.id);
  }

  selectAssessment(assessment: Assessment) {
    this.currentAssessment.set(assessment);
    this.loading.set(true);
    this.assessmentService.startAssessment(assessment.id).subscribe({
      next: (resp) => {
        this.studentAssessment.set(resp.student_assessment);
        this.questions.set(resp.questions || []);
        this.currentIndex.set(0);
        this.mode.set('taking');
        this.loading.set(false);
        this.startAutosave();
        this.checkForResume(assessment.id);
        if (assessment.time_limit_minutes > 0) {
          this.startTimer(assessment.time_limit_minutes * 60);
        }
      },
      error: (err) => {
        this.snackBar.open(err.error?.error || 'Error al iniciar evaluación', 'Cerrar', { duration: 3000 });
        this.loading.set(false);
      }
    });
  }

  startTimer(seconds: number) {
    this.timeRemaining.set(seconds);
    this.timerInterval = setInterval(() => {
      const current = this.timeRemaining();
      if (current !== null && current > 0) {
        this.timeRemaining.set(current - 1);
      } else {
        clearInterval(this.timerInterval);
        this.submitAssessment();
      }
    }, 1000);
  }

  formatTime(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  nextQuestion() {
    this.saveCurrentAnswer();
    if (this.currentIndex() < this.questions().length - 1) {
      this.currentIndex.set(this.currentIndex() + 1);
      this.loadCurrentAnswer();
    }
  }

  previousQuestion() {
    this.saveCurrentAnswer();
    if (this.currentIndex() > 0) {
      this.currentIndex.set(this.currentIndex() - 1);
      this.loadCurrentAnswer();
    }
  }

  goToQuestion(index: number) {
    this.saveCurrentAnswer();
    this.currentIndex.set(index);
    this.loadCurrentAnswer();
  }

  saveCurrentAnswer() {
    const q = this.currentQuestion();
    if (q) {
      const current = this.answers();
      this.answers.set({ ...current, [q.id]: this.currentAnswer });
    }
  }

  loadCurrentAnswer() {
    const q = this.currentQuestion();
    if (q) {
      this.currentAnswer = this.answers()[q.id] || '';
    }
  }

  submitAssessment() {
    this.saveCurrentAnswer();
    this.stopAutosave();
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }

    const assessment = this.currentAssessment();
    if (!assessment) return;

    const answersArray = Object.entries(this.answers()).map(([question_id, answer]) => ({
      question_id, answer, procedure: []
    }));

    this.loading.set(true);
    this.assessmentService.submitAssessment(assessment.id, answersArray).subscribe({
      next: (resp) => {
        this.result.set({
          ...this.studentAssessment()!,
          total_score: resp.total_score,
          max_score: resp.max_score,
          percentage: resp.percentage,
          passed: resp.passed,
          status: 'graded'
        } as StudentAssessment);

        this.assessmentService.getAssessmentResult(assessment.id).subscribe({
          next: (resultResp) => {
            this.resultAnswers.set(resultResp.answers || []);
            this.mode.set('results');
            this.loading.set(false);
          },
          error: () => {
            this.mode.set('results');
            this.loading.set(false);
          }
        });
      },
      error: () => {
        this.snackBar.open('Error al enviar evaluación', 'Cerrar', { duration: 3000 });
        this.loading.set(false);
      }
    });
  }

  goBack() {
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }
    this.stopAutosave();
    this.currentAssessment.set(null);
    this.studentResults.set([]);
    if (this.auth.hasRole('ADMIN', 'TEACHER')) {
      this.loadTeacherAssessments();
    } else {
      this.loadStudentAssessments();
    }
  }

  startAutosave() {
    this.autosaveInterval = setInterval(() => {
      if (this.mode() === 'taking' && this.currentAssessment()) {
        this.isSaving.set(true);
        this.assessmentService.autosave(
          this.currentAssessment()!.id,
          { ...this.answers() },
          this.currentIndex(),
          this.timeSpentSeconds()
        ).subscribe({
          next: () => {
            this.isSaving.set(false);
            this.lastSaved.set(new Date().toLocaleTimeString('es-ES'));
          },
          error: () => this.isSaving.set(false)
        });
      }
    }, 30000);
  }

  stopAutosave() {
    if (this.autosaveInterval) {
      clearInterval(this.autosaveInterval);
    }
  }

  checkForResume(assessmentId: string) {
    this.assessmentService.resumeAssessment(assessmentId).subscribe({
      next: (session) => {
        if (session && session.answers) {
          this.answers.set({ ...session.answers });
          this.currentIndex.set(session.current_index || 0);
          this.timeSpentSeconds.set(session.time_spent_seconds || 0);
        }
      },
      error: () => {}
    });
  }

  ngOnDestroy() {
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }
    this.stopAutosave();
  }
}
