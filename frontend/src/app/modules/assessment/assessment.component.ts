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
import { AssessmentService, Assessment, AssessmentQuestion, StudentAssessment, StudentAnswer } from '../../core/services/assessment.service';

@Component({
  selector: 'app-assessment',
  standalone: true,
  imports: [
    CommonModule, FormsModule, MatCardModule, MatButtonModule, MatIconModule,
    MatProgressBarModule, MatRadioModule, MatInputModule, MatFormFieldModule,
    MatTabsModule, MatChipsModule, MatSnackBarModule
  ],
  template: `
    @if (loading()) {
      <div class="loading-container">
        <mat-progress-bar mode="indeterminate"></mat-progress-bar>
        <p>Cargando evaluacion...</p>
      </div>
    } @else if (error()) {
      <div class="error-container">
        <mat-icon color="warn">error</mat-icon>
        <p>{{ error() }}</p>
        <button mat-raised-button color="primary" (click)="goBack()">Volver</button>
      </div>
    } @else if (mode() === 'list') {
      <div class="assessment-list">
        <h2>Evaluaciones Disponibles</h2>
        @for (assessment of assessments(); track assessment.id) {
          <mat-card class="assessment-card" (click)="selectAssessment(assessment)">
            <mat-card-header>
              <mat-card-title>{{ assessment.title }}</mat-card-title>
              <mat-card-subtitle>
                {{ assessment.assessment_type | titlecase }} . {{ assessment.total_points }} puntos
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
      <div class="assessment-taking">
        <div class="assessment-header">
          <h2>{{ currentAssessment()?.title }}</h2>
          @if (timeRemaining() !== null) {
            <div class="timer" [class.warning]="timeRemaining()! < 300">
              <mat-icon>timer</mat-icon>
              {{ formatTime(timeRemaining()!) }}
            </div>
          }
          <div class="progress">
            Pregunta {{ currentIndex() + 1 }} de {{ questions().length }}
          </div>
        </div>

        <mat-progress-bar [value]="((currentIndex() + 1) / questions().length) * 100"></mat-progress-bar>

        @if (currentQuestion()) {
          <mat-card class="question-card">
            <mat-card-header>
              <mat-card-subtitle>
                Pregunta {{ currentIndex() + 1 }} . {{ currentQuestion()!.points }} puntos
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
                <input matInput [(ngModel)]="currentAnswer" placeholder="Escribe tu respuesta...">
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
                  <mat-icon>send</mat-icon> Enviar Evaluacion
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
      <div class="assessment-results">
        <mat-card class="results-header">
          <mat-card-header>
            <mat-card-title>Resultados de la Evaluacion</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <div class="score-display">
              <div class="score-circle" [class.passed]="result()!.passed">
                {{ (result()!.percentage * 100) | number:'1.0-0' }}%
              </div>
              <div class="score-details">
                <p>{{ result()!.total_score }} / {{ result()!.max_score }} puntos</p>
                <p [class]="result()!.passed ? 'passed' : 'failed'">
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
                  Pregunta {{ i + 1 }} . {{ answer.points_earned }}/{{ answer.points_possible }} puntos
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
                  <p>{{ answer.feedback }}</p>
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
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 400px;
      gap: 16px;
    }
    .assessment-list {
      max-width: 800px;
      margin: 0 auto;
      padding: 24px;
    }
    .assessment-card {
      margin-bottom: 16px;
      cursor: pointer;
      transition: transform 0.2s;
    }
    .assessment-card:hover {
      transform: translateY(-2px);
    }
    .assessment-meta {
      display: flex;
      gap: 8px;
      margin-top: 12px;
    }
    .assessment-taking {
      max-width: 800px;
      margin: 0 auto;
      padding: 24px;
    }
    .assessment-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
    }
    .timer {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 1.2em;
      font-weight: bold;
    }
    .timer.warning {
      color: #f44336;
    }
    .question-card {
      margin: 24px 0;
    }
    .question-latex, .question-statement {
      font-size: 1.1em;
      margin-bottom: 24px;
    }
    .answer-field {
      width: 100%;
    }
    mat-card-actions {
      display: flex;
      align-items: center;
    }
    .spacer {
      flex: 1;
    }
    .question-nav {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-top: 24px;
      justify-content: center;
    }
    .results-header {
      text-align: center;
      margin-bottom: 24px;
    }
    .score-display {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 32px;
      padding: 32px;
    }
    .score-circle {
      font-size: 3em;
      font-weight: bold;
      width: 150px;
      height: 150px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      border: 8px solid #f44336;
    }
    .score-circle.passed {
      border-color: #4caf50;
    }
    .passed { color: #4caf50; }
    .failed { color: #f44336; }
    .correct-text { color: #4caf50; }
    .incorrect-text { color: #f44336; }
    .answers-breakdown {
      margin-bottom: 24px;
    }
    .answer-card {
      margin-bottom: 12px;
      border-left: 4px solid;
    }
    .answer-card.correct {
      border-left-color: #4caf50;
    }
    .answer-card.incorrect {
      border-left-color: #f44336;
    }
  `]
})
export class AssessmentComponent implements OnInit {
  loading = signal(true);
  error = signal<string | null>(null);
  mode = signal<'list' | 'taking' | 'results'>('list');
  assessments = signal<Assessment[]>([]);
  currentAssessment = signal<Assessment | null>(null);
  questions = signal<AssessmentQuestion[]>([]);
  currentIndex = signal(0);
  currentAnswer = '';
  answers = signal<Record<string, string>>({});
  studentAssessment = signal<StudentAssessment | null>(null);
  timeRemaining = signal<number | null>(null);
  result = signal<StudentAssessment | null>(null);
  resultAnswers = signal<StudentAnswer[]>([]);

  private timerInterval: any;

  currentQuestion = computed(() => {
    const qs = this.questions();
    const idx = this.currentIndex();
    return qs.length > idx ? qs[idx] : null;
  });

  constructor(
    private assessmentService: AssessmentService,
    private route: ActivatedRoute,
    private router: Router,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit() {
    this.loadAssessments();
  }

  loadAssessments() {
    this.loading.set(true);
    this.assessmentService.listAssessments(undefined, undefined, 'published').subscribe({
      next: (assessments) => {
        this.assessments.set(assessments || []);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set('Error al cargar evaluaciones');
        this.loading.set(false);
      }
    });
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
        if (assessment.time_limit_minutes > 0) {
          this.startTimer(assessment.time_limit_minutes * 60);
        }
      },
      error: (err) => {
        this.snackBar.open(err.error?.error || 'Error al iniciar evaluacion', 'Cerrar', { duration: 3000 });
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
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }

    const assessment = this.currentAssessment();
    if (!assessment) return;

    const answersArray = Object.entries(this.answers()).map(([question_id, answer]) => ({
      question_id,
      answer,
      procedure: []
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
      error: (err) => {
        this.snackBar.open('Error al enviar evaluacion', 'Cerrar', { duration: 3000 });
        this.loading.set(false);
      }
    });
  }

  goBack() {
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }
    this.mode.set('list');
    this.currentAssessment.set(null);
    this.loadAssessments();
  }

  ngOnDestroy() {
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }
  }
}
