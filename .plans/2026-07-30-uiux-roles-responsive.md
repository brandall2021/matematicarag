# UI/UX por Rol + Responsive + Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make MatematicaRAG usable and role-correct at 1366x768 and 390x844 by adapting navigation and permissions per role, improving the Chat/Tutor/Documentos/Evaluaciones/Progreso/Panel/Historial screens, fixing Spanish copy, and hardening role protection in the backend.

**Architecture:** Backend-first: restrict document upload/delete to ADMIN/TEACHER and make the document list return all material to STUDENT in `api/documents.go`. Then the Angular frontend adapts per role via the existing `AuthService.hasRole()` signal and the existing `roleGuard` + `data.roles` router pattern, with copy/UX fixes confined to component templates and styles. Verification is `go test`/`go build` for the API and `npm run build` plus a two-viewport manual matrix for the SPA (the frontend has no test runner configured).

**Tech Stack:** Go 1.25 + chi v5 (backend), Angular 20 standalone + Angular Material + signals, mathlive, PostgreSQL 16 + Qdrant.

## Global Constraints

- Roles are `ADMIN`, `TEACHER`, `STUDENT`. UI labels: `Administrador`, `Profesor`, `Alumno`.
- Copy rules: Rioplatense voseo, always with accents (`subí`, `arrastrá`, `hacé`, `resolvé`, `más`, `tardó`, `aún`, `débil`, `acción`, `sesión`, `descripción`, `configuración`). No `TODO`/`TBD` in shipped text.
- Viewports to verify: 1366x768 (desktop) and 390x844 (mobile).
- Only `api/documents.go` changes server logic (role enforcement + material listing). Everything else is frontend template/styles/guards. Do not alter math/chat/assessment backend behavior.
- Frontend verification command (run from `frontend/`): `npm run build`. Backend: `go build ./...` and `go test ./...` (run from `api/`).
- No new npm dependencies; no new backend dependencies.
- Follow existing patterns: standalone components with inline `template:`/`styles:`, `signals`, `@if` control flow, `auth.hasRole(...)` for role checks, `roleGuard` + `data: { roles }` for route protection.

---

### Task 1: Backend — role enforcement for documents + student material listing

**Files:**
- Modify: `api/documents.go:90` (upload handler), `api/documents.go:138` (list handler), `api/documents.go:196` (delete handler)
- Create: `api/documents_test.go`
- Modify: `api/middleware.go` — no change needed; `RoleKey` already populated (see `api/middleware.go:15,66`).

**Interfaces:**
- Consumes: `RoleKey` context key (`string`), `UserIDKey` context key, existing `DocumentRoutes(db, cfg)`.
- Produces:
  - `func canManageDocuments(role string) bool` — true only for `ADMIN`/`TEACHER`.
  - `func documentsListQuery(role string) string` — returns per-user query for managers, all-docs query for students.

- [ ] **Step 1: Write the failing test**

Create `api/documents_test.go`:

```go
package api

import (
	"strings"
	"testing"
)

func TestCanManageDocuments(t *testing.T) {
	cases := map[string]bool{
		"ADMIN":   true,
		"TEACHER": true,
		"STUDENT": false,
		"":        false,
		"GUEST":   false,
	}
	for role, want := range cases {
		if got := canManageDocuments(role); got != want {
			t.Errorf("canManageDocuments(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestDocumentsListQuery(t *testing.T) {
	studentQuery := documentsListQuery("STUDENT")
	if strings.Contains(studentQuery, "WHERE uploaded_by") {
		t.Errorf("student query should not filter by owner, got: %s", studentQuery)
	}
	teacherQuery := documentsListQuery("TEACHER")
	if !strings.Contains(teacherQuery, "WHERE uploaded_by = $1") {
		t.Errorf("teacher query should filter by owner, got: %s", teacherQuery)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./...` (from `api/`)
Expected: FAIL — `undefined: canManageDocuments`.

- [ ] **Step 3: Implement the helpers and wire the handlers**

Add at the top of `api/documents.go` (next to `sanitizeText`, line ~80):

```go
func canManageDocuments(role string) bool {
	return role == "ADMIN" || role == "TEACHER"
}

func documentsListQuery(role string) string {
	if canManageDocuments(role) {
		return `SELECT id, filename, original_name, type, size, status, created_at
				FROM documents WHERE uploaded_by = $1 ORDER BY created_at DESC`
	}
	return `SELECT id, filename, original_name, type, size, status, created_at
			FROM documents ORDER BY created_at DESC`
}
```

