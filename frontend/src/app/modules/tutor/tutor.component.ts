import { Component, signal, ViewChild, ElementRef, AfterViewInit, OnDestroy, NgZone } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService, TutorRequest, TutorResponse } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { MathfieldElement } from 'mathlive';
import { RenderMathPipe } from '../../shared/render-math.pipe';

MathfieldElement.fontsDirectory = 'https://cdn.jsdelivr.net/npm/mathlive@0.110.0/fonts';

@Component({
  selector: 'app-tutor',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule, MatButtonToggleModule, RenderMathPipe],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  template: `
    <div class="tutor-container">
      <div class="tutor-header">
        <h1><mat-icon>school</mat-icon> Tutor Matematico</h1>
        <p class="subtitle">Resuelve, verifica o recibe hints paso a paso</p>
      </div>

      <div class="controls-row">
        <div class="control-group">
          <label>Modo</label>
          <mat-button-toggle-group [value]="mode()" (change)="mode.set($event.value)">
            <mat-button-toggle value="solve">Resolver</mat-button-toggle>
            <mat-button-toggle value="verify">Verificar</mat-button-toggle>
            <mat-button-toggle value="hint">Hint</mat-button-toggle>
          </mat-button-toggle-group>
        </div>
        <div class="control-group">
          <label>Nivel</label>
          <mat-button-toggle-group [value]="level()" (change)="level.set($event.value)">
            <mat-button-toggle value="basic">Basico</mat-button-toggle>
            <mat-button-toggle value="intermediate">Intermedio</mat-button-toggle>
            <mat-button-toggle value="advanced">Avanzado</mat-button-toggle>
          </mat-button-toggle-group>
        </div>
      </div>

      <div class="input-section">
        <div class="mathfield-wrapper">
          <label>Expresion matematica</label>
          <math-field
            #mathField
            virtual-keyboard-mode="auto"
            smart-fence
            smart-superscript
            (input)="onInput($event)"
            (keydown.enter)="submit()"
            class="mathfield-element"
          ></math-field>
          <div class="latex-preview">
            <span class="latex-label">LaTeX:</span>
            <code>{{ latexValue() }}</code>
          </div>
        </div>

        <div class="text-query-wrapper">
          <label>Consulta en texto (opcional)</label>
          <input
            [(ngModel)]="textQuery"
            placeholder="Describe el problema..."
            class="text-input"
            (keydown.enter)="submit()"
          />
        </div>
      </div>

      <div class="submit-area">
        <button
          mat-raised-button
          color="primary"
          (click)="submit()"
          [disabled]="loading() || (!latexValue() && !textQuery)"
        >
          <mat-icon>{{ loading() ? 'sync' : 'send' }}</mat-icon>
          {{ loading() ? 'Resolviendo...' : 'Resolver' }}
        </button>
      </div>

      @if (error()) {
        <div class="error-box">
          <mat-icon>error</mat-icon>
          <span>{{ error() }}</span>
        </div>
      }

      @if (result()) {
        <div class="result-section">
          <div class="result-card problem-card">
            <div class="card-header">
              <mat-icon>extension</mat-icon>
              <h3>Problema identificado</h3>
            </div>
            <div class="card-body">
              <span class="problem-type">{{ result()!.problem.type }}</span>
              <span class="problem-expr" [innerHTML]="'$' + result()!.problem.expression + '$' | renderMath"></span>
              @if (result()!.problem.variable) {
                <span class="problem-var">Variable: {{ result()!.problem.variable }}</span>
              }
            </div>
          </div>

          <div class="result-card method-card">
            <div class="card-header">
              <mat-icon>menu_book</mat-icon>
              <h3>Metodo</h3>
            </div>
            <div class="card-body">
              <strong>{{ result()!.method.name }}</strong>
              <span class="method-desc">{{ result()!.method.description }}</span>
            </div>
          </div>

          @if (result()!.steps && result()!.steps.length > 0) {
            <div class="steps-section">
              <h3>Pasos</h3>
              @for (step of result()!.steps; track step.number) {
                <div class="step-card">
                  <div class="step-number">{{ step.number }}</div>
                  <div class="step-content">
                    <div class="step-title">{{ step.title }}</div>
                    <div class="step-explanation">{{ step.explanation }}</div>
                    @if (step.latex && step.is_math) {
                      <div class="step-math" [innerHTML]="'$' + step.latex + '$' | renderMath"></div>
                    }
                  </div>
                </div>
              }
            </div>
          }

          @if (result()!.result) {
            <div class="result-card final-result">
              <div class="card-header">
                <mat-icon>check_circle</mat-icon>
                <h3>Resultado</h3>
              </div>
              <div class="card-body">
                <div class="result-latex" [innerHTML]="'$$' + (result()!.result?.latex ?? '') + '$$' | renderMath"></div>
                <span class="result-text">{{ result()!.result?.result }}</span>
              </div>
            </div>
          }

          @if (result()!.verification) {
            <div class="verification-badge" [attr.data-status]="result()!.verification?.status">
              @switch (result()!.verification?.status) {
                @case ('verified') {
                  <mat-icon>verified</mat-icon>
                  <span>Verificado</span>
                }
                @case ('not_verified') {
                  <mat-icon>warning</mat-icon>
                  <span>No verificado</span>
                }
                @case ('failed') {
                  <mat-icon>cancel</mat-icon>
                  <span>Fallo la verificacion</span>
                }
              }
              @if (result()!.verification?.method) {
                <span class="verify-method">({{ result()!.verification!.method }})</span>
              }
            </div>
          }

          @if (result()!.math_computed) {
            <div class="math-computed-badge">
              <mat-icon>calculate</mat-icon>
              <span>Calculado con motor matematico</span>
            </div>
          }

          @if (result()!.sources && result()!.sources.length > 0) {
            <div class="sources-section">
              <div class="sources-label">
                <mat-icon>menu_book</mat-icon> Fuentes
              </div>
              <div class="source-chips">
                @for (src of result()!.sources; track $index) {
                  <button class="source-chip" [class.expanded]="expandedSource === $index" (click)="toggleSource($index)">
                    <mat-icon class="source-icon">description</mat-icon>
                    {{ src.filename || src.title || 'Fuente' }}
                    @if (src.page) {
                      <span class="source-page">p.{{ src.page }}</span>
                    }
                  </button>
                }
              </div>
              @if (expandedSource !== null) {
                <div class="source-detail">
                  @for (src of result()!.sources; track $index) {
                    @if (expandedSource === $index) {
                      <div class="citation-card">
                        <div class="citation-card-header">
                          <strong>{{ src.filename || src.title }}</strong>
                          @if (src.page) {
                            <span> (pagina {{ src.page }})</span>
                          }
                        </div>
                        @if (src.content) {
                          <div class="citation-card-content">{{ src.content }}</div>
                        }
                      </div>
                    }
                  }
                </div>
              }
            </div>
          }

          @if (result()!.citations && result()!.citations.length > 0) {
            <div class="citations-section">
              <div class="sources-label">
                <mat-icon>format_quote</mat-icon> Citas
              </div>
              <div class="source-chips">
                @for (cit of result()!.citations; track $index) {
                  <button class="source-chip citation-chip" (click)="expandedCitation = expandedCitation === $index ? null : $index">
                    <mat-icon class="source-icon">format_quote</mat-icon>
                    {{ cit.text || cit.title || 'Cita' }}
                  </button>
                }
              </div>
            </div>
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .tutor-container { padding: 1.5rem; max-width: 900px; margin: 0 auto; background: var(--bg); color: var(--text); }
    @media (min-width: 768px) { .tutor-container { padding: 2rem; } }

    .tutor-header { margin-bottom: 1.5rem; }
    .tutor-header h1 { color: var(--accent); font-family: 'Newsreader', serif; margin: 0 0 0.25rem 0; display: flex; align-items: center; gap: 0.5rem; }
    .tutor-header h1 mat-icon { font-size: 28px; width: 28px; height: 28px; }
    .subtitle { color: var(--text-secondary); margin: 0; font-size: 0.9rem; }

    .controls-row { display: flex; gap: 1.5rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
    .control-group { display: flex; flex-direction: column; gap: 0.4rem; }
    .control-group label { font-size: 0.75rem; color: var(--text-secondary); font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; }
    :host ::ng-deep .mat-button-toggle-group { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; }
    :host ::ng-deep .mat-button-toggle { color: var(--text-secondary); }
    :host ::ng-deep .mat-button-toggle-checked { background: var(--accent) !important; color: var(--bg) !important; }
    :host ::ng-deep .mat-button-toggle-button { background: transparent; color: inherit; }
    :host ::ng-deep .mat-button-toggle-bar { display: flex; }
    :host ::ng-deep .mat-button-toggle + .mat-button-toggle { border-left: 1px solid var(--border); }

    .input-section { display: flex; flex-direction: column; gap: 1rem; margin-bottom: 1rem; }
    .mathfield-wrapper { background: var(--surface); border-radius: 12px; padding: 1.25rem; border: 2px solid var(--border); }
    .mathfield-wrapper:focus-within { border-color: var(--accent); }
    .mathfield-wrapper label, .text-query-wrapper label { display: block; font-size: 0.75rem; color: var(--text-secondary); font-weight: 600; margin-bottom: 0.5rem; text-transform: uppercase; letter-spacing: 0.05em; }
    .mathfield-element { width: 100%; font-size: 1.3rem; min-height: 50px; }
    :host ::ng-deep math-field {
      --math-field-border: none;
      --math-field-border-radius: 0;
      --math-field-background: transparent;
      --math-field-color: var(--text);
      font-size: 1.3rem;
    }
    :host ::ng-deep .ML__virtual-keyboard { background: var(--surface) !important; }
    .latex-preview { display: flex; align-items: center; gap: 0.5rem; margin-top: 0.5rem; padding: 0.5rem 0.75rem; background: var(--bg); border-radius: 6px; }
    .latex-label { color: var(--text-secondary); font-size: 0.7rem; font-weight: 600; }
    .latex-preview code { color: var(--accent); font-size: 0.85rem; word-break: break-all; }

    .text-query-wrapper { background: var(--surface); border-radius: 12px; padding: 1.25rem; border: 2px solid var(--border); }
    .text-query-wrapper:focus-within { border-color: var(--accent); }
    .text-input { width: 100%; padding: 0.75rem; border-radius: 8px; border: 1px solid var(--border); background: var(--bg); color: var(--text); font-size: 1rem; outline: none; box-sizing: border-box; }
    .text-input:focus { border-color: var(--accent); }

    .submit-area { margin-bottom: 1.5rem; }

    .error-box { display: flex; align-items: center; gap: 0.5rem; background: rgba(244, 67, 54, 0.1); border: 1px solid rgba(244, 67, 54, 0.3); border-radius: 8px; padding: 0.75rem 1rem; color: #ef5350; margin-bottom: 1.5rem; }

    .result-section { display: flex; flex-direction: column; gap: 1rem; }
    .result-card { background: var(--surface); border-radius: 12px; border: 1px solid var(--border); overflow: hidden; }
    .card-header { display: flex; align-items: center; gap: 0.5rem; padding: 0.75rem 1rem; border-bottom: 1px solid var(--border); }
    .card-header mat-icon { font-size: 20px; width: 20px; height: 20px; color: var(--accent); }
    .card-header h3 { margin: 0; font-size: 0.9rem; color: var(--text); }
    .card-body { padding: 0.75rem 1rem; display: flex; flex-direction: column; gap: 0.35rem; }
    .problem-type { font-size: 0.8rem; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.05em; }
    .problem-expr { font-size: 1.1rem; }
    .problem-var { font-size: 0.85rem; color: var(--text-secondary); }
    .method-desc { font-size: 0.85rem; color: var(--text-secondary); margin-top: 0.25rem; }

    .steps-section h3 { color: var(--accent); margin: 0.5rem 0; font-size: 1rem; }
    .step-card { display: flex; gap: 0.75rem; background: var(--surface); border-radius: 10px; border: 1px solid var(--border); padding: 1rem; margin-bottom: 0.5rem; }
    .step-number { min-width: 32px; height: 32px; border-radius: 50%; background: var(--accent); color: var(--bg); display: flex; align-items: center; justify-content: center; font-weight: 700; font-size: 0.85rem; flex-shrink: 0; }
    .step-content { flex: 1; min-width: 0; }
    .step-title { font-weight: 600; font-size: 0.9rem; margin-bottom: 0.25rem; }
    .step-explanation { font-size: 0.85rem; color: var(--text-secondary); line-height: 1.5; }
    .step-math { margin-top: 0.5rem; overflow-x: auto; }
    .step-math :deep(.katex-display) { margin: 0.5rem 0; }

    .final-result { border-color: var(--accent); }
    .result-latex { overflow-x: auto; margin-bottom: 0.5rem; }
    .result-text { font-size: 0.85rem; color: var(--text-secondary); }

    .verification-badge { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.5rem 0.75rem; border-radius: 8px; font-size: 0.85rem; font-weight: 600; }
    .verification-badge[data-status="verified"] { background: rgba(76, 175, 80, 0.12); color: #4caf50; border: 1px solid rgba(76, 175, 80, 0.3); }
    .verification-badge[data-status="verified"] mat-icon { color: #4caf50; }
    .verification-badge[data-status="not_verified"] { background: rgba(255, 152, 0, 0.12); color: #ff9800; border: 1px solid rgba(255, 152, 0, 0.3); }
    .verification-badge[data-status="not_verified"] mat-icon { color: #ff9800; }
    .verification-badge[data-status="failed"] { background: rgba(244, 67, 54, 0.12); color: #ef5350; border: 1px solid rgba(244, 67, 54, 0.3); }
    .verification-badge[data-status="failed"] mat-icon { color: #ef5350; }
    .verification-badge mat-icon { font-size: 18px; width: 18px; height: 18px; }
    .verify-method { font-weight: 400; font-size: 0.8rem; opacity: 0.7; margin-left: 0.25rem; }

    .math-computed-badge { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.4rem 0.65rem; border-radius: 6px; font-size: 0.78rem; background: rgba(200, 170, 118, 0.12); color: #c8aa76; border: 1px solid rgba(200, 170, 118, 0.25); }
    .math-computed-badge mat-icon { font-size: 15px; width: 15px; height: 15px; }

    .sources-section, .citations-section { margin-top: 0.5rem; }
    .sources-label { display: inline-flex; align-items: center; gap: 0.3rem; font-size: 0.75rem; color: var(--text-secondary); font-weight: 600; margin-bottom: 0.4rem; }
    .sources-label mat-icon { font-size: 14px; width: 14px; height: 14px; }
    .source-chips { display: flex; flex-wrap: wrap; gap: 0.4rem; }
    .source-chip { display: inline-flex; align-items: center; gap: 0.3rem; padding: 0.2rem 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface); color: var(--text-secondary); font-size: 0.72rem; cursor: pointer; transition: all 0.15s; }
    .source-chip:hover { border-color: var(--accent); color: var(--accent); background: var(--bg); }
    .source-chip.expanded { background: rgba(200, 170, 118, 0.15); border-color: #c8aa76; color: #c8aa76; }
    .source-icon { font-size: 12px; width: 12px; height: 12px; }
    .source-page { color: var(--accent); font-weight: 600; }
    .source-detail { margin-top: 0.5rem; }
    .citation-card { background: rgba(0, 0, 0, 0.2); border: 1px solid var(--border); border-radius: 8px; padding: 0.65rem 0.85rem; }
    .citation-card-header { font-size: 0.7rem; color: var(--text-secondary); margin-bottom: 0.35rem; display: flex; align-items: center; flex-wrap: wrap; }
    .citation-card-content { font-size: 0.72rem; color: var(--text-secondary); line-height: 1.5; font-style: italic; }

    .citation-chip { border-style: dashed; }
  `]
})
export class TutorComponent implements AfterViewInit, OnDestroy {
  @ViewChild('mathField') mathFieldRef!: ElementRef<any>;

