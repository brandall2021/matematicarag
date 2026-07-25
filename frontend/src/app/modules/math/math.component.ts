import { Component, signal, ViewChild, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-math',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule, MatIconModule],
  template: `
    <div class="math-container">
      <h1>Motor Matematico</h1>
      <div class="operations">
        <button mat-raised-button (click)="op.set('evaluate')" [class.active]="op() === 'evaluate'">Evaluar</button>
        <button mat-raised-button (click)="op.set('derive')" [class.active]="op() === 'derive'">Derivar</button>
        <button mat-raised-button (click)="op.set('integrate')" [class.active]="op() === 'integrate'">Integrar</button>
        <button mat-raised-button (click)="op.set('solve')" [class.active]="op() === 'solve'">Resolver</button>
        <button mat-raised-button (click)="op.set('simplify')" [class.active]="op() === 'simplify'">Simplificar</button>
        <button mat-raised-button (click)="op.set('factor')" [class.active]="op() === 'factor'">Factorizar</button>
      </div>

      <div class="equation-bar">
        <div class="bar-group">
          <span class="bar-label">Basico</span>
          <div class="bar-buttons">
            @for (sym of basicSymbols; track sym) {
              <button class="sym-btn" (click)="insert(sym)">{{ sym }}</button>
            }
          </div>
        </div>
        <div class="bar-group">
          <span class="bar-label">Griego</span>
          <div class="bar-buttons">
            @for (sym of greekSymbols; track sym) {
              <button class="sym-btn" (click)="insert(sym)">{{ sym }}</button>
            }
          </div>
        </div>
        <div class="bar-group">
          <span class="bar-label">Funciones</span>
          <div class="bar-buttons">
            @for (sym of funcSymbols; track sym) {
              <button class="sym-btn" (click)="insert(sym)">{{ sym }}</button>
            }
          </div>
        </div>
        <div class="bar-group">
          <span class="bar-label">Avanzado</span>
          <div class="bar-buttons">
            @for (sym of advancedSymbols; track sym) {
              <button class="sym-btn" (click)="insert(sym)">{{ sym }}</button>
            }
          </div>
        </div>
      </div>

      <div class="input-area">
        <input #mathInput [(ngModel)]="expression" placeholder="Escribi tu ecuacion..." class="math-input" (keydown.enter)="calculate()">
        <button mat-raised-button color="primary" (click)="calculate()" [disabled]="!expression">
          <mat-icon>play_arrow</mat-icon> Calcular
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
    .operations { display: flex; gap: 0.5rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
    .operations button { font-size: 0.85rem; }
    .operations button.active { background: var(--accent); color: var(--bg); }
    .equation-bar { background: var(--surface); border-radius: 12px; padding: 1rem; margin-bottom: 1.5rem; display: flex; flex-direction: column; gap: 0.75rem; }
    .bar-group { display: flex; align-items: center; gap: 0.5rem; }
    .bar-label { color: var(--text-secondary); font-size: 0.75rem; width: 70px; flex-shrink: 0; text-transform: uppercase; letter-spacing: 0.5px; }
    .bar-buttons { display: flex; flex-wrap: wrap; gap: 4px; }
    .sym-btn { width: 36px; height: 32px; border: 1px solid var(--border); border-radius: 6px; background: var(--bg); color: var(--text); font-size: 0.95rem; cursor: pointer; display: flex; align-items: center; justify-content: center; transition: all 0.15s; }
    .sym-btn:hover { background: var(--accent); color: var(--bg); border-color: var(--accent); }
    .input-area { display: flex; gap: 0.5rem; margin-bottom: 2rem; }
    .math-input { flex: 1; padding: 0.75rem 1rem; border-radius: 8px; border: 2px solid var(--border); background: var(--input-bg); color: var(--text); font-size: 1.2rem; font-family: 'Courier New', monospace; outline: none; }
    .math-input:focus { border-color: var(--accent); }
    .result { background: var(--surface); padding: 1.5rem; border-radius: 12px; }
    .result h3 { color: var(--accent); margin-top: 0; }
    pre { white-space: pre-wrap; font-size: 1.1rem; color: var(--text); }
  `]
})
export class MathComponent {
  @ViewChild('mathInput') mathInput!: ElementRef<HTMLInputElement>;

  expression = '';
  op = signal('evaluate');
  result = signal('');

  basicSymbols = ['+', '-', '*', '/', '^', '(', ')', '=', '|', '!', '%'];
  greekSymbols = ['α', 'β', 'γ', 'δ', 'ε', 'θ', 'λ', 'μ', 'π', 'σ', 'φ', 'ω', '∞'];
  funcSymbols = ['sin', 'cos', 'tan', 'log', 'ln', 'sqrt', 'abs', 'exp', 'asin', 'acos', 'atan'];
  advancedSymbols = ['∫', '∑', '∏', '√', '∂', '∇', '≤', '≥', '≠', '≈', '±', '×', '÷'];

  constructor(private api: ApiService) {}

  insert(sym: string) {
    const input = this.mathInput.nativeElement;
    const start = input.selectionStart || this.expression.length;
    const end = input.selectionEnd || this.expression.length;
    this.expression = this.expression.slice(0, start) + sym + this.expression.slice(end);
    setTimeout(() => {
      input.focus();
      input.setSelectionRange(start + sym.length, start + sym.length);
    });
  }

  calculate() {
    if (!this.expression) return;
    this.api.mathEvaluate(this.expression).subscribe(res => {
      this.result.set(res.result || res.error);
    });
  }
}