In the upload handler (`api/documents.go:90`), immediately after `userID := ...`:

```go
role, _ := r.Context().Value(RoleKey).(string)
if !canManageDocuments(role) {
	http.Error(w, `{"error":"no autorizado: solo Admin o Profesor pueden subir documentos"}`, http.StatusForbidden)
	return
}
```

In the list handler (`api/documents.go:138`), replace the `db.Query(...)` call:

```go
role, _ := r.Context().Value(RoleKey).(string)
query := documentsListQuery(role)
args := []any{}
if canManageDocuments(role) {
	args = append(args, userID)
}
rows, err := db.Query(r.Context(), query, args...)
```

In the delete handler (`api/documents.go:196`), as the first statement:

```go
role, _ := r.Context().Value(RoleKey).(string)
if !canManageDocuments(role) {
	http.Error(w, `{"error":"no autorizado: solo Admin o Profesor pueden eliminar documentos"}`, http.StatusForbidden)
	return
}
```

- [ ] **Step 4: Run tests and build**

Run: `go test ./... && go build ./...` (from `api/`)
Expected: PASS for all tests, build succeeds.

- [ ] **Step 5: Commit**

```bash
git add api/documents.go api/documents_test.go
git commit -m "feat(api): restrict document upload/delete to ADMIN/TEACHER and expose material to students"
```

---

### Task 2: Route guards — hide BD Vectorial and Analíticas from Alumno

**Files:**
- Modify: `frontend/src/app/app.routes.ts:20` (`bdvectorial`), `frontend/src/app/app.routes.ts:25` (`analytics`)

**Interfaces:**
- Consumes: existing `roleGuard` and `data: { roles }` pattern from `frontend/src/app/core/guards/role.guard.ts`.

- [ ] **Step 1: Add roleGuard to the two routes**

In `frontend/src/app/app.routes.ts`, change:

```ts
{ path: 'bdvectorial', loadComponent: () => import('./modules/bdvectorial/bdvectorial.component').then(m => m.BdvectorialComponent) },
```
to:
```ts
{ path: 'bdvectorial', loadComponent: () => import('./modules/bdvectorial/bdvectorial.component').then(m => m.BdvectorialComponent), canActivate: [roleGuard], data: { roles: ['ADMIN', 'TEACHER'] } },
```

And change:
```ts
{ path: 'analytics', loadComponent: () => import('./modules/analytics/analytics.component').then(m => m.AnalyticsComponent) },
```
to:
```ts
{ path: 'analytics', loadComponent: () => import('./modules/analytics/analytics.component').then(m => m.AnalyticsComponent), canActivate: [roleGuard], data: { roles: ['ADMIN', 'TEACHER'] } },
```

- [ ] **Step 2: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds with no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/app.routes.ts
git commit -m "feat(ui): protect BD Vectorial and Analíticas routes for ADMIN/TEACHER only"
```

---

### Task 3: Sidebar nav by role + responsive label fixes

**Files:**
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Consumes: `AuthService` (public `auth` in component, `hasRole(...)`, `currentUser()`), `ThemeService`.
- Produces: role-conditional nav items and the translated hamburger aria-label.

- [ ] **Step 1: Make Contenido/Aprendizaje nav items role-aware and fix aria-label**

In `frontend/src/app/shared/layout.component.ts`:

Replace the hamburger (line 15):
```html
<button class="hamburger" (click)="sidebarOpen.set(!sidebarOpen())" aria-label="Toggle sidebar">
```
with:
```html
<button class="hamburger" (click)="sidebarOpen.set(!sidebarOpen())" aria-label="Alternar menú">
```

Replace the Contenido block (lines 34-37):
```html
<div class="nav-section-label">Contenido</div>
<a routerLink="/documents" ...><mat-icon>folder</mat-icon><span>Documentos</span></a>
<a routerLink="/bdvectorial" ...><mat-icon>database</mat-icon><span>BD Vectorial</span></a>
<a routerLink="/history" ...><mat-icon>history</mat-icon><span>Historial</span></a>
```
with:
```html
<div class="nav-section-label">Contenido</div>
<a routerLink="/documents" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>folder</mat-icon><span>{{ auth.hasRole('ADMIN', 'TEACHER') ? 'Documentos' : 'Material' }}</span></a>
@if (auth.hasRole('ADMIN', 'TEACHER')) {
  <a routerLink="/bdvectorial" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>database</mat-icon><span>BD Vectorial</span></a>
}
<a routerLink="/history" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>history</mat-icon><span>Historial</span></a>
```

Replace the Aprendizaje block (lines 39-43): wrap the Analíticas item in the same guard:
```html
@if (auth.hasRole('ADMIN', 'TEACHER')) {
  <a routerLink="/analytics" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>analytics</mat-icon><span>Analíticas</span></a>
}
```

- [ ] **Step 2: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/shared/layout.component.ts
git commit -m "feat(ui): role-aware sidebar navigation and translated aria-label"
```

