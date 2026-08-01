import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface Achievement {
  id: string;
  code: string;
  title: string;
  description: string;
  icon: string;
  points: number;
  unlocked: boolean;
  unlocked_at?: string;
}

export interface GamificationSummary {
  points: number;
  level: number;
  level_name: string;
  next_level_points: number;
  current_streak: number;
  best_streak: number;
  achievements: Achievement[];
  recent_activities: { source: string; points: number; created_at: string; metadata?: any }[];
}

@Injectable({ providedIn: 'root' })
export class GamificationService {
  private baseUrl = environment.apiUrl + '/api';

  constructor(private http: HttpClient) {}

  getSummary(): Observable<GamificationSummary> {
    return this.http.get<GamificationSummary>(`${this.baseUrl}/gamification/me`);
  }
}
