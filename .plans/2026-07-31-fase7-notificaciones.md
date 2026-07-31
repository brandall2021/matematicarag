# Notificaciones y Alertas en App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Agregar un centro de notificaciones in-app con campana en el sidebar: listado, conteo de no leídas y marcado como leídas. Las notificaciones se generan desde flujos existentes (evaluación calificada, documento indexado, alerta académica, logro desbloqueado).

**Architecture:** Una tabla nueva `notifications` en `internal/database/database.go` y un archivo `api/notifications.go` con structs, una función productora `CreateNotification` y handlers bajo `/api/notifications/*` montados en el grupo autenticado (`cmd/server/main.go:101-119`). Se inyectan llamadas `CreateNotification` en los flujos existentes: `api/grading.go` (evaluación calificada), `api/indexer.go` (documento indexado), `api/alerts.go` (alerta académica creada) y `api/gamification.go` (logro desbloqueado). El frontend agrega `NotificationService` y un componente `notification-center` con campana y badge de no leídas en `layout.component.ts`.

**Tech Stack:** Go (chi, pgx), Angular 20 (standalone, signals), PostgreSQL.

## Global Constraints

- Copy en español rioplatense con acentos (`Notificaciones`, `Sin notificaciones`, `marcar como leída`, `Evaluación calificada`, `Documento indexado`).
- No alterar el comportamiento de los flujos existentes: `CreateNotification` es no-bloqueante y nunca hace fallar el flujo principal (los errores se loguean).
- Las notificaciones se almacenan en BD (no email, no push web en esta fase). El re-polling usa `setInterval` cada 60s en el layout.
- Verificación backend: `go build ./...` y `go test ./...` (desde `api/`). Frontend: `npm run build` (desde `frontend/`).

---
### Task 1: Esquema — tabla notifications

**Files:**
- Modify: `internal/database/database.go:27-40` (slice `migrations`)

**Interfaces:**
- Consumes: nada. Produce: tabla `notifications(id UUID, user_id UUID, type VARCHAR, title VARCHAR, message TEXT, link VARCHAR, read BOOLEAN, created_at)` con índice por usuario.

- [ ] **Step 1: Agregar la tabla al slice de migraciones**

En `internal/database/database.go`, en el slice `migrations`, agregar:

```go
	`CREATE TABLE IF NOT EXISTS notifications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		type VARCHAR(50) NOT NULL,
		title VARCHAR(255) NOT NULL,
		message TEXT NOT NULL,
		link VARCHAR(500),
		read BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, read)`,
```

- [ ] **Step 2: Verificar build**

Run: `go build ./...` (desde `api/`)
Expected: build exitoso.

- [ ] **Step 3: Commit**

```bash
git add internal/database/database.go
git commit -m "feat(notifications): add notifications table"
```

### Task 2: api/notifications.go — productor y handlers

**Files:**
- Create: `api/notifications.go`

**Interfaces:**
- Consumes: `db *pgxpool.Pool`, `cfg *config.Config`, middleware de auth.
- Produce:
  - `type Notification { ID, Type, Title, Message, Link string; Read bool; CreatedAt string }`
  - `func CreateNotification(ctx, db, userID, notifType, title, message, link string) error`
  - `func NotificationsRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router)`:
    - `GET /api/notifications` → lista (default 20, con `?unread=true` y `?limit=`)
    - `PUT /api/notifications/{id}/read` → marca como leída
    - `PUT /api/notifications/read-all` → marca todas como leídas
    - `GET /api/notifications/unread-count` → `{ "count": N }`

