# i18n (es/en) + Niveles Educativos Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Agregar soporte de idioma español/inglés (runtime) y un nivel educativo por usuario (secundaria/universidad) que filtra los conceptos ofrecidos y ajusta la dificultad.

**Architecture:** Sin migrar a `@angular/localize` (sería un refactor masivo de los 15+ templates inline), se implementa un `LanguageService` (signals) con diccionarios `es`/`en` y un pipe `translate` que busca por clave con fallback a la cadena original (key = texto en español). El idioma se persiste en `localStorage`. El `LayoutComponent` aplica traducciones en navegación y el toggle de idioma vive en `settings`. A nivel de datos, se agrega la columna `education_level` a `users` (migración idempotente en `internal/database/database.go`), el registro/login la envía, y las consultas de conceptos de `api/knowledge.go` filtran por nivel; `api/adaptive/difficulty.go` clampa la dificultad máxima según el nivel.

**Tech Stack:** Angular 20 (standalone, signals), Go (chi, pgx), PostgreSQL.

## Global Constraints

- El pipe `translate` debe tener fallback: si la clave no está en el diccionario activo, muestra la cadena original (no dejar huecos en inglés).
- `LanguageService` es `providedIn: 'root'`, persistente en `localStorage` bajo `matematicarag.lang` (`'es'` default).
- `education_level` valores válidos: `'secondary'` (secundaria) y `'university'` (universidad). Default: `'university'`.
- El filtrado por nivel NUNCA rompe consultas existentes: si `education_level` está vacío, no filtrar.
- Verificación backend: `go build ./...` y `go test ./...` (desde `api/`). Frontend: `npm run build` (desde `frontend/`).
- Solo se traducen las piezas de UI core (navegación, login/registro, settings, dashboard, logros, notificaciones). El contenido pedagógico (explicaciones del tutor, agentes, ejercicios) queda en español en esta fase.
- No se agregan dependencias npm nuevas ni Go nuevas.

---
### Task 1: LanguageService + pipe translate

**Files:**
- Create: `frontend/src/app/core/services/language.service.ts`
- Create: `frontend/src/app/shared/translate.pipe.ts`
- Create: `frontend/src/app/shared/i18n-dictionaries.ts`

**Interfaces:**
- Consumes: nada. Produce:
  - `LanguageService.lang: Signal<'es'|'en'>`, `LanguageService.setLang(lang)`, `LanguageService.t(key): string`
  - `TranslatePipe.transform(key: string): string`
  - Dicionarios `ES_DICT` y `EN_DICT` (Record<string,string>).

- [ ] **Step 1: Crear diccionarios**

