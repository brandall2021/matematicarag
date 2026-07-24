import { Component } from '@angular/core';

@Component({
  selector: 'app-admin',
  standalone: true,
  template: `<div class="container"><h1>Administracion</h1><p>Panel de administracion del sistema.</p></div>`,
  styles: [`.container { padding: 2rem; max-width: 800px; margin: 0 auto; background: #1a1a2e; color: white; min-height: 100vh; } h1 { color: #e2b714; }`]
})
export class AdminComponent {}