- [ ] **Step 1: Crear el archivo**

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Link      string `json:"link"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
}

func CreateNotification(ctx context.Context, db *pgxpool.Pool, userID, notifType, title, message, link string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, message, link)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, notifType, title, message, link)
	return err
}

func NotificationsRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			onlyUnread := r.URL.Query().Get("unread") == "true"
			limit := 20
			if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
				limit = l
			}

			query := `SELECT id, type, title, message, COALESCE(link, ''), read, created_at
			          FROM notifications WHERE user_id = $1`
			args := []any{userID}
			if onlyUnread {
				query += ` AND read = FALSE`
			}
			query += ` ORDER BY created_at DESC LIMIT ` + strconv.Itoa(limit)

			rows, err := db.Query(r.Context(), query, args...)
			if err != nil {
				http.Error(w, `{"error":"failed to get notifications"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			notifications := []Notification{}
			for rows.Next() {
				var n Notification
				var created time.Time
				if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.Link, &n.Read, &created); err != nil {
					continue
				}
				n.CreatedAt = created.Format(time.RFC3339)
				notifications = append(notifications, n)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(notifications)
		})

		r.Get("/unread-count", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			var count int
			if err := db.QueryRow(r.Context(),
				`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`, userID).Scan(&count); err != nil {
				http.Error(w, `{"error":"failed to count notifications"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int{"count": count})
		})

		r.Put("/{notificationID}/read", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			notificationID := chi.URLParam(r, "notificationID")
			_, err := db.Exec(r.Context(),
				`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`,
				notificationID, userID)
			if err != nil {
				http.Error(w, `{"error":"failed to mark notification as read"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Put("/read-all", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			_, err := db.Exec(r.Context(),
				`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID)
			if err != nil {
				http.Error(w, `{"error":"failed to mark all as read"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}
```

- [ ] **Step 2: Verificar build**

Run: `go build ./...` (desde `api/`)
Expected: build exitoso.

- [ ] **Step 3: Commit**

```bash
git add api/notifications.go
git commit -m "feat(notifications): add notifications producer and API handlers"
```

### Task 3: Montar rutas e inyectar productores en flujos existentes

**Files:**
- Modify: `cmd/server/main.go:101-119`
- Modify: `api/grading.go` (evaluación calificada → notificar al alumno)
- Modify: `api/indexer.go` (documento indexado → notificar a TEACHER/ADMIN)
- Modify: `api/alerts.go` (alerta académica creada → notificar al alumno)
- Modify: `api/gamification.go` (logro desbloqueado → notificar al alumno)

**Interfaces:**
- Consumes: `CreateNotification`, `NotificationsRoutes` (Task 2). Produce: notificaciones reales en flujos.

- [ ] **Step 1: Montar rutas**

En `cmd/server/main.go`, grupo autenticado (líneas 101-119), después de `r.Route("/gamification", api.GamificationRoutes(db, cfg))` agregar:

```go
			r.Route("/notifications", api.NotificationsRoutes(db, cfg))
```

- [ ] **Step 2: Notificar al alumno cuando se califica su evaluación**

En `api/grading.go`, en la función que persiste la nota final de una evaluación, después de guardar el resultado (junto a la lógica existente que registra puntos de gamificación si aplica), agregar:

```go
		_ = CreateNotification(ctx, db, studentID, "assessment_graded",
			"Evaluación calificada",
			"Tu evaluación fue calificada. Revisá el resultado en Evaluaciones.",
			"/assessment")
```

> Ajustar `studentID`, `ctx` y `db` a los nombres reales en scope del archivo. Si el archivo ya llama `CreateNotification` para otro evento, usar el mismo patrón.

- [ ] **Step 3: Notificar a docentes/admins cuando se indexa un documento**

En `api/indexer.go`, en la función que completa la indexación de un documento, después de finalizar el indexado agregar (consultar todos los usuarios TEACHER/ADMIN):

```go
		rows, err := db.Query(r.Context(), `SELECT id FROM users WHERE role IN ('TEACHER','ADMIN')`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var teacherID string
				if rows.Scan(&teacherID) == nil {
					_ = CreateNotification(r.Context(), db, teacherID, "document_indexed",
						"Documento indexado",
						"Un documento fue indexado en la base vectorial.",
						"/documents")
				}
			}
			rows.Close()
		}
```

> Ajustar el handler real (si `r.Context()` no aplica, usar `context.Background()`). Este paso puede moverse a una goroutine no bloqueante si el índice es lento.

- [ ] **Step 4: Notificar cuando se crea una alerta académica**

En `api/alerts.go`, en la función que crea la alerta (`CheckAlerts` o el handler `POST /check`), después de insertar la alerta agregar:

```go
	_ = CreateNotification(ctx, db, studentID, "academic_alert",
		"Alerta académica",
		"Tenés una nueva alerta académica. Revisala para no perderte el detalle.",
		"/my-progress")
```

> Ajustar nombres reales. La creación de alertas suele estar centralizada en una función `CreateAlert` o similar; buscar `INSERT INTO academic_alerts` para localizar el punto exacto.

- [ ] **Step 5: Notificar al desbloquear un logro**

En `api/gamification.go`, dentro de `CheckAchievements`, en el bloque donde se inserta el logro (después de `RecordActivity`), agregar:

```go
				_ = CreateNotification(ctx, db, studentID, "achievement_unlocked",
					"¡Logro desbloqueado!",
					"Desbloqueaste el logro: "+a.Title,
					"/logros")
```

- [ ] **Step 6: Verificar build y tests**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: build y tests OK.

- [ ] **Step 7: Commit**

```bash
git add cmd/server/main.go api/grading.go api/indexer.go api/alerts.go api/gamification.go
git commit -m "feat(notifications): mount routes and emit notifications from grading, indexing, alerts and achievements"
```

### Task 4: Frontend — servicio y campana de notificaciones

**Files:**
- Create: `frontend/src/app/core/services/notification.service.ts`
- Create: `frontend/src/app/modules/notifications/notification-center.component.ts`
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Consumes: `GET /api/notifications`, `GET /api/notifications/unread-count`, `PUT /api/notifications/{id}/read`, `PUT /api/notifications/read-all` (Task 2).
- Produce: `NotificationService.getNotifications()`, `NotificationService.getUnreadCount()`, `NotificationService.markRead(id)`, `NotificationService.markAllRead()`.

- [ ] **Step 1: Crear el servicio**

```typescript
import { Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, interval, tap } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface AppNotification {
  id: string;
  type: string;
  title: string;
  message: string;
  link: string;
  read: boolean;
  created_at: string;
}

@Injectable({ providedIn: 'root' })
export class NotificationService {
  private baseUrl = environment.apiUrl + '/api';
  private unreadSignal = signal(0);
  readonly unread = computed(() => this.unreadSignal());
  open = false;

  constructor(private http: HttpClient) {
    interval(60000).subscribe(() => this.refreshUnread());
  }

  getNotifications(): Observable<AppNotification[]> {
    return this.http.get<AppNotification[]>(`${this.baseUrl}/notifications?limit=20`);
  }

  refreshUnread(): void {
    this.http.get<{ count: number }>(`${this.baseUrl}/notifications/unread-count`)
      .subscribe({ next: r => this.unreadSignal.set(r.count), error: () => {} });
  }

  markRead(id: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/notifications/${id}/read`, {})
      .pipe(tap(() => this.unreadSignal.update(n => Math.max(0, n - 1))));
  }

  markAllRead(): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/notifications/read-all`, {})
      .pipe(tap(() => this.unreadSignal.set(0)));
  }
}
```

- [ ] **Step 2: Crear el componente de campana**

```typescript
import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { Router } from '@angular/router';
import { NotificationService, AppNotification } from '../../core/services/notification.service';

@Component({
  selector: 'app-notification-center',
  standalone: true,
  imports: [CommonModule, MatIconModule],
  template: `
    <div class="notif-wrap" (click)="stop($event)">
      <button mat-icon-button class="notif-bell" (click)="toggle()" [attr.aria-label]="'Notificaciones'">
        <mat-icon>notifications</mat-icon>
        @if (service.unread() > 0) {
          <span class="notif-badge">{{ service.unread() > 9 ? '9+' : service.unread() }}</span>
        }
      </button>

      @if (open()) {
        <div class="notif-panel">
          <div class="notif-header">
            <span>Notificaciones</span>
            <button mat-button class="notif-mark-all" (click)="markAllRead()">Marcar todas como leídas</button>
          </div>
          @if (items(); as list) {
            @if (list.length === 0) {
              <div class="notif-empty">Sin notificaciones.</div>
            } @else {
              <div class="notif-list">
                @for (n of list; track n.id) {
                  <div class="notif-item" [class.unread]="!n.read" (click)="openNotification(n)">
                    <div class="notif-title">{{ n.title }}</div>
                    <div class="notif-message">{{ n.message }}</div>
                    <div class="notif-time">{{ n.created_at | date:'dd/MM/yyyy HH:mm' }}</div>
                  </div>
                }
              </div>
            }
          } @else {
            <div class="notif-empty">Cargando...</div>
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .notif-wrap { position: relative; }
    .notif-bell { width: 32px; height: 32px; color: var(--text-secondary); position: relative; }
    .notif-bell:hover { color: var(--accent); }
    .notif-badge {
      position: absolute; top: 2px; right: 2px;
      background: var(--warn, #e53935); color: #fff;
      border-radius: 10px; font-size: 0.6rem; font-weight: 700;
      min-width: 16px; height: 16px; display: flex; align-items: center; justify-content: center;
      padding: 0 3px;
    }
    .notif-panel {
      position: absolute; top: 2.5rem; right: 0; z-index: 1400;
      width: 320px; max-height: 420px; overflow-y: auto;
      background: var(--surface-elevated); border: 1px solid var(--border);
      border-radius: var(--radius-md); box-shadow: var(--shadow-lg);
      display: flex; flex-direction: column;
    }
    .notif-header {
      display: flex; justify-content: space-between; align-items: center;
      padding: 0.6rem 0.75rem; border-bottom: 1px solid var(--border);
      font-weight: 600; font-size: 0.85rem; position: sticky; top: 0;
      background: var(--surface-elevated);
    }
    .notif-mark-all { font-size: 0.7rem; color: var(--accent-text); cursor: pointer; background: none; border: none; }
    .notif-empty { padding: 1.5rem; text-align: center; color: var(--text-secondary); font-size: 0.85rem; }
    .notif-item { padding: 0.6rem 0.75rem; border-bottom: 1px solid var(--border-light); cursor: pointer; }
    .notif-item:hover { background: var(--accent-muted); }
    .notif-item.unread { border-left: 3px solid var(--accent); }
    .notif-title { font-weight: 600; font-size: 0.85rem; }
    .notif-message { font-size: 0.78rem; color: var(--text-secondary); margin-top: 0.15rem; }
    .notif-time { font-size: 0.68rem; color: var(--text-tertiary); margin-top: 0.2rem; }
  `]
})
export class NotificationCenterComponent implements OnInit {
  open = signal(false);
  items = signal<AppNotification[] | null>(null);

  constructor(public service: NotificationService, private router: Router) {}

  ngOnInit(): void {
    this.service.refreshUnread();
  }

  toggle(): void {
    this.open.update(o => !o);
    if (this.open()) this.load();
  }

  load(): void {
    this.service.getNotifications().subscribe({ next: n => this.items.set(n), error: () => this.items.set([]) });
  }

  markAllRead(): void {
    this.service.markAllRead().subscribe({ next: () => this.load() });
  }

  openNotification(n: AppNotification): void {
    this.service.markRead(n.id).subscribe();
    this.open.set(false);
    if (n.link) this.router.navigateByUrl(n.link);
  }

  stop(e: Event): void { e.stopPropagation(); }
}
```

- [ ] **Step 3: Integrar la campana en el layout**

En `frontend/src/app/shared/layout.component.ts`:
- Agregar import: `import { NotificationCenterComponent } from '../modules/notifications/notification-center.component';`
- Agregar `NotificationCenterComponent` al array `imports` del componente.
- En el `footer-actions` del sidebar (líneas 64-71), antes del botón de tema, agregar:

```html
            <app-notification-center></app-notification-center>
```

- [ ] **Step 4: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/core/services/notification.service.ts frontend/src/app/modules/notifications frontend/src/app/shared/layout.component.ts
git commit -m "feat(notifications): add notification center service, bell component and layout integration"
```

### Task 5: Verificación final

- [ ] **Step 1: Tests de backend**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: todo OK.

- [ ] **Step 2: Test manual**

Con servidor y BD levantados: iniciar sesión como alumno, que el docente califique una evaluación (o forzar un logro en gamificación).
Expected: la campana muestra un badge con el conteo, el panel lista la notificación, al abrirla se marca como leída y redirige al link.

- [ ] **Step 3: Build frontend**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.