```typescript
export const ES_DICT: Record<string, string> = {
  'app.title': 'Tutor Inteligente de Matemática',
  'nav.principal': 'Principal',
  'nav.contenido': 'Contenido',
  'nav.aprendizaje': 'Aprendizaje',
  'nav.gestion': 'Gestión',
  'nav.chat': 'Chat',
  'nav.agente': 'Agente',
  'nav.matematica': 'Matemática',
  'nav.tutor': 'Tutor',
  'nav.documentos': 'Documentos',
  'nav.material': 'Material',
  'nav.bdvectorial': 'BD Vectorial',
  'nav.historial': 'Historial',
  'nav.evaluaciones': 'Evaluaciones',
  'nav.analiticas': 'Analíticas',
  'nav.mi-progreso': 'Mi Progreso',
  'nav.aprendizaje-link': 'Aprendizaje',
  'nav.logros': 'Logros',
  'nav.panel': 'Panel',
  'nav.panel-profesor': 'Panel Profesor',
  'nav.configuracion': 'Configuración',
  'common.tema': 'Cambiar tema',
  'common.cerrar-sesion': 'Cerrar sesión',
  'common.notificaciones': 'Notificaciones',
  'login.title': 'Iniciar Sesión',
  'login.email': 'Email',
  'login.password': 'Contraseña',
  'login.submit': 'Iniciar Sesión',
  'login.no-cuenta': '¿No tenés cuenta? Registrate',
  'register.title': 'Registrarse',
  'register.name': 'Nombre',
  'register.lastName': 'Apellido',
  'register.submit': 'Crear cuenta',
  'register.education-level': 'Nivel educativo',
  'register.education-secondary': 'Secundaria',
  'register.education-university': 'Universidad',
  'settings.title': 'Configuración',
  'settings.language': 'Idioma',
  'settings.language-es': 'Español',
  'settings.language-en': 'English',
  'gamification.puntos': 'Puntos',
  'gamification.nivel': 'Nivel',
  'gamification.racha-actual': 'Racha actual',
  'gamification.mejor-racha': 'Mejor racha',
  'gamification.medallas': 'Medallas',
  'notif.title': 'Notificaciones',
  'notif.empty': 'Sin notificaciones.',
  'notif.mark-all': 'Marcar todas como leídas',
  'offline.banner': 'Estás sin conexión. Algunas funciones pueden no estar disponibles.',
  'offline.retry': 'Volver a intentar',
};

export const EN_DICT: Record<string, string> = {
  'app.title': 'Smart Math Tutor',
  'nav.principal': 'Main',
  'nav.contenido': 'Content',
  'nav.aprendizaje': 'Learning',
  'nav.gestion': 'Management',
  'nav.chat': 'Chat',
  'nav.agente': 'Agent',
  'nav.matematica': 'Math',
  'nav.tutor': 'Tutor',
  'nav.documentos': 'Documents',
  'nav.material': 'Material',
  'nav.bdvectorial': 'Vector DB',
  'nav.historial': 'History',
  'nav.evaluaciones': 'Assessments',
  'nav.analiticas': 'Analytics',
  'nav.mi-progreso': 'My Progress',
  'nav.aprendizaje-link': 'Learning',
  'nav.logros': 'Achievements',
  'nav.panel': 'Dashboard',
  'nav.panel-profesor': 'Teacher Panel',
  'nav.configuracion': 'Settings',
  'common.tema': 'Toggle theme',
  'common.cerrar-sesion': 'Log out',
  'common.notificaciones': 'Notifications',
  'login.title': 'Sign In',
  'login.email': 'Email',
  'login.password': 'Password',
  'login.submit': 'Sign In',
  'login.no-cuenta': 'No account? Register',
  'register.title': 'Sign Up',
  'register.name': 'First name',
  'register.lastName': 'Last name',
  'register.submit': 'Create account',
  'register.education-level': 'Education level',
  'register.education-secondary': 'High school',
  'register.education-university': 'University',
  'settings.title': 'Settings',
  'settings.language': 'Language',
  'settings.language-es': 'Spanish',
  'settings.language-en': 'English',
  'gamification.puntos': 'Points',
  'gamification.nivel': 'Level',
  'gamification.racha-actual': 'Current streak',
  'gamification.mejor-racha': 'Best streak',
  'gamification.medallas': 'Badges',
  'notif.title': 'Notifications',
  'notif.empty': 'No notifications.',
  'notif.mark-all': 'Mark all as read',
  'offline.banner': 'You are offline. Some features may be unavailable.',
  'offline.retry': 'Retry',
};
```

- [ ] **Step 2: Crear el servicio**

```typescript
import { Injectable, signal, computed } from '@angular/core';
import { ES_DICT, EN_DICT } from '../../shared/i18n-dictionaries';

export type Lang = 'es' | 'en';

@Injectable({ providedIn: 'root' })
export class LanguageService {
  private langSignal = signal<Lang>('es');
  readonly lang = computed(() => this.langSignal());

  private dicts: Record<Lang, Record<string, string>> = { es: ES_DICT, en: EN_DICT };

  constructor() {
    const saved = localStorage.getItem('matematicarag.lang') as Lang | null;
    if (saved === 'es' || saved === 'en') this.langSignal.set(saved);
  }

  setLang(lang: Lang): void {
    this.langSignal.set(lang);
    localStorage.setItem('matematicarag.lang', lang);
  }

  t(key: string): string {
    const dict = this.dicts[this.langSignal()];
    return dict[key] ?? key;
  }
}
```

- [ ] **Step 3: Crear el pipe**

```typescript
import { Pipe, PipeTransform } from '@angular/core';
import { LanguageService } from '../core/services/language.service';

@Pipe({ name: 'translate', standalone: true })
export class TranslatePipe implements PipeTransform {
  constructor(private language: LanguageService) {}
  transform(key: string): string {
    return this.language.t(key);
  }
}
```

