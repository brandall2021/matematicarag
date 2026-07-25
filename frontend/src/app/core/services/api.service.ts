import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class ApiService {
  private baseUrl = environment.apiUrl + '/api';

  constructor(private http: HttpClient) {}

  chat(message: string, sessionId?: string, model?: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/chat/message`, { content: message, sessionId, model });
  }

  getSessions(): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/chat/sessions`);
  }

  getSessionMessages(sessionId: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/chat/sessions/${sessionId}/messages`);
  }

  ragQuery(query: string, topK?: number): Observable<any> {
    return this.http.post(`${this.baseUrl}/rag/query`, { query, topK });
  }

  mathEvaluate(expression: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/math/evaluate`, { expression });
  }

  mathOperation(operation: string, expression: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/math/${operation}`, { expression });
  }

  mathPlot(expression: string, xMin: number, xMax: number): Observable<any> {
    return this.http.post(`${this.baseUrl}/math/plot`, { expression, xMin, xMax });
  }

  getDocuments(): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/documents`);
  }

  uploadDocument(file: File): Observable<any> {
    const fd = new FormData();
    fd.append('file', file);
    return this.http.post(`${this.baseUrl}/documents/upload`, fd);
  }

  getDocumentChunks(docId: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/documents/${docId}/chunks`);
  }

  getHistory(): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/history`);
  }

  getAdminStats(): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/stats/admin`);
  }
}