---

### Task 4: Login/Registro — copy, accents, mobile fit

**Files:**
- Modify: `frontend/src/app/modules/auth/login.component.ts:32,35,71`
- Modify: `frontend/src/app/modules/auth/register.component.ts` (minor copy)

**Interfaces:**
- Consumes: nothing new. Produces: correct voseo/accents in auth screens.

- [ ] **Step 1: Fix login copy**

In `login.component.ts`:
- Line 32: `{{ loading() ? 'Ingresando...' : 'Iniciar Sesión' }}` → `{{ loading() ? 'Ingresando...' : 'Iniciar sesión' }}`
- Line 35: `<a routerLink="/register">Registrate</a>` — already correct Rioplatense (`registrá` + enclitic `te` is llana, no accent). Do NOT change it.
- Line 71: `'Error al iniciar sesion'` → `'Error al iniciar sesión'`

Note: the login page already has `autocomplete="email"` and `autocomplete="current-password"` (lines 22, 26) — leave them.

- [ ] **Step 2: Fix register copy and add missing fields to the create-account flow**

In `register.component.ts`:
- Line 36: `{{ loading() ? 'Creando...' : 'Registrarse' }}` — OK, leave.
- Line 39: `¿Ya tenés cuenta? <a routerLink="/login">Inicia sesión</a>` — OK.
- Add `autocomplete="name"` to the Nombre input (line 22) and `autocomplete="email"`/`autocomplete="new-password"` are already present on Email/Contraseña.

- [ ] **Step 3: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/modules/auth/login.component.ts frontend/src/app/modules/auth/register.component.ts
git commit -m "fix(ui): correct accents in login/register copy"
```

---

### Task 5: Chat — role-aware welcome with suggestions

**Files:**
- Modify: `frontend/src/app/modules/chat/chat.component.ts`

**Interfaces:**
- Consumes: `AuthService` (inject into constructor).
- Produces: personalized welcome (`¿Qué querés resolver hoy?`), send-button aria-label.

- [ ] **Step 1: Inject AuthService and personalize the welcome**

Add to the constructor (line 185):
```ts
constructor(private api: ApiService, private router: Router, private zone: NgZone, public auth: AuthService) {}
```
Add `AuthService` to the imports from `'../../core/services/auth.service'`.

Replace the welcome block (lines 40-42):
```html
<div class="welcome-icon"><mat-icon>auto_awesome</mat-icon></div>
<h2>Chat Matemático</h2>
<p>Hacé consultas sobre matemática con análisis de tus documentos de estudio.</p>
```
with:
```html
<div class="welcome-icon"><mat-icon>auto_awesome</mat-icon></div>
<h2>¿Qué querés resolver hoy?</h2>
<p>¡Hola {{ auth.currentUser()?.name || 'estudiante' }}! Hacé consultas sobre matemática con análisis de tus documentos de estudio.</p>
```

The suggestion chips (lines 43-59) already exist — keep them unchanged.

Add an aria-label to the send button (line 120):
```html
<button mat-raised-button color="primary" (click)="sendMessage()" [disabled]="!newMessage" aria-label="Enviar mensaje">
```

- [ ] **Step 2: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/modules/chat/chat.component.ts
git commit -m "feat(ui): personalized chat welcome and send button aria-label"
```

---

### Task 6: Tutor — copy, mode help, disabled-button explanation

**Files:**
- Modify: `frontend/src/app/modules/tutor/tutor.component.ts`

**Interfaces:**
- Consumes: nothing new. Produces: corrected accents + per-mode helper text.

- [ ] **Step 1: Fix accents**

