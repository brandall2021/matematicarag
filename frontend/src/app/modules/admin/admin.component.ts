import { Component } from '@angular/core';

@Component({
  selector: 'app-admin',
  standalone: true,
  template: `<div class="container"><h1>Administracion</h1><p>Panel de administracion del sistema.</p></div>`,
  styles: [`.container { padding: 1.5rem; max-width: 800px; margin: 0 auto; background: var(--bg); color: var(--text); } @media (min-width: 768px) { .container { padding: 2rem; } } h1 { color: var(--accent); }`]
})
export class AdminComponent {}
