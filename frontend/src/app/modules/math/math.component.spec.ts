import { ComponentFixture, TestBed, fakeAsync, flush } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { of, throwError } from 'rxjs';
import { MathComponent } from './math.component';
import { ApiService } from '../../core/services/api.service';

describe('MathComponent', () => {
  let component: MathComponent;
  let fixture: ComponentFixture<MathComponent>;
  let apiSpy: jasmine.SpyObj<ApiService>;

  beforeEach(async () => {
    apiSpy = jasmine.createSpyObj('ApiService', ['mathEvaluate', 'mathPlot']);
    await TestBed.configureTestingModule({
      imports: [MathComponent],
      providers: [
        provideHttpClient(),
        { provide: ApiService, useValue: apiSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(MathComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('creates', () => {
    expect(component).toBeTruthy();
  });

  it('grafica la expresión y guarda los puntos', fakeAsync(() => {
    apiSpy.mathPlot.and.returnValue(of({ points: [[0, 0], [1, 1]], xmin: -10, xmax: 10 }));

    component.latexValue.set('x');
    component.plot();

    expect(apiSpy.mathPlot).toHaveBeenCalledWith('x', -10, 10);
    expect(component.plotPoints().length).toBe(2);
    flush();
  }));

  it('notifica un error cuando no se puede graficar', fakeAsync(() => {
    apiSpy.mathPlot.and.returnValue(of({ error: 'expresión inválida', points: [] }));

    component.latexValue.set('x');
    component.plot();

    expect(component.plotError()).toContain('expresión inválida');
    flush();
  }));

  it('maneja fallos de red al graficar', () => {
    apiSpy.mathPlot.and.returnValue(throwError(() => ({ error: { error: 'boom' } })));

    component.latexValue.set('x');
    component.plot();

    expect(component.plotError()).toContain('boom');
  });

  it('limpia el gráfico con clearPlot', () => {
    component.plotPoints.set([[0, 0]]);
    component.clearPlot();
    expect(component.plotPoints().length).toBe(0);
  });

  it('limpia el gráfico al cambiar la expresión', () => {
    component.plotPoints.set([[0, 0]]);
    component.onInput({ target: { value: 'x^2' } });
    expect(component.latexValue()).toBe('x^2');
    expect(component.plotPoints().length).toBe(0);
  });
});