In `tutor.component.ts`:
- Line 23: `<h1><mat-icon>school</mat-icon> Tutor Matematico</h1>` → `<h1><mat-icon>school</mat-icon> Tutor Matemático</h1>`
- Line 90: `{{ loading() ? 'Iniciando...' : 'Iniciar Sesión' }}` → `{{ loading() ? 'Iniciando...' : 'Iniciar sesión' }}`
- Line 212: `Calculado con motor matematico` → `Calculado con el motor matemático`
- Line 56: `<label>Expresión matemática</label>` — OK.
- Line 47: `Básico` — OK.

- [ ] **Step 2: Add contextual help per mode**

Add after the subtitle (line 24), inside the header:

```html
<p class="mode-hint">
  @switch (tutorMode()) {
    @case ('solve') { Resolvé o verificá expresiones matemáticas paso a paso, o pedí una pista (hint) para llegar a la solución. }
    @case ('tutor') { Un tutor interactivo te guía: respondé ejercicios y recibí feedback inmediato. }
    @case ('practice') { Practicá ejercicios adaptados a tu nivel para reforzar conceptos. }
    @case ('review') { Repasá temas ya trabajados para consolidar lo aprendido. }
  }
</p>
```

Add the CSS for `.mode-hint` in the component styles:
```css
.mode-hint { color: var(--text-secondary); font-size: 0.85rem; margin: 0.25rem 0 1rem 0; max-width: 720px; line-height: 1.5; }
```

- [ ] **Step 3: Explain the disabled submit button**

In the solve submit area (line 78), add a helper line right below the button:

```html
@if (loading() || (!latexValue() && !textQuery)) {
  <p class="input-hint">Escribí tu consulta para habilitar el botón Resolver.</p>
}
```
And when the Pista button is disabled at the hint limit, the existing `Pista ({{ hintIndex() }}/3)` label already communicates state — no change needed.

Add CSS:
```css
.input-hint { color: var(--text-tertiary); font-size: 0.75rem; margin: 0.5rem 0 0 0; }
```

- [ ] **Step 4: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/tutor/tutor.component.ts
git commit -m "feat(ui): tutor accents, per-mode help text and disabled-state hint"
```

---

### Task 7: Documentos — Material de estudio mode for Alumno, indexing states, copy

**Files:**
- Modify: `frontend/src/app/modules/documents/documents.component.ts`

**Interfaces:**
- Consumes: `auth.hasRole(...)` (already injected as `protected auth`), `GET /api/documents` returning all docs for STUDENT (from Task 1).

- [ ] **Step 1: Role-aware heading and subtitle**

Replace line 34-35:
```html
<h1>Gestión Documental</h1>
<p class="subtitle">Subí material de estudio (PDF, DOCX, TXT, Markdown) para indexarlo en la base vectorial</p>
```
with:
```html
@if (auth.hasRole('ADMIN', 'TEACHER')) {
  <h1>Gestión Documental</h1>
  <p class="subtitle">Subí material de estudio (PDF, DOCX, TXT, Markdown) para indexarlo en la base vectorial</p>
} @else {
  <h1>Material de estudio</h1>
  <p class="subtitle">Documentos indexados por tus docentes, disponibles para consultar en el Chat.</p>
}
```

- [ ] **Step 2: Role-aware empty state**

Replace line 110:
```html
<div class="empty">No hay documentos cargados aun.</div>
```
with:
```html
@if (auth.hasRole('ADMIN', 'TEACHER')) {
  <div class="empty">Todavía no hay documentos cargados. Subí el primer material para indexarlo en la base vectorial.</div>
} @else {
  <div class="empty">Todavía no hay material de estudio disponible. Consultá con tu docente.</div>
}
```

- [ ] **Step 3: Handle indexed-with-zero-chunks state and polling timeout copy**

In the status switch (lines 83-88), after the `@default` case add:
```html
@if (doc.status === 'indexed' && doc.chunkCount === 0) {
  <span class="status"><mat-icon>hourglass_empty</mat-icon> Sin contenido indexado</span>
}
```
(Place this *inside* the `doc-meta` div, after the existing status `@switch` block, guarded by `@if (doc.status === 'indexed' && doc.chunkCount === 0)`.)

Fix line 264: `'Tarde mas de lo esperado. Refresca para ver el estado.'` → `'Tardó más de lo esperado. Refrescá para ver el estado.'`

Also hide the upload drop-area from students is already handled by the existing `@if (auth.hasRole('ADMIN', 'TEACHER'))` wrapper (line 37) — keep it.

- [ ] **Step 4: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/documents/documents.component.ts
git commit -m "feat(ui): material de estudio mode for students and indexing state fixes"
```

---

