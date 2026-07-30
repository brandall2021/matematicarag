import { Component, signal, ViewChild, ElementRef, AfterViewInit, OnDestroy, NgZone } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { MathfieldElement } from 'mathlive';
import { RenderMathPipe } from '../../shared/render-math.pipe';

MathfieldElement.fontsDirectory = 'https://cdn.jsdelivr.net/npm/mathlive@0.110.0/fonts';

@Component({
  selector: 'app-math',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule, RenderMathPipe],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  template: `
    <div class="math-container">
      <header class="math-header">
        <div>
          <h1>Motor Matemático</h1>
          <p class="math-subtitle">Resolvé expresiones simbólicas con asistencia de SymPy</p>
        </div>
      </header>

      <div class="mathfield-wrapper">
        <math-field
          #mathField
          id="mathfield"
          virtual-keyboard-mode="auto"
          smart-fence
          smart-superscript
          (input)="onInput($event)"
          (keydown.enter)="calculate()"
          class="mathfield-element"
        ></math-field>
      </div>

      <div class="input-row">
        <div class="latex-chip">
          <span class="latex-dot"></span>
          <span class="latex-label">LaTeX:</span>
          <code class="latex-value">{{ latexValue() || '—' }}</code>
        </div>
        <button class="calc-btn" (click)="calculate()" [disabled]="!latexValue() || loading()">
          @if (loading()) {
            <span class="calc-spinner"></span>
            <span>Calculando</span>
          } @else {
            <mat-icon>calculate</mat-icon>
            <span>Calcular</span>
          }
        </button>
      </div>

      @if (result()) {
        <div class="result-card">
          <div class="result-card-header">
            <mat-icon class="result-icon">check_circle</mat-icon>
            <h3>Resultado</h3>
          </div>
          <div class="result-content" [innerHTML]="result() | renderMath"></div>
        </div>
      }
    </div>
  `,
  styles: [`
    .math-container {
      padding: var(--space-xl);
      max-width: 800px;
      margin: 0 auto;
      color: var(--text);
    }
    @media (min-width: 768px) { .math-container { padding: var(--space-2xl); } }

    .math-header {
      margin-bottom: var(--space-xl);
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: var(--space-md);
    }
    .math-header h1 {
      font-family: var(--font-serif);
      font-size: 1.5rem;
      font-weight: 600;
      color: var(--text);
      margin: 0;
      letter-spacing: -0.02em;
    }
    .math-subtitle {
      margin: var(--space-xs) 0 0;
      font-size: 0.85rem;
      color: var(--text-secondary);
    }

    .mathfield-wrapper {
      background: var(--surface);
      border-radius: var(--radius-lg);
      padding: var(--space-lg);
      margin-bottom: var(--space-md);
      border: 2px solid var(--border);
      transition: all 0.2s ease;
      box-shadow: var(--shadow-sm);
    }
    .mathfield-wrapper:focus-within {
      border-color: var(--accent);
      box-shadow: 0 0 0 4px var(--accent-muted), var(--shadow-sm);
    }
    .mathfield-element { width: 100%; font-size: 1.3rem; min-height: 56px; }
    :host ::ng-deep math-field {
      --math-field-border: none;
      --math-field-border-radius: 0;
      --math-field-background: transparent;
      --math-field-color: var(--text);
      font-size: 1.3rem;
    }
    :host ::ng-deep .ML__virtual-keyboard {
      background: var(--surface) !important;
      border-radius: var(--radius-md) !important;
    }

    .input-row {
      display: flex;
      gap: var(--space-sm);
      margin-bottom: var(--space-xl);
      align-items: center;
    }
    .latex-chip {
      flex: 1;
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      background: var(--surface);
      padding: 0.6rem var(--space-md);
      border-radius: var(--radius-md);
      border: 1px solid var(--border);
      min-width: 0;
    }
    .latex-dot {
      width: 6px; height: 6px; border-radius: 50%;
      background: var(--accent); flex-shrink: 0;
    }
    .latex-label {
      color: var(--text-tertiary);
      font-size: 0.75rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      flex-shrink: 0;
    }
    .latex-value {
      color: var(--accent-text);
      font-family: var(--font-mono, monospace);
      font-size: 0.85rem;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .calc-btn {
      display: inline-flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 0.65rem 1.25rem;
      border: none;
      border-radius: var(--radius-md);
      background: var(--accent);
      color: #0f0f1a;
      font-family: var(--font-sans);
      font-size: 0.9rem;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.15s ease;
      white-space: nowrap;
      flex-shrink: 0;
    }
    .calc-btn:hover:not(:disabled) {
      background: var(--accent-hover);
      transform: translateY(-1px);
      box-shadow: var(--shadow-md);
    }
    .calc-btn:active:not(:disabled) {
      transform: translateY(0);
    }
    .calc-btn:disabled {
      opacity: 0.4;
      cursor: not-allowed;
      transform: none;
    }
    .calc-btn mat-icon { font-size: 20px; width: 20px; height: 20px; }
    .calc-spinner {
      width: 16px; height: 16px;
      border: 2px solid rgba(15,15,26,0.2);
      border-top-color: #0f0f1a;
      border-radius: 50%;
      animation: spin 0.6s linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }

    .result-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      padding: var(--space-lg);
      box-shadow: var(--shadow-sm);
      animation: fadeUp 0.25s ease;
    }
    @keyframes fadeUp {
      from { opacity: 0; transform: translateY(8px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .result-card-header {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      margin-bottom: var(--space-md);
      padding-bottom: var(--space-sm);
      border-bottom: 1px solid var(--border-light);
    }
    .result-icon { color: var(--success); font-size: 20px; width: 20px; height: 20px; }
    .result-card-header h3 {
      font-family: var(--font-serif);
      font-size: 1.05rem;
      font-weight: 600;
      color: var(--text);
      margin: 0;
    }
    .result-content {
      font-size: 1rem;
      color: var(--text);
      line-height: 1.7;
    }
    .result-content p { margin: 0.5rem 0; }
    .result-content .katex-display { margin: 1rem 0; overflow-x: auto; padding: 0.5rem 0; }
    .result-content .katex { font-size: 1.1em; }
  `]
})
export class MathComponent implements AfterViewInit, OnDestroy {
  @ViewChild('mathField') mathFieldRef!: ElementRef<any>;

  result = signal('');
  latexValue = signal('');
  loading = signal(false);

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

  calculate() {
    const expr = this.latexValue();
    if (!expr) return;

    this.loading.set(true);
    this.result.set('');

    this.api.mathEvaluate(expr).subscribe({
      next: (res: any) => {
        this.loading.set(false);
        this.result.set(res.result || res.error || 'Sin resultado');
      },
      error: (err: any) => {
        this.loading.set(false);
        this.result.set(err.error?.error || 'Error al calcular');
      }
    });
  }
}