- [ ] **Step 4: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/core/services/language.service.ts frontend/src/app/shared/translate.pipe.ts frontend/src/app/shared/i18n-dictionaries.ts
git commit -m "feat(i18n): add language service with es/en dictionaries and translate pipe"
```

### Task 2: Aplicar traducción en el Layout (navegación + offline + campana)

**Files:**
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Consumes: `LanguageService`, `TranslatePipe` (Task 1). Produce: navegación bilingüe.

- [ ] **Step 1: Inyectar el servicio y el pipe**

En `layout.component.ts`:
- Agregar imports: `import { TranslatePipe } from './translate.pipe';` y `import { LanguageService } from '../core/services/language.service';`
- Agregar `TranslatePipe` al array `imports`.
- Agregar `public language: LanguageService` al constructor.

- [ ] **Step 2: Traducir las etiquetas de navegación**

Reemplazar los textos visibles de los `<span>` de `.nav-item` y `.nav-section-label` usando el pipe. Ejemplo (Chat):

```html
<span>{{ 'nav.chat' | translate }}</span>
```

Aplicar a cada elemento de navegación (secciones Principal/Contenido/Aprendizaje/Gestión + los ítems) y a los `matTooltip` cuando corresponda:

```html
<div class="nav-section-label">{{ 'nav.principal' | translate }}</div>
<a routerLink="/chat" ... matTooltip="{{ 'nav.chat' | translate }}"><mat-icon>chat</mat-icon><span>{{ 'nav.chat' | translate }}</span></a>
```

Traducir también:
- `{{ auth.hasRole('ADMIN', 'TEACHER') ? ('nav.documentos' | translate) : ('nav.material' | translate) }}`
- el título de la campana de notificaciones (`aria-label="{{ 'common.notificaciones' | translate }}"`) y el texto interno del `NotificationCenterComponent` (`notif.title`, `notif.empty`, `notif.mark-all`).
- el banner offline: `{{ 'offline.banner' | translate }}` y `{{ 'offline.retry' | translate }}`.
- los tooltips de tema y cerrar sesión.

> Nota: `matTooltip` acepta interpolación; para textos con pipe en tooltip usar `[matTooltip]="'common.cerrar-sesion' | translate"`.

- [ ] **Step 3: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/shared/layout.component.ts frontend/src/app/modules/notifications/notification-center.component.ts
git commit -m "feat(i18n): translate layout navigation, notification center and offline banner"
```

### Task 3: Toggle de idioma en Configuración + traducción de login/registro

**Files:**
- Modify: `frontend/src/app/modules/settings/settings.component.ts`
- Modify: `frontend/src/app/modules/auth/login.component.ts`
- Modify: `frontend/src/app/modules/auth/register.component.ts`

**Interfaces:**
- Consumes: `LanguageService`, `TranslatePipe` (Task 1). Produce: selector de idioma y pantallas de auth bilingües.

- [ ] **Step 1: Agregar toggle de idioma en Settings**

En `settings.component.ts`:
- Agregar `TranslatePipe` a imports y `public language: LanguageService` al constructor.
- Agregar una sección al template (después del `<h1>Configuración</h1>`, línea 31):

```html
<div class="section">
  <h2>{{ 'settings.language' | translate }}</h2>
  <div class="language-toggle">
    <button mat-raised-button
      [class.active-lang]="language.lang() === 'es'"
      (click)="language.setLang('es')">{{ 'settings.language-es' | translate }}</button>
    <button mat-raised-button
      [class.active-lang]="language.lang() === 'en'"
      (click)="language.setLang('en')">{{ 'settings.language-en' | translate }}</button>
  </div>
</div>
```

Agregar estilo:
```css
.language-toggle { display: flex; gap: 0.5rem; }
.language-toggle button.active-lang { background: var(--accent); color: #fff; }
```

- [ ] **Step 2: Traducir Login**

En `login.component.ts`: agregar `TranslatePipe` a imports e `LanguageService` al constructor, y reemplazar los textos `Iniciar Sesión`, `Email`, `Contraseña`, `¿No tenés cuenta? Registrate` por sus claves con el pipe (`login.*`).

- [ ] **Step 3: Traducir Registro (incluye nivel educativo)**

En `register.component.ts`:
- Agregar `TranslatePipe` e `LanguageService`.
- Traducir `Nombre`, `Apellido`, `Email`, `Contraseña`, `Crear cuenta` (`register.*`).
- En el formulario, agregar el campo de nivel educativo con las opciones `secondary`/`university` y etiquetas traducidas:

```html
<select [(ngModel)]="educationLevel" class="form-select" required>
  <option value="university">{{ 'register.education-university' | translate }}</option>
  <option value="secondary">{{ 'register.education-secondary' | translate }}</option>
</select>
```

- [ ] **Step 4: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/app/modules/settings/settings.component.ts frontend/src/app/modules/auth/login.component.ts frontend/src/app/modules/auth/register.component.ts
git commit -m "feat(i18n): add language toggle in settings and translate auth screens with education level field"
```

### Task 4: Backend — columna education_level + persistencia en registro

**Files:**
- Modify: `internal/database/database.go:31-40` (tabla users)
- Modify: `api/auth.go` (register)

**Interfaces:**
- Consumes: nada. Produce:
  - Columna `education_level VARCHAR(20) NOT NULL DEFAULT 'university'` en `users`.
  - Registro acepta y persiste `education_level`.
  - `GET /api/auth/me` o el objeto `user` de login/register incluye `education_level`.

- [ ] **Step 1: Agregar la columna**

En `internal/database/database.go`, en el slice `migrations`, agregar (después del `CREATE TABLE IF NOT EXISTS users`):

```go
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS education_level VARCHAR(20) NOT NULL DEFAULT 'university'`,
```

