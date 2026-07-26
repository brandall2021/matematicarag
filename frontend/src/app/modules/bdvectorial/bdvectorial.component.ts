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
  filename: string;
  page?: number;
  section?: string;
}

@Component({
  selector: 'app-bdvectorial',
  standalone: true,
  imports: [CommonModule, MatButtonModule, MatIconModule],
  template: `
    <div class="container">
      <h1><mat-icon>database</mat-icon> Base Vectorial (Qdrant)</h1>
      <p class="subtitle">Documentos indexados en la coleccion <code>matematica_chunks</code> — vectores de 1536 dimensiones (OpenAI text-embedding-3-small)</p>

      @if (loading()) {
        <div class="loading"><mat-icon class="spin">sync</mat-icon> Cargando documentos...</div>
      }

      @if (error()) {
        <div class="msg error">{{ error() }}</div>
      }

      <div class="doc-list">
        @for (doc of docs(); track doc.id) {
          <div class="doc-card" [class.expanded]="expandedDoc() === doc.id">
            <div class="doc-header" (click)="toggleDoc(doc)">
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
                      @case ('processing') { <mat-icon>sync</mat-icon> Procesando }
                      @case ('error') { <mat-icon>error</mat-icon> Error }
                      @default { {{ doc.status }} }
                    }
                  </span>
                  @if (doc.status === 'indexed') {
                    &middot; {{ doc.chunkCount }} chunks vectoriales
                  }
                </div>
              </div>
              <mat-icon class="expand-icon">{{ expandedDoc() === doc.id ? 'expand_less' : 'expand_more' }}</mat-icon>
            </div>

            @if (expandedDoc() === doc.id) {
              <div class="chunks-panel">
                @if (loadingChunks()) {
                  <div class="loading"><mat-icon class="spin">sync</mat-icon> Cargando chunks...</div>
                }
                <div class="chunks-stats">
                  {{ chunks().length }} chunks &middot;
                  {{ totalChars() }} caracteres &middot;
                  {{ totalWords() }} palabras
                </div>
                <div class="chunks-list">
                  @for (chunk of chunks(); track chunk.id) {
                    <div class="chunk-card">
                      <div class="chunk-header">
                        <span class="chunk-index">Chunk #{{ chunk.chunkIndex + 1 }}</span>
                        <div class="chunk-meta">
                          @if (chunk.page) {
                            <span class="badge page">Pag {{ chunk.page }}</span>
                          }
                          @if (chunk.section) {
                            <span class="badge section">{{ chunk.section }}</span>
                          }
                          <span class="badge chars">{{ chunk.content.length }} chars</span>
                          <span class="badge words">{{ countWords(chunk.content) }} words</span>
                        </div>
                      </div>
                      <div class="chunk-content">{{ chunk.content }}</div>
                    </div>
                  }
                  @if (chunks().length === 0 && !loadingChunks()) {
                    <div class="empty">No hay chunks para este documento.</div>
                  }
                </div>
              </div>
            }
          </div>
        }
        @if (docs().length === 0 && !loading()) {
          <div class="empty">No hay documentos indexados en la base vectorial.</div>
        }
      </div>
    </div>
  `,
  styles: [`
    .container { padding: 1.5rem; max-width: 900px; margin: 0 auto; background: var(--bg); color: var(--text); min-height: 100vh; }
    @media (min-width: 768px) { .container { padding: 2rem; } }
    h1 { color: var(--accent); display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.25rem; }
    h1 mat-icon { font-size: 1.8rem; width: 1.8rem; height: 1.8rem; }
    .subtitle { color: var(--text-secondary); margin-bottom: 1.5rem; font-size: 0.85rem; }
    .subtitle code { background: var(--surface); padding: 0.15rem 0.4rem; border-radius: 4px; font-size: 0.8rem; }
    .loading { display: flex; align-items: center; gap: 0.5rem; color: var(--text-secondary); padding: 2rem; justify-content: center; }
    .msg { padding: 0.75rem 1rem; border-radius: 8px; margin-bottom: 1rem; }
    .msg.error { background: #dc262620; color: #dc2626; border: 1px solid #dc262640; }
    .doc-list { display: flex; flex-direction: column; gap: 0.75rem; }
    .doc-card { background: var(--surface); border-radius: 10px; border: 1px solid var(--border); overflow: hidden; transition: border-color 0.2s; }
    .doc-card.expanded { border-color: var(--accent); }
    .doc-header { display: flex; align-items: center; gap: 0.75rem; padding: 1rem; cursor: pointer; }
    .doc-header:hover { background: var(--hover); }
    .doc-icon mat-icon { font-size: 2rem; width: 2rem; height: 2rem; }
    .doc-info { flex: 1; }
    .doc-name { font-weight: 600; }
    .doc-meta { font-size: 0.8rem; color: var(--text-secondary); display: flex; align-items: center; gap: 0.25rem; flex-wrap: wrap; }
    .status { display: inline-flex; align-items: center; gap: 0.2rem; }
    .status mat-icon { font-size: 0.9rem; width: 0.9rem; height: 0.9rem; }
    .status.indexed { color: #16a34a; }
    .status.processing { color: #f59e0b; }
    .status.error { color: #dc2626; }
    .expand-icon { color: var(--text-secondary); }
    .chunks-panel { border-top: 1px solid var(--border); padding: 1rem; }
    .chunks-stats { font-size: 0.8rem; color: var(--text-secondary); margin-bottom: 1rem; padding: 0.5rem; background: var(--bg); border-radius: 6px; text-align: center; }
    .chunks-list { display: flex; flex-direction: column; gap: 0.75rem; max-height: 600px; overflow-y: auto; }
    .chunk-card { background: var(--bg); border: 1px solid var(--border); border-radius: 8px; padding: 0.75rem; }
    .chunk-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; flex-wrap: wrap; gap: 0.5rem; }
    .chunk-index { font-weight: 600; font-size: 0.85rem; color: var(--accent); }
    .chunk-meta { display: flex; gap: 0.4rem; flex-wrap: wrap; }
    .badge { font-size: 0.7rem; padding: 0.15rem 0.5rem; border-radius: 999px; font-weight: 500; }
    .badge.page { background: #3b82f620; color: #3b82f6; }
    .badge.section { background: #8b5cf620; color: #8b5cf6; }
    .badge.chars { background: var(--border); color: var(--text-secondary); }
    .badge.words { background: var(--border); color: var(--text-secondary); }
    .chunk-content { font-size: 0.82rem; line-height: 1.5; color: var(--text); white-space: pre-wrap; word-break: break-word; max-height: 200px; overflow-y: auto; }
    .empty { text-align: center; color: var(--text-secondary); padding: 2rem; }
    .spin { animation: spin 1s linear infinite; }
    @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
  `]
})
export class BdvectorialComponent {
  docs = signal<Doc[]>([]);
  chunks = signal<Chunk[]>([]);
  expandedDoc = signal<string | null>(null);
  loading = signal(false);
  loadingChunks = signal(false);
  error = signal('');

