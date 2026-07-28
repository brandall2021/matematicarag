import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface StudentProfile {
  id: string;
  student_id: string;
  course_id: string;
  overall_level: number;
  total_attempts: number;
  correct_attempts: number;
  total_hints_used: number;
  study_time_seconds: number;
}

export interface ConceptMastery {
  id: string;
  student_id: string;
  concept_id: string;
  mastery: number;
  status: 'not_started' | 'learning' | 'developing' | 'mastered';
  attempts: number;
  correct: number;
  hints_used: number;
  error_count: number;
}

export interface ConceptNode {
  id: string;
  name: string;
  description: string;
  parent_id?: string;
  course_id: string;
  difficulty_base: number;
  children?: ConceptNode[];
}

export interface Exercise {
  id: string;
  concept_id: string;
  difficulty: number;
  statement: string;
  latex: string;
  expected_answer: string;
  solution: string;
  solution_steps: any[];
  hints: string[];
  common_errors: any[];
  source: string;
  verified_by_math: boolean;
}

export interface SessionResponse {
  session_id: string;
  mode: string;
  exercise?: Exercise;
  message?: string;
}

export interface AnswerResponse {
  correct: boolean;
  score: number;
  feedback: string;
  first_error_step?: number;
  error_type?: string;
  mastery_before: number;
  mastery_after: number;
  mastery_status: string;
  next_exercise?: Exercise;
  math_verified: boolean;
  step_analysis?: any[];
}

export interface HintResponse {
  hint_index: number;
  hint: string;
  total_hints: number;
}

export interface AdaptiveRecommendation {
  recommended_concept: string;
  concept_name: string;
  recommended_difficulty: number;
  reason: string;
  prerequisites_met: boolean;
  missing_prereqs?: string[];
}

export interface StudentDashboard {
  profile: StudentProfile;
  mastery_map: Record<string, ConceptMastery>;
  recent_errors: any[];
  recommendations: string[];
  sessions_summary: {
    total_sessions: number;
    total_exercises: number;
    correct_rate: number;
    average_hints: number;
    study_time_hours: number;
  };
}

export interface TeacherCourseProgress {
  course_id: string;
  total_students: number;
  average_mastery: number;
}

export interface TopicMastery {
  concept_id: string;
  concept_name: string;
  average_mastery: number;
  student_count: number;
  struggling_count: number;
}

export interface CommonError {
  error_type: string;
  error_subtype: string;
  count: number;
  affected_students: number;
}

export interface StudentProgress {
  student_id: string;
  student_name: string;
  email: string;
  overall_level: number;
  total_attempts: number;
  last_active: string;
}

@Injectable({ providedIn: 'root' })
export class LearningService {
  private baseUrl = environment.apiUrl + '/api';

  constructor(private http: HttpClient) {}

  getMyProgress(): Observable<StudentDashboard> {
    return this.http.get<StudentDashboard>(`${this.baseUrl}/student/my-progress`);
  }

  getRecommendation(): Observable<AdaptiveRecommendation> {
    return this.http.get<AdaptiveRecommendation>(`${this.baseUrl}/student/recommendations`);
  }

  getConceptTree(courseID: string): Observable<ConceptNode[]> {
    return this.http.get<ConceptNode[]>(`${this.baseUrl}/learning/courses/${courseID}/concepts`);
  }

  getMastery(): Observable<Record<string, ConceptMastery>> {
    return this.http.get<Record<string, ConceptMastery>>(`${this.baseUrl}/learning/mastery`);
  }

  getErrors(): Observable<any[]> {
    return this.http.get<any[]>(`${this.baseUrl}/learning/errors`);
  }

  createSession(mode: string, courseID?: string): Observable<SessionResponse> {
    return this.http.post<SessionResponse>(`${this.baseUrl}/sessions/session`, { mode, course_id: courseID || 'matematica-1' });
  }

  submitAnswer(sessionID: string, exerciseID: string, answer: string, procedure?: string[]): Observable<AnswerResponse> {
    return this.http.post<AnswerResponse>(`${this.baseUrl}/sessions/answer`, { session_id: sessionID, exercise_id: exerciseID, answer, procedure });
  }

  requestHint(sessionID: string, exerciseID: string, hintIndex: number): Observable<HintResponse> {
    return this.http.post<HintResponse>(`${this.baseUrl}/sessions/hint`, { session_id: sessionID, exercise_id: exerciseID, hint_index: hintIndex });
  }

  generateExercise(conceptID: string, difficulty: number): Observable<Exercise> {
    return this.http.post<Exercise>(`${this.baseUrl}/exercises/generate`, { concept_id: conceptID, difficulty });
  }

  getExercise(conceptID: string, difficulty?: number): Observable<Exercise> {
    const params = difficulty ? `?difficulty=${difficulty}` : '';
    return this.http.get<Exercise>(`${this.baseUrl}/exercises/concept/${conceptID}${params}`);
  }

  getTeacherCourseProgress(courseID?: string): Observable<TeacherCourseProgress> {
    const params = courseID ? `?course_id=${courseID}` : '';
    return this.http.get<TeacherCourseProgress>(`${this.baseUrl}/teacher/course-progress${params}`);
  }

  getTeacherTopicMastery(courseID?: string): Observable<TopicMastery[]> {
    const params = courseID ? `?course_id=${courseID}` : '';
    return this.http.get<TopicMastery[]>(`${this.baseUrl}/teacher/topic-mastery${params}`);
  }

  getTeacherCommonErrors(): Observable<CommonError[]> {
    return this.http.get<CommonError[]>(`${this.baseUrl}/teacher/common-errors`);
  }

  getTeacherStudentProgress(courseID?: string): Observable<StudentProgress[]> {
    const params = courseID ? `?course_id=${courseID}` : '';
    return this.http.get<StudentProgress[]>(`${this.baseUrl}/teacher/student-progress${params}`);
  }
}