- [ ] **Step 2: Persistir en el registro**

En `api/auth.go`, en la función de registro:
- Aceptar `educationLevel` en el request (opcional; default `university`).
- Incluir la columna en el `INSERT` y en la respuesta del usuario.

Ejemplo de cambio en el INSERT (ajustar nombres reales):

```go
	educationLevel := req.EducationLevel
	if educationLevel == "" {
		educationLevel = "university"
	}
	// en el INSERT agregar education_level y pasar educationLevel como parámetro
```

Y agregar `education_level` al struct/JSON del usuario devuelto en `login`, `register` y `refresh`:

```json
"education_level": "university"
```

- [ ] **Step 3: Verificar build y tests**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: build y tests OK.

- [ ] **Step 4: Commit**

```bash
git add internal/database/database.go api/auth.go
git commit -m "feat(levels): add education_level column and persist it on register"
```

### Task 5: Filtrar conceptos por nivel educativo + clamp de dificultad

**Files:**
- Modify: `api/knowledge.go` (consultas de conceptos)
- Modify: `api/adaptive/difficulty.go` (clamp por nivel)

**Interfaces:**
- Consumes: `education_level` del usuario autenticado (Task 4). Produce: conceptos filtrados y dificultad máxima acotada para `secondary`.

- [ ] **Step 1: Filtrar conceptos por nivel**

En `api/knowledge.go`, en la consulta que lista conceptos, obtener el `education_level` del usuario (los handlers ya tienen `studentID` desde el contexto):

```go
	var educationLevel string
	err := db.QueryRow(r.Context(), `SELECT education_level FROM users WHERE id = $1`, studentID).Scan(&educationLevel)
	if err != nil || educationLevel == "" {
		educationLevel = "university"
	}
```

Y agregar el filtro a las consultas de conceptos (si la tabla `concepts` tiene `education_level`; si no, agregar la columna con migración idempotente en la misma Task):

```go
	// agregar columna en migraciones si no existe:
	// `ALTER TABLE concepts ADD COLUMN IF NOT EXISTS education_level VARCHAR(20) NOT NULL DEFAULT 'university'`
	// y filtrar:
	// WHERE ($1 = 'university' OR education_level = $1 OR education_level = 'both')
```

> Nota de implementación: decidir entre (a) marcar cada concepto con su nivel en el seed de `internal/database/database.go`/`api/knowledge.go`, o (b) solo filtrar por nivel cuando la columna existe. La opción más segura es (b): agregar la columna con default `university`, marcar un puñado de conceptos base como `both`, y filtrar `WHERE education_level IN ('both', $1)` para que secundaria vea los conceptos compartidos.

- [ ] **Step 2: Clampar dificultad máxima para secundaria**

En `api/adaptive/difficulty.go`, en `SelectDifficulty`/`ClampDifficulty`, aceptar el nivel educativo del estudiante y acotar el rango:

```go
func (d *DifficultyEngine) ClampDifficulty(level int, educationLevel string) int {
	if educationLevel == "secondary" && level > 3 {
		level = 3
	}
	if level < 1 {
		level = 1
	}
	if level > d.config.MaxDifficulty {
		level = d.config.MaxDifficulty
	}
	return level
}
```

Propagar `educationLevel` desde el caller (las funciones que llaman `ClampDifficulty` ya reciben el `studentID`; consultar el nivel una vez y pasarlo).

- [ ] **Step 3: Verificar build y tests**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: build y tests OK. Si `adaptive_test.go` llama `ClampDifficulty` con la firma vieja, actualizar las llamadas del test (verificar con `grep -rn 'ClampDifficulty' api/`).

- [ ] **Step 4: Commit**

```bash
git add api/knowledge.go api/adaptive/difficulty.go internal/database/database.go
git commit -m "feat(levels): filter concepts by education level and clamp difficulty for secondary"
```

### Task 6: Verificación final

- [ ] **Step 1: Tests de backend**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: todo OK.

- [ ] **Step 2: Build frontend**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 3: Test manual de i18n**

Levantar la app, iniciar sesión, ir a Configuración, cambiar idioma a English.
Expected: la navegación, el offline banner y las etiquetas de login/registro cambian a inglés; al recargar persiste; al volver a español, todo en español.

- [ ] **Step 4: Test manual de niveles**

Registrar un usuario con nivel "Secundaria".
Expected: la dificultad máxima de ejercicios adaptativos se clampa a 3 y los conceptos visibles son los compartidos/`both`.
