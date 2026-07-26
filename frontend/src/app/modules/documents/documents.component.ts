import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { environment } from '../../../environments/environment';

interface Doc {
  id: string;
  filename: string;
  originalName: string;
  type: string;
  size: number;
  status: string;
  chunkCount: number;
  createdAt: string;
}

interface Chunk {
  id: string;
  documentId: string;
  chunkIndex: number;
  content: string;
  metadata: any;
  createdAt: string;
}

@Component({
  selector: 'app-documents',
  standalone: true,
  imports: [CommonModule, MatButtonModule, MatIconModule],
  template: `
    <div class="container">
      <h1>Gestion Documental</h1>
      <p class="subtitle">Subi material de estudio (PDF, DOCX, TXT, Markdown) para indexarlo en la base vectorial</p>

      <div class="upload-area"
           (dragover)="$event.preventDefault(); dragActive = true"
           (dragleave)="dragActive = false"
           (drop)="onDrop($event)"
           [class.drag-active]="dragActive">
        <input #fileInput type="file" accept=".pdf,.docx,.txt,.md" (change)="onFileSelect($event)" hidden>
        <mat-icon class="upload-icon">cloud_upload</mat-icon>
        <p>Arrastra un archivo aqui o</p>
        <button mat-raised-button color="primary" (click)="fileInput.click()">
          <mat-icon>add</mat-icon> Seleccionar archivo
        </button>
      </div>

      @if (uploading()) {
        <div class="upload-progress">
          <mat-icon class="spin">sync</mat-icon> Subiendo y procesando...
        </div>
      }

      @if (message()) {
        <div class="msg" [class.error]="msgError()">{{ message() }}</div>
      }

      <div class="doc-list">
        @for (doc of docs(); track doc.id) {
          <div class="doc-card">
            <div class="doc-icon">
              @switch (doc.type) {
                @case ('.pdf') { <mat-icon color="warn">picture_as_pdf</mat-icon> }
                @case ('.docx') { <mat-icon color="primary">description</mat-icon> }
                @default { <mat-icon>article</mat-icon> }
              }
            </div>
            <div class="doc-info">
              <div class="doc-name">{{ doc.originalName }}</div>
              <div class="doc-meta">
                {{ formatSize(doc.size) }} &middot;
                <span class="status" [class]="doc.status">
                  @switch (doc.status) {
                    @case ('indexed') { <mat-icon>check_circle</mat-icon> Indexado }
                    @case ('processing') { <mat-icon>sync</mat-icon> Procesando... }
                    @case ('error') { <mat-icon>error</mat-icon> Error }
                    @default { {{ doc.status }} }
                  }
                </span>
                @if (doc.status === 'indexed') {
                  &middot; {{ doc.chunkCount }} chunks
                }
              </div>
            </div>
            <div class="doc-actions">
              @if (doc.status === 'indexed' && doc.chunkCount > 0) {
                <button mat-stroked-button (click)="viewChunks(doc)">
                  <mat-icon>visibility</mat-icon> Ver chunks
                </button>
              }
              <button mat-icon-button color="warn" (click)="deleteDoc(doc)">
                <mat-icon>delete</mat-icon>
              </button>
            </div>
          </div>
        }
        @if (docs().length === 0 && !loading()) {
          <div class="empty">No hay documentos cargados aun.</div>
        }
      </div>

      @if (selectedDoc()) {
        <div class="chunks-panel">
          <div class="chunks-header">
            <h3>Chunks vectoriales: {{ selectedDoc()!.originalName }}</h3>
            <button mat-icon-button (click)="selectedDoc.set(null)"><mat-icon>close</mat-icon></button>
          </div>
          <div class="chunks-list">
            @for (chunk of chunks(); track chunk.id; let i = $index) {
              <div class="chunk-card">
                <div class="chunk-index">#{{ i + 1 }}</div>
                <div class="chunk-content">{{ chunk.content }}</div>
              </div>
            }
            @if (chunks().length === 0) {
              <div class="empty">Cargando chunks...</div>
            }
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .container { padding: 1.5rem; max-width: 900px; margin: 0 auto; background: var(--bg); color: var(--text); }
    @media (min-width: 768px) { .container { padding: 2rem; } }
    h1 { color: var(--accent); margin-bottom: 0.25rem; }
    .subtitle { color: var(--text-secondary); margin-bottom: 1.5rem; font-size: 0.9rem; }
    .upload-area { border: 2px dashed var(--border); border-radius: 12px; padding: 1.5rem; text-align: center; background: var(--surface); transition: border-color 0.2s; cursor: pointer; }
    @media (min-width: 768px) { .upload-area { padding: 2.5rem; } }
    .upload-area.drag-active { border-color: var(--accent); background: var(--bg); }
    .upload-icon { font-size: 48px; width: 48px; height: 48px; color: var(--text-secondary); margin-bottom: 0.5rem; }
    .upload-progress { display: flex; align-items: center; gap: 0.5rem; padding: 1rem; background: var(--surface); border-radius: 8px; margin-top: 1rem; color: var(--accent); }
    .spin { animation: spin 1s linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
    .msg { margin-top: 1rem; padding: 0.75rem; border-radius: 8px; background: #1b5e20; color: white; text-align: center; font-size: 0.9rem; }
    .msg.error { background: #b71c1c; }
    .doc-list { margin-top: 1.5rem; display: flex; flex-direction: column; gap: 0.5rem; }
    .doc-card { display: flex; align-items: center; gap: 1rem; padding: 1rem; background: var(--surface); border-radius: 8px; flex-wrap: wrap; }
    .doc-icon mat-icon { font-size: 32px; width: 32px; height: 32px; }
    .doc-info { flex: 1; }
    .doc-name { font-weight: 600; font-size: 0.95rem; }
    .doc-meta { color: var(--text-secondary); font-size: 0.8rem; display: flex; align-items: center; gap: 0.3rem; }
    .status { display: inline-flex; align-items: center; gap: 0.2rem; }
    .status mat-icon { font-size: 14px; width: 14px; height: 14px; }
    .status.indexed { color: #4caf50; }
    .status.processing { color: var(--accent); }
    .status.error { color: #f44336; }
    .doc-actions { display: flex; gap: 0.5rem; align-items: center; }
    .empty { text-align: center; padding: 2rem; color: var(--text-secondary); background: var(--surface); border-radius: 8px; }
    .chunks-panel { margin-top: 1.5rem; background: var(--surface); border-radius: 12px; padding: 1.5rem; }
    .chunks-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
    .chunks-header h3 { margin: 0; color: var(--accent); font-size: 1rem; }
    .chunks-list { display: flex; flex-direction: column; gap: 0.75rem; max-height: 500px; overflow-y: auto; }
    .chunk-card { background: var(--bg); padding: 1rem; border-radius: 8px; border-left: 3px solid var(--accent); }
    .chunk-index { font-size: 0.75rem; color: var(--accent); font-weight: 600; margin-bottom: 0.3rem; }
    .chunk-content { font-size: 0.85rem; line-height: 1.5; white-space: pre-wrap; word-break: break-word; }
  `]
})
export class DocumentsComponent {
  docs = signal<Doc[]>([]);
  chunks = signal<Chunk[]>([]);
  selectedDoc = signal<Doc | null>(null);
  loading = signal(false);
  uploading = signal(false);
  message = signal('');
  msgError = signal(false);
  dragActive = false;