  result = signal<TutorResponse | null>(null);
  latexValue = signal('');
  loading = signal(false);
  error = signal('');
  mode = signal<'solve' | 'verify' | 'hint'>('solve');
  level = signal<'basic' | 'intermediate' | 'advanced'>('intermediate');

  textQuery = '';
  expandedSource: number | null = null;
  expandedCitation: number | null = null;

  private mf: any = null;

  constructor(private api: ApiService, private zone: NgZone) {}

  ngAfterViewInit() {
    setTimeout(() => {
      this.mf = this.mathFieldRef.nativeElement;
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
        this.latexValue.set(this.mf.value || '');
      }
    });
  };

  onInput(event: any) {
    const val = event.target?.value || this.mf?.value || '';
    this.latexValue.set(val);
  }

  toggleSource(index: number) {
    this.expandedSource = this.expandedSource === index ? null : index;
  }

  submit() {
    const latex = this.latexValue();
    const query = this.textQuery || latex;
    if (!query) return;

    this.loading.set(true);
    this.result.set(null);
    this.error.set('');

    const request: TutorRequest = {
      query,
      mode: this.mode(),
      explanation_level: this.level(),
    };

    this.api.tutorSolve(request).subscribe({
      next: (res) => {
        this.loading.set(false);
        this.result.set(res);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.error || err.error?.detail || 'Error al resolver el problema');
      }
    });
  }
}
