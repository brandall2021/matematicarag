import { Component, signal, ViewChild, ElementRef, AfterViewInit, OnDestroy, NgZone, HostListener } from '@angular/core';
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
        <button class="calc-btn plot-btn" (click)="plot()" [disabled]="!latexValue() || plotLoading()">
          @if (plotLoading()) {
            <span class="calc-spinner plot-spinner"></span>
            <span>Graficando</span>
          } @else {
            <mat-icon>show_chart</mat-icon>
            <span>Graficar</span>
          }
        </button>
      </div>

      <div class="plot-toolbar">
        <label class="range-field">
          <span class="range-label">X desde</span>
          <input type="number" step="any" [value]="xMin()" (input)="setXMin($event)" />
        </label>
        <label class="range-field">
          <span class="range-label">X hasta</span>
          <input type="number" step="any" [value]="xMax()" (input)="setXMax($event)" />
        </label>
        @if (plotPoints().length) {
          <button class="plot-clear" (click)="clearPlot()">Quitar gráfico</button>
        }
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

      @if (plotError()) {
        <div class="plot-error">
          <mat-icon class="plot-error-icon">error_outline</mat-icon>
          <span>{{ plotError() }}</span>
        </div>
      }

      @if (plotPoints().length) {
        <div class="plot-card">
          <div class="plot-card-header">
            <mat-icon class="plot-icon">show_chart</mat-icon>
            <h3>Gráfica de {{ latexValue() }}</h3>
          </div>
          <div class="plot-canvas-wrap">
            <canvas #plotCanvas></canvas>
          </div>
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

    .plot-btn {
      background: transparent;
      color: var(--accent);
      border: 2px solid var(--accent);
    }
    .plot-btn:hover:not(:disabled) {
      background: var(--accent-muted);
      color: var(--accent-hover);
      border-color: var(--accent-hover);
    }
    .plot-btn .plot-spinner {
      border-color: var(--accent-muted);
      border-top-color: var(--accent);
    }

    .plot-toolbar {
      display: flex;
      align-items: center;
      gap: var(--space-md);
      margin: calc(-1 * var(--space-sm)) 0 var(--space-md);
      flex-wrap: wrap;
    }
    .range-field {
      display: inline-flex;
      align-items: center;
      gap: var(--space-sm);
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-md);
      padding: 0.35rem var(--space-sm);
    }
    .range-label {
      color: var(--text-secondary);
      font-size: 0.72rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }
    .range-field input {
      width: 6.5rem;
      background: transparent;
      border: none;
      outline: none;
      color: var(--text);
      font-family: var(--font-mono, monospace);
      font-size: 0.85rem;
      padding: 0.15rem 0;
    }
    .plot-clear {
      margin-left: auto;
      background: transparent;
      border: none;
      color: var(--text-tertiary);
      font-size: 0.78rem;
      font-weight: 600;
      cursor: pointer;
      text-decoration: underline;
      padding: 0.25rem 0.5rem;
    }
    .plot-clear:hover { color: var(--danger); }

    .plot-error {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      background: var(--danger-muted);
      color: var(--danger);
      border: 1px solid var(--danger);
      border-radius: var(--radius-md);
      padding: var(--space-sm) var(--space-md);
      font-size: 0.88rem;
      margin-bottom: var(--space-md);
    }
    .plot-error-icon { font-size: 18px; width: 18px; height: 18px; }

    .plot-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      box-shadow: var(--shadow-sm);
      animation: fadeUp 0.25s ease;
      overflow: hidden;
    }
    .plot-card-header {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: var(--space-md) var(--space-lg);
      border-bottom: 1px solid var(--border-light);
    }
    .plot-icon { color: var(--accent); font-size: 20px; width: 20px; height: 20px; }
    .plot-card-header h3 {
      font-family: var(--font-serif);
      font-size: 1rem;
      font-weight: 600;
      color: var(--text);
      margin: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .plot-canvas-wrap {
      padding: var(--space-md);
    }
    .plot-canvas-wrap canvas {
      display: block;
      width: 100%;
      height: 320px;
      border-radius: var(--radius-md);
    }

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
  @ViewChild('plotCanvas') plotCanvasRef!: ElementRef<HTMLCanvasElement>;

  result = signal('');
  latexValue = signal('');
  loading = signal(false);
  plotPoints = signal<Array<[number, number | null]>>([]);
  plotError = signal('');
  plotLoading = signal(false);
  xMin = signal(-10);
  xMax = signal(10);

  private mf: any = null;

  constructor(private api: ApiService, private zone: NgZone, private elRef: ElementRef) {}

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

  @HostListener('window:resize')
  onResize() {
    if (this.plotPoints().length) {
      setTimeout(() => this.drawPlot(), 0);
    }
  }

  private handleInput = () => {
    this.zone.run(() => {
      if (this.mf) {
        this.applyLatex(this.mf.value || '');
      }
    });
  };

  onInput(event: any) {
    this.applyLatex(event.target?.value || this.mf?.value || '');
  }

  private applyLatex(val: string) {
    if (this.latexValue() !== val) {
      this.latexValue.set(val);
      this.clearPlot();
    }
  }

  setXMin(event: any) {
    const v = parseFloat(event?.target?.value);
    if (!isNaN(v)) this.xMin.set(v);
  }

  setXMax(event: any) {
    const v = parseFloat(event?.target?.value);
    if (!isNaN(v)) this.xMax.set(v);
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

  plot() {
    const expr = this.latexValue();
    if (!expr) return;

    this.plotLoading.set(true);
    this.plotError.set('');

    this.api.mathPlot(expr, this.xMin(), this.xMax()).subscribe({
      next: (res: any) => {
        this.plotLoading.set(false);
        if (res && Array.isArray(res.points) && res.points.length) {
          this.plotPoints.set(res.points);
          setTimeout(() => this.drawPlot(), 0);
        } else {
          this.plotError.set(res?.error || 'No se pudo graficar la expresión');
        }
      },
      error: (err: any) => {
        this.plotLoading.set(false);
        this.plotError.set(err.error?.error || 'Error al graficar');
      }
    });
  }

  clearPlot() {
    this.plotPoints.set([]);
    this.plotError.set('');
  }

  private niceStep(range: number, targetTicks = 8): number {
    if (!isFinite(range) || range <= 0) return 1;
    const rough = range / targetTicks;
    const mag = Math.pow(10, Math.floor(Math.log10(rough)));
    const norm = rough / mag;
    const step = norm >= 5 ? 10 : norm >= 2 ? 5 : norm >= 1 ? 2 : 1;
    return step * mag;
  }

  private formatTick(v: number): string {
    const a = Math.abs(v);
    if (a >= 1e6 || (a > 0 && a < 1e-3)) return v.toExponential(1);
    return String(Math.round(v * 100) / 100);
  }

  private cssVar(name: string, fallback: string): string {
    try {
      const val = getComputedStyle(this.elRef.nativeElement).getPropertyValue(name).trim();
      return val || fallback;
    } catch {
      return fallback;
    }
  }

  private drawPlot() {
    const canvas = this.plotCanvasRef?.nativeElement;
    if (!canvas) return;
    const points = this.plotPoints();
    if (!points.length) return;

    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();
    const W = Math.max(rect.width, 40);
    const H = Math.max(rect.height, 40);
    canvas.width = Math.round(W * dpr);
    canvas.height = Math.round(H * dpr);
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    const pad = { l: 56, r: 14, t: 14, b: 30 };
    const pw = W - pad.l - pad.r;
    const ph = H - pad.t - pad.b;
    const xmin = this.xMin();
    const xmax = this.xMax();

    const finite = points.filter(p => p[1] !== null && isFinite(p[1] as number));
    let ymin: number;
    let ymax: number;
    if (finite.length >= 20) {
      const ys = finite.map(p => p[1] as number).sort((a, b) => a - b);
      const t = Math.floor(ys.length * 0.025);
      ymin = ys[t];
      ymax = ys[ys.length - 1 - t];
    } else if (finite.length) {
      const ys = finite.map(p => p[1] as number);
      ymin = Math.min(...ys);
      ymax = Math.max(...ys);
    } else {
      ymin = -10;
      ymax = 10;
    }
    if (!isFinite(ymin) || !isFinite(ymax)) { ymin = -10; ymax = 10; }
    if (ymax - ymin < 1e-9) { ymin -= 1; ymax += 1; }
    const ypad = (ymax - ymin) * 0.1 || 1;
    ymin -= ypad;
    ymax += ypad;

    const sx = (x: number) => pad.l + ((x - xmin) / (xmax - xmin)) * pw;
    const sy = (y: number) => pad.t + (1 - (y - ymin) / (ymax - ymin)) * ph;

    const bg = this.cssVar('--surface', '#ffffff');
    const grid = this.cssVar('--border-light', '#e4e6ef');
    const axis = this.cssVar('--text-secondary', '#5a5a72');
    const text = this.cssVar('--text', '#1a1a2e');
    const curve = this.cssVar('--accent', '#b8940a');

    ctx.fillStyle = bg;
    ctx.fillRect(0, 0, W, H);
    ctx.font = '11px "JetBrains Mono", monospace';

    const xStep = this.niceStep(xmax - xmin);
    for (let x = Math.ceil(xmin / xStep) * xStep; x <= xmax; x += xStep) {
      const px = sx(x);
      ctx.strokeStyle = grid;
      ctx.beginPath();
      ctx.moveTo(px, pad.t);
      ctx.lineTo(px, pad.t + ph);
      ctx.stroke();
      ctx.fillStyle = text;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'top';
      ctx.fillText(this.formatTick(x), px, pad.t + ph + 6);
    }

    const yStep = this.niceStep(ymax - ymin);
    for (let y = Math.ceil(ymin / yStep) * yStep; y <= ymax; y += yStep) {
      const py = sy(y);
      ctx.strokeStyle = grid;
      ctx.beginPath();
      ctx.moveTo(pad.l, py);
      ctx.lineTo(pad.l + pw, py);
      ctx.stroke();
      ctx.fillStyle = text;
      ctx.textAlign = 'right';
      ctx.textBaseline = 'middle';
      ctx.fillText(this.formatTick(y), pad.l - 6, py);
    }

    ctx.strokeStyle = axis;
    ctx.lineWidth = 1.5;
    if (xmin <= 0 && xmax >= 0) {
      const x0 = sx(0);
      ctx.beginPath();
      ctx.moveTo(x0, pad.t);
      ctx.lineTo(x0, pad.t + ph);
      ctx.stroke();
    }
    if (ymin <= 0 && ymax >= 0) {
      const y0 = sy(0);
      ctx.beginPath();
      ctx.moveTo(pad.l, y0);
      ctx.lineTo(pad.l + pw, y0);
      ctx.stroke();
    }

    ctx.strokeStyle = curve;
    ctx.lineWidth = 2;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';
    ctx.beginPath();
    let drawing = false;
    for (let i = 0; i < points.length; i++) {
      const p = points[i];
      const pyv = p[1];
      if (pyv === null || !isFinite(pyv)) {
        drawing = false;
        continue;
      }
      const px = sx(p[0]);
      const py = sy(pyv);
      if (drawing) {
        const prev = points[i - 1];
        if (prev && prev[1] !== null && Math.abs(sy(prev[1]) - py) > ph * 0.9) {
          ctx.stroke();
          ctx.beginPath();
        }
        ctx.lineTo(px, py);
      } else {
        ctx.moveTo(px, py);
        drawing = true;
      }
    }
    ctx.stroke();
  }
}