  constructor(private http: HttpClient) {
    this.loadDocs();
  }

  loadDocs() {
    this.loading.set(true);
    this.http.get<Doc[]>(`${environment.apiUrl}/api/documents`).subscribe({
      next: (d) => { this.docs.set(d); this.loading.set(false); },
      error: () => { this.loading.set(false); this.showMsg('Error al cargar documentos', true); }
    });
  }

  onFileSelect(event: any) {
    const file = event.target.files[0];
    if (file) this.upload(file);
  }

  onDrop(event: DragEvent) {
    event.preventDefault();
    this.dragActive = false;
    const file = event.dataTransfer?.files[0];
    if (file) this.upload(file);
  }

  upload(file: File) {
    this.uploading.set(true);
    this.message.set('');
    const fd = new FormData();
    fd.append('file', file);
    this.http.post(`${environment.apiUrl}/api/documents/upload`, fd).subscribe({
      next: () => {
        this.uploading.set(false);
        this.showMsg('Documento subido y procesado correctamente');
        setTimeout(() => this.loadDocs(), 1000);
      },
      error: (err) => {
        this.uploading.set(false);
        this.showMsg(err.error?.error || 'Error al subir', true);
      }
    });
  }

  viewChunks(doc: Doc) {
    this.selectedDoc.set(doc);
    this.http.get<Chunk[]>(`${environment.apiUrl}/api/documents/${doc.id}/chunks`).subscribe({
      next: (c) => this.chunks.set(c),
      error: () => this.chunks.set([])
    });
  }

  deleteDoc(doc: Doc) {
    if (!confirm(`Eliminar "${doc.originalName}"?`)) return;
    this.http.delete(`${environment.apiUrl}/api/documents/${doc.id}`).subscribe({
      next: () => { this.loadDocs(); this.selectedDoc.set(null); this.showMsg('Documento eliminado'); },
      error: () => this.showMsg('Error al eliminar', true)
    });
  }

  formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1048576).toFixed(1) + ' MB';
  }

  private showMsg(msg: string, error = false) {
    this.message.set(msg);
    this.msgError.set(error);
    setTimeout(() => this.message.set(''), 3000);
  }
}
