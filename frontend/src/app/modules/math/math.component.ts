import { Component, signal, ViewChild, ElementRef, AfterViewInit, OnDestroy, NgZone } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { MathfieldElement } from 'mathlive';

MathfieldElement.fontsDirectory = 'https://cdn.jsdelivr.net/npm/mathlive@0.110.0/fonts';

@Component({
  selector: 'app-math',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  template: `
    <div class="math-container">
      <h1>Motor Matematico</h1>

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

      <div class="input-area">
        <div class="latex-preview">
          <span class="latex-label">LaTeX:</span>
          <code>{{ latexValue() }}</code>
        </div>
        <button mat-raised-button color="primary" (click)="calculate()" [disabled]="!latexValue() || loading()">
          <mat-icon>{{ loading() ? 'sync' : 'play_arrow' }}</mat-icon>
          {{ loading() ? 'Calculando...' : 'Calcular' }}
        </button>
      </div>

      @if (result()) {
        <div class="result">
          <h3>Resultado:</h3>
          <pre>{{ result() }}</pre>
        </div>
      }
    </div>
  `,
  styles: [`
    .math-container { padding: 2rem; max-width: 900px; margin: 0 auto; background: var(--bg); color: var(--text); min-height: 100vh; }
    h1 { color: var(--accent); font-family: 'Newsreader', serif; margin-bottom: 1.5rem; }
    .mathfield-wrapper { background: var(--surface); border-radius: 12px; padding: 1.5rem; margin-bottom: 1.5rem; border: 2px solid var(--border); }
    .mathfield-wrapper:focus-within { border-color: var(--accent); }
    .mathfield-element { width: 100%; font-size: 1.4rem; min-height: 60px; }
    :host ::ng-deep math-field {
      --math-field-border: none;
      --math-field-border-radius: 0;
      --math-field-background: transparent;
      --math-field-color: var(--text);
      font-size: 1.4rem;
    }
    :host ::ng-deep .ML__virtual-keyboard {
      background: var(--surface) !important;
    }
    .input-area { display: flex; gap: 0.5rem; margin-bottom: 2rem; align-items: center; }
    .latex-preview { flex: 1; display: flex; align-items: center; gap: 0.5rem; background: var(--surface); padding: 0.75rem 1rem; border-radius: 8px; }
    .latex-label { color: var(--text-secondary); font-size: 0.8rem; font-weight: 600; }
    .latex-preview code { color: var(--accent); font-size: 0.95rem; word-break: break-all; }
    .result { background: var(--surface); padding: 1.5rem; border-radius: 12px; }
    .result h3 { color: var(--accent); margin-top: 0; }
    pre { white-space: pre-wrap; font-size: 1.1rem; color: var(--text); }
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
