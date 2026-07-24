import { Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap } from 'rxjs';
import { environment } from '../../environments/environment';

export interface AuthResponse {
  token: string;
  refreshToken: string;
  user: { id: string; email: string; name: string; lastName: string; role: string };
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private currentUserSignal = signal<any>(null);
  private tokenSignal = signal<string | null>(null);

  isAuthenticated = computed(() => this.tokenSignal() !== null);
  currentUser = computed(() => this.currentUserSignal());
  role = computed(() => this.currentUserSignal()?.role ?? '');

  constructor(private http: HttpClient, private router: Router) {
    this.loadFromStorage();
  }

  login(email: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${environment.apiUrl}/api/auth/login`, { email, password })
      .pipe(tap(res => this.handleAuth(res)));
  }

  register(email: string, password: string, name: string, lastName?: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${environment.apiUrl}/api/auth/register`, { email, password, name, lastName })
      .pipe(tap(res => this.handleAuth(res)));
  }

  logout(): void {
    localStorage.removeItem('token');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('user');
    this.currentUserSignal.set(null);
    this.tokenSignal.set(null);
    this.router.navigate(['/login']);
  }

  getToken(): string | null {
    return this.tokenSignal();
  }

  hasRole(...roles: string[]): boolean {
    return roles.includes(this.role());
  }

  private handleAuth(res: AuthResponse): void {
    localStorage.setItem('token', res.token);
    localStorage.setItem('refreshToken', res.refreshToken);
    localStorage.setItem('user', JSON.stringify(res.user));
    this.tokenSignal.set(res.token);
    this.currentUserSignal.set(res.user);
  }

  private loadFromStorage(): void {
    const token = localStorage.getItem('token');
    const user = localStorage.getItem('user');
    if (token) this.tokenSignal.set(token);
    if (user) {
      try { this.currentUserSignal.set(JSON.parse(user)); } catch { localStorage.removeItem('user'); }
    }
  }
}
