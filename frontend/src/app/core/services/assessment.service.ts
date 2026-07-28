import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface Assessment {
  id: string;
  title: string;
  description: string;
  course_id: string;
  assessment_type: 'diagnostic' | 'formative' | 'summative' | 'recovery' | 'practice';
  mode: 'fixed' | 'generated' | 'adaptive';
  time_limit_minutes: number;
  max_attempts: number;
  shuffle_questions: boolean;
  show_results: boolean;
  show_solutions: boolean;
  passing_score: number;
  total_points: number;
  created_by: string;
  status: 'draft' | 'published' | 'archived';
  published_at?: string;
  expires_at?: string;
  metadata?: any;
}

export interface AssessmentQuestion {
  id: string;
  assessment_id: string;
  exercise_id?: string;
  question_order: number;
  points: number;
  question_type: 'exercise' | 'generated' | 'text' | 'multiple_choice';
  statement_override: string;
  statement?: string;
  latex?: string;
  difficulty?: number;
  concept_id?: string;
}

export interface StudentAssessment {
  id: string;
  student_id: string;
  assessment_id: string;
  status: 'in_progress' | 'submitted' | 'graded' | 'returned';
  started_at: string;
  submitted_at?: string;
  time_spent_seconds: number;
  attempt_number: number;
  total_score: number;
  max_score: number;
  percentage: number;
  passed: boolean;
  feedback: string;
}

export interface StudentAnswer {
  id: string;
  student_assessment_id: string;
  question_id: string;
  answer: string;
  procedure?: string[];
  is_correct: boolean;
  score: number;
  points_earned: number;
  points_possible: number;
  math_verified: boolean;
  rubric_scores?: any;
  feedback: string;
  time_spent_seconds: number;
  question_order?: number;
}

export interface Rubric {
  id: string;
  assessment_id: string;
  name: string;
  description: string;
  rubric_type: 'analytic' | 'holistic';
  max_score: number;
  criteria: any[];
}

export interface StudentAnalytics {
  id: string;
  student_id: string;
  course_id: string;
  total_assessments: number;
  passed_assessments: number;
  average_score: number;
  average_time_seconds: number;
  competency_level: 'beginner' | 'developing' | 'proficient' | 'advanced' | 'exceptional';
  competency_score: number;
  weakest_concepts: string[];
  strongest_concepts: string[];
  improvement_trend: number;
  study_streak_days: number;
}

export interface RecoveryPlan {
  id: string;
  student_id: string;
  course_id: string;
  trigger_assessment_id?: string;
  trigger_score: number;
  status: 'active' | 'completed' | 'expired' | 'cancelled';
  priority: number;
  concepts_to_review: string[];
  recommended_activities: any[];
  target_date?: string;
  completed_at?: string;
}

export interface AcademicAlert {
  id: string;
  student_id: string;
  alert_type: string;
  severity: 'info' | 'warning' | 'critical';
  title: string;
  message: string;
  concept_id: string;
  assessment_id?: string;
  acknowledged: boolean;
  acknowledged_by?: string;
  acknowledged_at?: string;
  created_at?: string;
  metadata?: any;
}

@Injectable({ providedIn: 'root' })
export class AssessmentService {
  private baseUrl = environment.apiUrl + '/api';

  constructor(private http: HttpClient) {}

  createAssessment(assessment: Partial<Assessment> & { question_ids?: string[] }): Observable<Assessment> {
    return this.http.post<Assessment>(`${this.baseUrl}/assessments/`, assessment);
  }