### Task 8: Evaluaciones — differentiated empty states

**Files:**
- Modify: `frontend/src/app/modules/assessment/assessment.component.ts`

**Interfaces:**
- Consumes: `auth.hasRole(...)` (already imported and used), `Router` (already imported).

- [ ] **Step 1: Teacher empty state with CTA**

Replace lines 64-69:
```html
@if (filteredAssessments().length === 0) {
  <div class="empty-state">
    <mat-icon>quiz</mat-icon>
    <p>No hay evaluaciones {{ teacherTab() === 'draft' ? 'en borrador' : teacherTab() === 'published' ? 'publicadas' : '' }}</p>
  </div>
}
```
with:
```html
@if (filteredAssessments().length === 0) {
  <div class="empty-state">
    <mat-icon>quiz</mat-icon>
    <p>Todavía no hay evaluaciones {{ teacherTab() === 'draft' ? 'en borrador' : teacherTab() === 'published' ? 'publicadas' : 'creadas' }}</p>
    <span class="empty-hint">Creá tu primera evaluación para compartirla con tus estudiantes.</span>
    <button mat-raised-button color="primary" (click)="showCreateForm()" class="empty-cta">
      <mat-icon>add</mat-icon> Nueva Evaluación
    </button>
  </div>
}
```
Add CSS:
```css
.empty-cta { margin-top: 1rem; }
.empty-hint { display: block; color: var(--text-secondary); font-size: 0.85rem; margin-top: 0.25rem; }
```

- [ ] **Step 2: Student empty state with "Practicar con el tutor" CTA**

Replace lines 278-284:
```html
@if (studentAssessments().length === 0) {
  <div class="empty-state">
    <mat-icon>quiz</mat-icon>
    <p>Todavía no hay evaluaciones publicadas</p>
    <span class="empty-hint">Consultá con tu docente o volvé más tarde</span>
  </div>
}
```
with:
```html
@if (studentAssessments().length === 0) {
  <div class="empty-state">
    <mat-icon>quiz</mat-icon>
    <p>No hay evaluaciones disponibles</p>
    <span class="empty-hint">Consultá con tu docente o practicá con el tutor mientras tanto.</span>
    <button mat-raised-button color="primary" routerLink="/tutor" class="empty-cta">
      <mat-icon>school</mat-icon> Practicar con el tutor
    </button>
  </div>
}
```
(The `RouterLink` directive comes from `CommonModule`, already imported.)

- [ ] **Step 3: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/modules/assessment/assessment.component.ts
git commit -m "feat(ui): differentiated empty states for evaluaciones with CTAs"
```

---

### Task 9: Mi Progreso + Panel de Aprendizaje — copy and zero-state polish

**Files:**
- Modify: `frontend/src/app/modules/student-progress/student-progress.component.ts`
- Modify: `frontend/src/app/modules/adaptive-dashboard/adaptive-dashboard.component.ts`

**Interfaces:**
- Consumes: nothing new.

- [ ] **Step 1: Fix Mi Progreso copy and show explicit zero stats**

In `student-progress.component.ts`:
- Line 60: `Recomendacion:` → `Recomendación:`
- Empty state (lines 23-33) already has CTAs to Chat/Tutor — keep.
- The stats grid (lines 36-54) already renders `0%`, `0`, `0%`, `0.0h` when `dashboard()` has zero data because the template falls back to `dashboard()!.profile...` — confirm these render `0` (they will; `overall_level` defaults to 0). No change needed beyond confirming the `toFixed(0)` outputs.

- [ ] **Step 2: Fix Aprendizaje (adaptive dashboard) accents**

In `adaptive-dashboard.component.ts`:
- Line 66: `Aun no hay datos de concepto. Practica para ver tu progreso.` → `Aún no hay datos de concepto. Practicá para ver tu progreso.`
- Line 72: `Accion Recomendada` → `Acción Recomendada`
- Line 79: `debiles` (stat chip "X debiles") → `débiles`
- Line 96: `Conceptos Debiles` → `Conceptos Débiles`
- Line 129: `No hay recomendaciones pendientes. Sigue practicando!` → `No hay recomendaciones pendientes. ¡Seguí practicando!`

- [ ] **Step 3: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/modules/student-progress/student-progress.component.ts frontend/src/app/modules/adaptive-dashboard/adaptive-dashboard.component.ts
git commit -m "fix(ui): accents in Mi Progreso and Panel de Aprendizaje"
```

---

