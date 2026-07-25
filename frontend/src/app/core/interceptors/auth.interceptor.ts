import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from '../services/auth.service';
import { catchError, switchMap, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const token = auth.getToken();
  if (token) {
    req = req.clone({ setHeaders: { Authorization: `Bearer ${token}` } });
  }
  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      if (error.status === 401 && !req.url.includes('/api/auth/')) {
        const refreshToken = localStorage.getItem('refreshToken');
        if (refreshToken) {
          return new Promise<any>((resolve) => {
            const xhr = new XMLHttpRequest();
            xhr.open('POST', `${environment.apiUrl}/api/auth/refresh`);
            xhr.setRequestHeader('Content-Type', 'application/json');
            xhr.onload = () => {
              if (xhr.status === 200) {
                const res = JSON.parse(xhr.responseText);
                localStorage.setItem('token', res.token);
                localStorage.setItem('refreshToken', res.refreshToken);
                auth['tokenSignal'].set(res.token);
                if (res.user) {
                  localStorage.setItem('user', JSON.stringify(res.user));
                  auth['currentUserSignal'].set(res.user);
                }
                const newReq = req.clone({ setHeaders: { Authorization: `Bearer ${res.token}` } });
                next(newReq).subscribe(r => resolve(r));
              } else {
                auth.logout();
                resolve(error);
              }
            };
            xhr.onerror = () => resolve(error);
            xhr.send(JSON.stringify({ refreshToken }));
          });
        }
      }
      return throwError(() => error);
    })
  );
};