  listAssessments(courseId?: string, type?: string, status?: string): Observable<Assessment[]> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    if (type) params = params.set('type', type);
    if (status) params = params.set('status', status);
    return this.http.get<Assessment[]>(`${this.baseUrl}/assessments/`, { params });
  }

  getAssessment(assessmentId: string): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/assessments/${assessmentId}`);
  }

  updateAssessment(assessmentId: string, updates: Partial<Assessment>): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/assessments/${assessmentId}`, updates);
  }

  deleteAssessment(assessmentId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/assessments/${assessmentId}`);
  }

  publishAssessment(assessmentId: string): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/assessments/${assessmentId}/publish`, {});
  }

  startAssessment(assessmentId: string): Observable<{ student_assessment: StudentAssessment; questions: AssessmentQuestion[] }> {
    return this.http.post<{ student_assessment: StudentAssessment; questions: AssessmentQuestion[] }>(
      `${this.baseUrl}/assessments/${assessmentId}/start`, {}
    );
  }

  submitAssessment(assessmentId: string, answers: { question_id: string; answer: string; procedure?: string[] }[]): Observable<any> {
    return this.http.post<any>(`${this.baseUrl}/assessments/${assessmentId}/submit`, { answers });
  }

  getAssessmentResult(assessmentId: string): Observable<{ student_assessment: StudentAssessment; answers: StudentAnswer[] }> {
    return this.http.get<{ student_assessment: StudentAssessment; answers: StudentAnswer[] }>(
      `${this.baseUrl}/assessments/${assessmentId}/results`
    );
  }

  getAssessmentStudentResults(assessmentId: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/assessments/${assessmentId}/student-results`);
  }

  manualGradeAnswer(answerId: string, score: number, feedback: string): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/grading/answer/${answerId}`, { score, feedback });
  }

  createRubric(assessmentId: string, rubric: Partial<Rubric>): Observable<Rubric> {
    return this.http.post<Rubric>(`${this.baseUrl}/grading/rubric/${assessmentId}`, rubric);
  }

  getRubrics(assessmentId: string): Observable<Rubric[]> {
    return this.http.get<Rubric[]>(`${this.baseUrl}/grading/rubric/${assessmentId}`);
  }

  evaluateWithRubric(answerId: string, rubricId: string): Observable<any> {
    return this.http.post<any>(`${this.baseUrl}/grading/evaluate/${answerId}`, { rubric_id: rubricId });
  }

  batchAutoGrade(assessmentId: string): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/grading/batch-grade/${assessmentId}`, {});
  }

  getStudentAnalytics(studentId: string, courseId?: string): Observable<StudentAnalytics> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<StudentAnalytics>(`${this.baseUrl}/analytics/v2/student/${studentId}`, { params });
  }

  getCourseAnalytics(courseId: string): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/analytics/v2/course/${courseId}`);
  }

  getCompetencyReport(studentId: string, courseId?: string): Observable<any> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<any>(`${this.baseUrl}/analytics/v2/student/${studentId}/competency`, { params });
  }

  getPerformanceTrend(studentId: string, courseId?: string): Observable<any> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<any>(`${this.baseUrl}/analytics/v2/student/${studentId}/trend`, { params });
  }

  updateStudentAnalytics(studentId: string, courseId?: string): Observable<void> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.post<void>(`${this.baseUrl}/analytics/v2/student/${studentId}/update`, {}, { params });
  }

  createRecoveryPlan(assessmentId: string, score: number, courseId?: string): Observable<RecoveryPlan> {
    return this.http.post<RecoveryPlan>(`${this.baseUrl}/recovery/`, {
      assessment_id: assessmentId,
      score,
      course_id: courseId || 'matematica-1'
    });
  }

  getRecoveryPlans(courseId?: string): Observable<RecoveryPlan[]> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.get<RecoveryPlan[]>(`${this.baseUrl}/recovery/`, { params });
  }

  completeRecoveryPlan(planId: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/recovery/${planId}/complete`, {});
  }

  cancelRecoveryPlan(planId: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/recovery/${planId}/cancel`, {});
  }

  getAlerts(severity?: string): Observable<AcademicAlert[]> {
    let params = new HttpParams();
    if (severity) params = params.set('severity', severity);
    return this.http.get<AcademicAlert[]>(`${this.baseUrl}/alerts/`, { params });
  }

  acknowledgeAlert(alertId: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/alerts/${alertId}/acknowledge`, {});
  }

  checkForAlerts(courseId?: string): Observable<{ alerts_created: number; alerts: AcademicAlert[] }> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    return this.http.post<{ alerts_created: number; alerts: AcademicAlert[] }>(
      `${this.baseUrl}/alerts/check`, {}, { params }
    );
  }

  getAllAlerts(courseId?: string, severity?: string): Observable<any[]> {
    let params = new HttpParams();
    if (courseId) params = params.set('course_id', courseId);
    if (severity) params = params.set('severity', severity);
    return this.http.get<any[]>(`${this.baseUrl}/alerts/all`, { params });
  }
}
