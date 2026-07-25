import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from '../services/auth.service';
import { HttpClient } from '@angular/common/http';
import { catchError, switchMap, throwError, Observable, shareReplay, of } from 'rxjs';
import { environment } from '../../../environments/environment';

let refresh$: Observable<any> | null = null;

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const http = inject(HttpClient);
  const token = auth.getToken();
  if (token) {
    req = req.clone({ setHeaders: { Authorization: `Bearer ${token}` } });
  }
  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      if (error.status === 401 && !req.url.includes('/api/auth/')) {
        const refreshToken = localStorage.getItem('refreshToken');
        if (!refreshToken) {
          auth.logout();
          return throwError(() => error);
        }

        if (!refresh$) {
          refresh$ = http.post<any>(`${environment.apiUrl}/api/auth/refresh`, { refreshToken }).pipe(
            shareReplay(1),
            catchError((refreshError) => {
              refresh$ = null;
              auth.logout();
              return throwError(() => refreshError);
            })
          );
        }

        return refresh$.pipe(
          switchMap((res) => {
            refresh$ = null;
            localStorage.setItem('token', res.token);
            localStorage.setItem('refreshToken', res.refreshToken);
            auth['tokenSignal'].set(res.token);
            if (res.user) {
              localStorage.setItem('user', JSON.stringify(res.user));
              auth['currentUserSignal'].set(res.user);
            }
            const newReq = req.clone({ setHeaders: { Authorization: `Bearer ${res.token}` } });
            return next(newReq);
          })
        );
      }
      return throwError(() => error);
    })
  );
};