  constructor(private http: HttpClient) {
    this.loadDocs();
  }

  loadDocs() {
    this.loading.set(true);
    this.error.set('');
    this.http.get<Doc[]>(`${environment.apiUrl}/api/documents`).subscribe({
      next: (d) => { this.docs.set(d.filter(doc => doc.status === 'indexed')); this.loading.set(false); },
      error: () => { this.loading.set(false); this.error.set('Error al cargar documentos de la base vectorial'); }
    });
  }

  toggleDoc(doc: Doc) {
    if (this.expandedDoc() === doc.id) {
      this.expandedDoc.set(null);
      this.chunks.set([]);
    } else {
      this.expandedDoc.set(doc.id);
      this.loadChunks(doc);
    }
  }

  loadChunks(doc: Doc) {
    this.loadingChunks.set(true);
    this.chunks.set([]);
    this.http.get<Chunk[]>(`${environment.apiUrl}/api/documents/${doc.id}/chunks`).subscribe({
      next: (c) => {
        const sorted = [...c].sort((a, b) => a.chunkIndex - b.chunkIndex);
        this.chunks.set(sorted);
        this.loadingChunks.set(false);
      },
      error: () => { this.chunks.set([]); this.loadingChunks.set(false); }
    });
  }

  totalChars(): number {
    return this.chunks().reduce((sum, c) => sum + c.content.length, 0);
  }

  totalWords(): number {
    return this.chunks().reduce((sum, c) => sum + this.countWords(c.content), 0);
  }

  countWords(text: string): number {
    return text.trim().split(/\s+/).filter(w => w.length > 0).length;
  }

  formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1048576).toFixed(1) + ' MB';
  }
}