### Task 10: Panel Admin — period caption, empty-docs state, self-protection in settings

**Files:**
- Modify: `frontend/src/app/modules/dashboard/dashboard.component.ts`
- Modify: `frontend/src/app/modules/settings/settings.component.ts`

**Interfaces:**
- Consumes: `AuthService` (settings), `getAdminStats()` (dashboard, already used).

- [ ] **Step 1: Dashboard period caption + empty-documents hint**

In `dashboard.component.ts`:
- Line 12: `<h1><mat-icon>dashboard</mat-icon> Dashboard Admin</h1>` → `<h1><mat-icon>dashboard</mat-icon> Panel de Administración</h1>`
- After line 12, add a caption:
```html
<p class="period-caption">Resumen general de la plataforma</p>
```
- After the `stats-grid` (after line 37), add an empty-docs hint:
```html
@if (stats().totalDocuments === 0) {
  <div class="empty-docs">
    <mat-icon>description</mat-icon>
    <p>No hay documentos indexados todavía. Subí material desde la sección Documentos.</p>
  </div>
}
```
Add CSS:
```css
.period-caption { color: var(--text-secondary); font-size: 0.85rem; margin: -1rem 0 1.5rem 0; }
.empty-docs { display: flex; align-items: center; gap: 0.5rem; margin-top: 1rem; padding: 1rem; background: var(--surface); border: 1px solid var(--border); border-radius: 12px; color: var(--text-secondary); font-size: 0.85rem; }
.empty-docs mat-icon { color: var(--accent); }
```

- [ ] **Step 2: Settings — protect the active admin**

In `settings.component.ts`:
- Import `AuthService` and inject it:
```ts
import { AuthService } from '../../core/services/auth.service';
...
constructor(private http: HttpClient, public auth: AuthService) {}
```
- In `updateRole` (line 417), guard self-demotion before the confirm:
```ts
updateRole(userId: string, newRole: string) {
  if (this.auth.currentUser()?.id === userId && newRole !== 'ADMIN') {
    this.showMessage('No podés cambiar tu propio rol de Administrador', 'error');
    return;
  }
  const roleLabel: Record<string, string> = { STUDENT: 'Alumno', TEACHER: 'Profesor', ADMIN: 'Administrador' };
  if (!confirm(`¿Cambiar el rol a "${roleLabel[newRole] || newRole}"?`)) return;
  ...
}
```
- In `deleteUser` (line 426), guard self-deletion:
```ts
deleteUser(userId: string, name: string) {
  if (this.auth.currentUser()?.id === userId) {
    this.showMessage('No podés eliminar tu propio usuario', 'error');
    return;
  }
  if (!confirm(`Eliminar usuario "${name}"?`)) return;
  ...
}
```

- [ ] **Step 3: Settings — copy and aria-labels**

- Line 51: `placeholder="Password"` → `placeholder="Contraseña"`
- Line 116: `placeholder="Descripcion (opcional)"` → `placeholder="Descripción (opcional)"`
- Line 125: `<h3>Configuracion de IA</h3>` → `<h3>Configuración de IA</h3>`
- Line 199: `Hace click en "Agregar key" para crear una.` → `Hacé clic en "Agregar key" para crear una.`
- Lines 281, 284, 288, 293 (`mas`, `rapido`, `economico`): fix accents → `más`, `rápido`, `económico`.
- Add aria-labels to the icon-only buttons: visibility toggle (line 188) → `aria-label="Mostrar u ocultar valor"`; delete setting (line 192) → `aria-label="Eliminar configuración"`; delete user (line 77) → `aria-label="Eliminar usuario"`.

- [ ] **Step 4: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/dashboard/dashboard.component.ts frontend/src/app/modules/settings/settings.component.ts
git commit -m "feat(ui): admin panel polish and settings self-protection for active admin"
```

---

### Task 11: Historial — readable date, search, copy

**Files:**
- Modify: `frontend/src/app/modules/history/history.component.ts`

**Interfaces:**
- Consumes: `ApiService.getHistory()` (already used) returning `[{ id, title, messages, createdAt }]`.
- Produces: `query` signal (filter), formatted date.

- [ ] **Step 1: Rewrite the template**

Replace the whole `template` of `history.component.ts` (lines 9-21):

```html
<div class="container">
  <h1>Historial</h1>
  <div class="search-box">
    <mat-icon>search</mat-icon>
    <input #searchInput
           placeholder="Buscar por título..."
           (input)="query.set(searchInput.value)"
           aria-label="Buscar en el historial">
  </div>

  @if (filteredHistory().length === 0) {
    <div class="empty-state">
      @if (query()) {
        <p>No se encontraron conversaciones para "{{ query() }}".</p>
      } @else {
        <p>Aún no hay historial.</p>
      }
    </div>
  }

  @for (entry of filteredHistory(); track entry.id) {
    <div class="history-item">
      <div class="history-icon"><mat-icon>chat</mat-icon></div>
      <div class="history-body">
        <h3>{{ entry.title }}</h3>
        <p>{{ entry.messages }} mensajes · {{ entry.createdAt | date:'dd/MM/yyyy, HH:mm' }}</p>
      </div>
    </div>
  }
