import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-math',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule],
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
      <div class="input-area">
        <input [(ngModel)]="expression" placeholder="Ingresa una expresion matematica..." class="math-input">
        <button mat-raised-button color="primary" (click)="calculate()" [disabled]="!expression">Calcular</button>
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
    .math-container { padding: 2rem; max-width: 800px; margin: 0 auto; background: #1a1a2e; color: white; min-height: 100vh; }
    h1 { color: #e2b714; font-family: 'Newsreader', serif; }
    .operations { display: flex; gap: 0.5rem; margin-bottom: 1rem; flex-wrap: wrap; }
    .operations button.active { background: #e2b714; color: #1a1a2e; }
    .input-area { display: flex; gap: 0.5rem; margin-bottom: 2rem; }
    .math-input { flex: 1; padding: 0.75rem; border-radius: 8px; border: 1px solid #2a2a4a; background: #16213e; color: white; font-size: 1.1rem; outline: none; }
    .math-input:focus { border-color: #e2b714; }
    .result { background: #16213e; padding: 1.5rem; border-radius: 12px; }
    pre { white-space: pre-wrap; font-size: 1.1rem; }
  `]
})
export class MathComponent {
  expression = '';
  op = signal('evaluate');
  result = signal('');

  constructor(private api: ApiService) {}

  calculate() {
    this.api.mathEvaluate(this.expression).subscribe(res => {
      this.result.set(res.result || res.error);
    });
  }
}
