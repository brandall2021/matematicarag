import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../core/services/auth.service';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-documents',
  standalone: true,
  imports: [CommonModule, MatButtonModule],
  template: `
    <div class="container">
      <h1>Gestion Documental</h1>
      <p>Subi material de estudio (PDF, DOCX, PPTX, TXT, Markdown, YouTube)</p>
      <div class="empty-state">
        <p>No hay documentos cargados aun.</p>
        <button mat-raised-button color="primary">Subir documento</button>
      </div>
    </div>
  `,
  styles: [`
    .container { padding: 2rem; max-width: 800px; margin: 0 auto; background: var(--bg); color: var(--text); min-height: 100vh; }
    h1 { color: var(--accent); }
    .empty-state { text-align: center; margin-top: 3rem; padding: 3rem; background: var(--surface); border-radius: 12px; }
  `]
})
export class DocumentsComponent {}
