import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface TutorRequest {
  query: string;
  course_id?: string;
  unit_id?: string;
  explanation_level?: 'basic' | 'intermediate' | 'advanced';
  mode?: 'solve' | 'verify' | 'hint' | 'explain_error';
  user_result?: string;
  user_procedure?: string[];
}

export interface TutorStep {
  number: number;
  title: string;
  explanation: string;
  latex?: string;
  is_math: boolean;
}

export interface TutorResponse {
  problem: { type: string; expression: string; variable?: string };
  method: { name: string; description: string };
  steps: TutorStep[];
  result?: { success: boolean; result: string; latex: string };
  verification?: { status: string; method?: string };
  citations: any[];
  sources: any[];
  math_computed: boolean;
  confidence: string;
}

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

  tutorSolve(request: TutorRequest): Observable<TutorResponse> {
    return this.http.post<TutorResponse>(`${this.baseUrl}/tutor/solve`, request);
  }

  createTutorSession(mode: string, courseID?: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/sessions/session`, { mode, course_id: courseID || 'matematica-1' });
  }

  submitTutorAnswer(sessionID: string, exerciseID: string, answer: string, procedure?: string[]): Observable<any> {
    return this.http.post(`${this.baseUrl}/sessions/answer`, { session_id: sessionID, exercise_id: exerciseID, answer, procedure });
  }

  requestTutorHint(sessionID: string, exerciseID: string, hintIndex: number): Observable<any> {
    return this.http.post(`${this.baseUrl}/sessions/hint`, { session_id: sessionID, exercise_id: exerciseID, hint_index: hintIndex });
  }
}