</div>
```

- [ ] **Step 2: Update the class + imports + styles**

Replace the imports and class body (lines 2-3 and 30-34):

```ts
import { Component, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { ApiService } from '../../core/services/api.service';
```
```ts
export class HistoryComponent implements OnInit {
  history = signal<any[]>([]);
  query = signal('');

  filteredHistory = computed(() => {
    const q = this.query().trim().toLowerCase();
    if (!q) return this.history();
    return this.history().filter(h => (h.title || '').toLowerCase().includes(q));
  });

  constructor(private api: ApiService) {}
  ngOnInit() { this.api.getHistory().subscribe(h => this.history.set(h)); }
}
```
Update the imports in the `@Component` decorator to `imports: [CommonModule, MatIconModule]`.

Replace the styles block (lines 22-28):
```css
.container { padding: 1.5rem; max-width: 800px; margin: 0 auto; background: var(--bg); color: var(--text); }
@media (min-width: 768px) { .container { padding: 2rem; } }
h1 { color: var(--accent); margin-bottom: 1.5rem; font-family: 'Newsreader', serif; }
.search-box { display: flex; align-items: center; gap: 0.5rem; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.5rem 0.75rem; margin-bottom: 1.5rem; }
.search-box mat-icon { color: var(--text-tertiary); }
.search-box input { flex: 1; border: none; outline: none; background: transparent; color: var(--text); font-size: 0.9rem; }
.history-item { display: flex; gap: 0.75rem; align-items: flex-start; background: var(--surface); padding: 1rem; border-radius: 8px; margin-bottom: 0.5rem; border: 1px solid var(--border); }
.history-icon mat-icon { color: var(--accent); margin-top: 2px; }
.history-body h3 { margin: 0 0 0.25rem 0; color: var(--text); font-size: 0.95rem; }
.history-body p { margin: 0; color: var(--text-secondary); font-size: 0.8rem; }
.empty-state { text-align: center; margin-top: 3rem; color: var(--text-secondary); }
```

- [ ] **Step 3: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/modules/history/history.component.ts
git commit -m "feat(ui): historial search, dd/MM/yyyy date format and empty states"
```

---

### Task 12: Copy sweep across remaining screens

**Files:**
- Modify: `frontend/src/app/modules/bdvectorial/bdvectorial.component.ts`
- Modify: `frontend/src/app/modules/math/math.component.ts`
- Modify: `frontend/src/app/modules/agent-chat/agent-chat.component.ts`
- Modify: `frontend/src/app/modules/analytics/analytics.component.ts`
- Modify: `frontend/src/app/modules/teacher-dashboard/teacher-dashboard.component.ts`
- Delete: `frontend/src/app/modules/admin/admin.component.ts` (unused placeholder, no route references it)

**Interfaces:**
- Consumes: nothing new.

- [ ] **Step 1: Fix remaining accents**

In `bdvectorial.component.ts`:
- Line 38: `coleccion` → `colección`; `1536 dimensiones (OpenAI text-embedding-3-small)` — OK.

In `teacher-dashboard.component.ts`:
- Line 79: `Ultima vez` → `Última vez` (column header inside `student-header`).
- Line 15: `Panel del Profesor` — OK.

In `math.component.ts`, `agent-chat.component.ts`, `analytics.component.ts`: run the grep below and fix every match; replace each missing-accent word with its accented form (`más`, `débil`, `acción`, `sesión`, `cálculo`, `matemático`, `aún`, `solución`, `validación`, `explicación`, `evaluación`, `resumen`, `introducción`, `instrucción`, `comunicación`, `configuración`, `descripción`, `verificación`).

- [ ] **Step 2: Run the accent sweep grep**

Run (from `frontend/src/app/modules`):
```bash
rg -n "matematico|Matematico|basico|aun|Aun|Accion|Debiles|sesion|Descripcion|Configuracion|coleccion|Ultima|Administracion|mas capaz|rapido|economico|Sigue practicando" --glob '*.component.ts'
```
Expected: no remaining matches (all fixed). Re-run and fix any leftovers before committing.

- [ ] **Step 3: Delete the unused admin placeholder**

Verify nothing imports it, then delete:
```bash
rg -rn "admin/admin.component" frontend/src || echo "no references"
git rm frontend/src/app/modules/admin/admin.component.ts
```

- [ ] **Step 4: Verify build**

Run: `npm run build` (from `frontend/`)
Expected: build succeeds with no errors.

- [ ] **Step 5: Commit**

```bash
git add -A frontend/src/app/modules
git commit -m "fix(ui): spanish copy sweep across remaining screens, remove unused admin placeholder"
```

---

### Task 13: Full verification — backend, build, two-viewport manual matrix

**Files:**
- (no source changes)

- [ ] **Step 1: Run backend tests and build**

Run (from `api/`): `go test ./...`
Expected: all tests PASS (including the new `documents_test.go`).

Run (from `api/`): `go build ./...`
Expected: build succeeds.

- [ ] **Step 2: Run frontend production build**

Run (from `frontend/`): `npm run build`
Expected: build succeeds with no errors/warnings.

- [ ] **Step 3: Manual matrix — viewport 1366x768 (Chrome DevTools)**

Verify each of these by logging in as STUDENT, then as ADMIN:
1. **Login/Registro** — card centered, no horizontal scroll, footer visible; `Registrate` link accented.
2. **Chat** — welcome shows `¿Qué querés resolver hoy?` + personalized name + 5 suggestion chips; chips fill the input; send button has tooltip/aria.
3. **Tutor** — header `Tutor Matemático`; mode toggle shows Resolver/Tutor/Practicar/Repaso; helper text changes per mode; Resolver disabled until input typed (hint text visible).
4. **Documentos** — STUDENT: `Material de estudio`, no upload area, empty text `Todavía no hay material de estudio disponible...`; ADMIN: `Gestión Documental` + upload area, indexed docs show `N chunks`, delete button present.
5. **Evaluaciones** — STUDENT: empty state with `Practicar con el tutor` button; ADMIN: teacher list empty state with `Nueva Evaluación` CTA.
6. **Mi Progreso** — empty state with `Ir al Chat`/`Ir al Tutor`; Aprendizaje panel has accented headings.
7. **Panel** (ADMIN) — `Panel de Administración`, 4 stats, empty-docs hint when no documents; Settings role change shows confirm, self-demotion/self-delete blocked.
8. **Historial** — date `dd/MM/yyyy, HH:mm`, search filters by title, empty state `Aún no hay historial.`.
9. **Sidebar** — STUDENT hides BD Vectorial, Analíticas, Panel, Panel Profesor, Configuración; Documentos label reads `Material`; ADMIN sees all items.

- [ ] **Step 4: Manual matrix — viewport 390x844**

Repeat the same checks with the mobile drawer: hamburger opens the sidebar, backdrop closes it, content has `padding-top` clearance under the fixed hamburger, `stats-grid` in Mi Progreso collapses to 2 columns, documents/assessment cards do not overflow horizontally, tutor button rows wrap, math-field remains usable (virtual keyboard opens).

- [ ] **Step 5: Commit (only if fixes were needed)**

If any step above required a code fix, commit it with a message describing the fix. Otherwise nothing to commit.

---

## Self-Review

**1. Spec coverage:** The 12 spec sections map to Tasks 4 (login/responsive), 3+2 (nav por rol + route protection), 5 (chat inicial), 6 (tutor), 1+7 (documentos/material + permisos backend), 8 (evaluaciones), 9 (progreso), 10 (panel admin), 11 (historial), 12+5+10 (accesibilidad/aria), 12 (copy/acentos), 13 (verificación 1366x768 / 390x844). The "Material de estudio para Alumno" requirement exposed a real backend gap (list filtered by `uploaded_by`) which Task 1 closes.

**2. Placeholder scan:** All steps contain exact strings, code, and commands. No TBD/TODO.

**3. Type consistency:** `canManageDocuments`, `documentsListQuery`, `auth.currentUser()?.id`, `query`/`filteredHistory` names are consistent across the tasks that reference them.
