# PWA Instalable + Modo Offline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convertir el frontend Angular en una Progressive Web App instalable con service worker, precaching del app shell, caché de consultas GET (documentos y progreso) y un banner de modo offline con reintento automático al volver la conexión.

**Architecture:** Se agrega `@angular/service-worker` con configuración `ngsw-config.json` (precache de assets del build + dataGroups para API GET con estrategia `freshness`/`performance`). El ServiceWorker se registra en `app.config.ts` vía `provideServiceWorker`. Un servicio `OfflineService` expone un signal `isOffline` basado en `navigator.onLine` + eventos `online`/`offline`; el `LayoutComponent` muestra un banner y un botón de reintento. El backend ya sirve el `dist/browser` como SPA estática (`cmd/server/main.go:174-237`), y `spaHandler` sirve cualquier archivo existente (incluido `ngsw-worker.js`, `ngsw.json` y el manifest), así que no requiere cambios de servidor.

**Tech Stack:** Angular 20 (standalone, signals), `@angular/service-worker` (^20), Go (sin cambios).

## Global Constraints

- No se agregan dependencias fuera de `@angular/service-worker` (ya alineado con Angular 20). Si el plan usa `ng add`, aceptar las modificaciones que genere.
- Copy en español rioplatense con acentos (`Estás sin conexión`, `Volver a intentar`, `Instalar aplicación`).
- Verificación frontend: `npm run build` (desde `frontend/`). Backend: `go build ./...` y `go test ./...` (desde `api/`) — solo si se tocó Go (no debería).
- El service worker requiere HTTPS o `localhost` para registrarse; en desarrollo con `ng serve` funciona porque sirve en `localhost`.
- El `ngsw-config.json` solo precachea el app shell (assets del build). Nunca cachear peticiones POST ni rutas autenticadas por el service worker a nivel de red; el token JWT NO debe persistirse en cache (los dataGroups no cachean cabeceras `Authorization` de forma predeterminada y las rutas `POST` quedan fuera).
- Viewports a verificar: 1366x768 y 390x844.

---
### Task 1: Instalar @angular/service-worker y habilitar el service worker en el build

**Files:**
- Modify: `frontend/package.json` (dependencia nueva)
- Modify: `frontend/angular.json` (build options: `serviceWorker: true`)
- Modify: `frontend/src/main.ts` (registro manual del SW)

**Interfaces:**
- Consumes: nada. Produce: `ngsw-worker.js` generado por el build en `dist/browser/` y `ngsw.json` en la raíz de `dist/browser/`.

- [ ] **Step 1: Instalar la dependencia**

Run (desde `frontend/`): `npm install @angular/service-worker@^20.0.0`
Expected: `@angular/service-worker` aparece en `package.json` dependencies.

- [ ] **Step 2: Habilitar el service worker en el builder**

En `frontend/angular.json`, en `projects.matematicarag-frontend.architect.build.options` agregar (después de `"outputHashing": "all"` no; en `options`, junto a `"outputPath"`):

```json
"options": {
  "outputPath": "dist/browser",
  "index": "src/index.html",
  "browser": "src/main.ts",
  "polyfills": ["zone.js"],
  "tsConfig": "tsconfig.app.json",
  "assets": ["src/favicon.ico", "src/assets", "src/manifest.webmanifest"],
  "styles": ["node_modules/@angular/material/prebuilt-themes/indigo-pink.css", "src/styles.scss"],
  "scripts": [],
  "serviceWorker": true,
  "ngswConfigPath": "ngsw-config.json"
},
```

- [ ] **Step 3: Registrar el ServiceWorker en `main.ts`**

Modificar `frontend/src/main.ts` para que el bootstrap registre el worker después de bootstrap, solo en producción:

```typescript
import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';
import { AppComponent } from './app/app.component';

bootstrapApplication(AppComponent, appConfig)
  .then(() => {
    if (location.protocol === 'https:' || location.hostname === 'localhost') {
      if ('serviceWorker' in navigator) {
        navigator.serviceWorker.register('ngsw-worker.js').catch(err => {
          console.warn('Service worker registration failed', err);
        });
      }
    }
  })
  .catch(err => console.error(err));
```

- [ ] **Step 4: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso; `dist/browser/ngsw-worker.js` y `dist/browser/ngsw.json` existen (verificar con `ls dist/browser/ngsw*`).

- [ ] **Step 5: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/angular.json frontend/src/main.ts
git commit -m "feat(pwa): add service worker registration and build integration"
```

### Task 2: Configuración ngsw-config.json — precache del app shell + dataGroups

**Files:**
- Create: `frontend/ngsw-config.json`

**Interfaces:**
- Consumes: `dist/browser` (archivos con hash generados por Angular CLI). Produce: `ngsw.json` con la política de cache.
- Depende de: Task 1 (build habilitado). Task 3 la consume para registrar el worker.

- [ ] **Step 1: Crear el archivo de configuración**

```json
{
  "$schema": "./node_modules/@angular/service-worker/config/schema.json",
  "index": "/index.html",
  "assetGroups": [
    {
      "name": "app-shell",
      "installMode": "prefetch",
      "updateMode": "prefetch",
      "resources": {
        "files": ["/favicon.ico", "/index.html", "/manifest.webmanifest", "/*.css", "/*.js", "/*.woff2", "/*.ttf", "/*.svg"]
      }
    },
    {
      "name": "assets",
      "installMode": "lazy",
      "updateMode": "prefetch",
      "resources": {
        "files": ["/assets/**", "/icons/**"]
      }
    }
  ],
  "dataGroups": [
    {
      "name": "api-get-readonly",
      "urls": ["/api/documents", "/api/learning/profile", "/api/learning/progress", "/api/dashboard/student", "/api/history", "/api/analytics"],
      "cacheConfig": {
        "strategy": "freshness",
        "maxSize": 50,
        "maxAge": "12h",
        "timeout": "10s"
      }
    }
  ]
}
```

- [ ] **Step 2: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso; `ngsw.json` generado con los assetGroups y dataGroups.

- [ ] **Step 3: Commit**

```bash
git add frontend/ngsw-config.json
git commit -m "feat(pwa): add ngsw config with app-shell precache and GET data groups"
```

### Task 3: OfflineService — detección de conexión + señal de reintento

**Files:**
- Create: `frontend/src/app/core/services/offline.service.ts`

**Interfaces:**
- Consumes: nada (usuario vía `localStorage`). Produce:
  - `isOffline: Signal<boolean>` — `true` cuando `navigator.onLine === false` o un request falló con error de red.
  - `networkBack: () => void` — método que el banner llama para reintentar (recarga o re-dispatch de cola).
  - Se inyecta como `OfflineService` (`providedIn: 'root'`).

- [ ] **Step 1: Crear el servicio**

```typescript
import { Injectable, signal, computed } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class OfflineService {
  private offlineSignal = signal(typeof navigator !== 'undefined' ? !navigator.onLine : false);
  readonly isOffline = computed(() => this.offlineSignal());

  constructor() {
    if (typeof window === 'undefined') return;
    window.addEventListener('online', () => {
      this.offlineSignal.set(false);
      this.retry();
    });
    window.addEventListener('offline', () => this.offlineSignal.set(true));
  }

  markOffline(): void {
    this.offlineSignal.set(true);
  }

  retry(): void {
    this.offlineSignal.set(false);
    window.location.reload();
  }
}
```

- [ ] **Step 2: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/core/services/offline.service.ts
git commit -m "feat(pwa): add offline detection service"
```

### Task 4: Banner de modo offline en el Layout

**Files:**
- Modify: `frontend/src/app/shared/layout.component.ts`

**Interfaces:**
- Consumes: `OfflineService` (de Task 3). Produce: banner visible al perder conexión, botón "Volver a intentar".

- [ ] **Step 1: Inyectar el servicio y agregar el banner**

En `layout.component.ts`:
- Agregar import: `import { OfflineService } from '../core/services/offline.service';`
- Agregar al constructor: `constructor(public auth: AuthService, public themeService: ThemeService, public offline: OfflineService) {}`
- Agregar en el template, justo después de `<div class="layout">` (línea 14), el banner:

```html
@if (offline.isOffline()) {
  <div class="offline-banner" role="status">
    <mat-icon>cloud_off</mat-icon>
    <span>Estás sin conexión. Algunas funciones pueden no estar disponibles.</span>
    <button mat-button class="offline-retry" (click)="offline.retry()">Volver a intentar</button>
  </div>
}
```

- Agregar estilos al array `styles` del componente:

```css
.offline-banner {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1300;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border);
  color: var(--text);
  font-size: 0.8rem;
  box-shadow: var(--shadow-md);
}
.offline-banner mat-icon { color: var(--warn); flex-shrink: 0; }
.offline-retry { margin-left: auto; }
```

- [ ] **Step 2: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/shared/layout.component.ts
git commit -m "feat(pwa): show offline banner with retry in layout"
```

### Task 5: Manifest webmanifest + meta tags + iconos

**Files:**
- Create: `frontend/src/manifest.webmanifest`
- Create: `frontend/src/icons/icon-192x192.png` y `frontend/src/icons/icon-512x512.png` (o reutilizar `src/favicon.ico`; si no hay PNG, generar dos PNG de 192 y 512 con el logo MateRAG)
- Modify: `frontend/src/index.html`

**Interfaces:**
- Consumes: Task 1 (assets incluyen `src/manifest.webmanifest`). Produce: instalabilidad (add to home screen).

- [ ] **Step 1: Crear el manifest**

```json
{
  "name": "MateRAG - Tutor Inteligente de Matemática",
  "short_name": "MateRAG",
  "description": "Tutor inteligente de matemática universitaria con IA, RAG y motor simbólico.",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#0f1115",
  "theme_color": "#4f8cff",
  "orientation": "portrait-primary",
  "icons": [
    { "src": "icons/icon-192x192.png", "sizes": "192x192", "type": "image/png", "purpose": "any maskable" },
    { "src": "icons/icon-512x512.png", "sizes": "512x512", "type": "image/png", "purpose": "any maskable" }
  ]
}
```

- [ ] **Step 2: Generar iconos**

Run (desde `frontend/`): si existe `src/favicon.ico`, generar PNGs con ImageMagick si está disponible, o crear dos PNG placeholder:
```
convert src/favicon.ico -resize 192x192 src/icons/icon-192x192.png
convert src/favicon.ico -resize 512x512 src/icons/icon-512x512.png
```
Si `convert` no está disponible: crear `src/icons/` y colocar dos PNG de 192x192 y 512x512 (puede ser un PNG sólido con el color de acento; la calidad del icono es un to-do visual separado).

- [ ] **Step 3: Actualizar `index.html`**

Agregar dentro de `<head>` (después de la línea 8, el favicon):

```html
<meta name="theme-color" content="#4f8cff">
<link rel="manifest" href="manifest.webmanifest">
<link rel="apple-touch-icon" href="icons/icon-192x192.png">
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="default">
```

- [ ] **Step 4: Verificar build**

Run: `npm run build` (desde `frontend/`)
Expected: build exitoso; `dist/browser/manifest.webmanifest` y `dist/browser/icons/*` existen.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/manifest.webmanifest frontend/src/icons frontend/src/index.html
git commit -m "feat(pwa): add web manifest, meta tags and icons for installability"
```

### Task 6: Verificación end-to-end del PWA

- [ ] **Step 1: Verificar build y artefactos**

Run: `npm run build && ls dist/browser/ngsw.json dist/browser/ngsw-worker.js dist/browser/manifest.webmanifest`
Expected: los 3 archivos existen.

- [ ] **Step 2: Servir y probar en localhost**

Run: `npm run start` (desde `frontend/`) y abrir `http://localhost:4200`.
Expected: en DevTools → Application → Service Workers, el worker está registrado y "activated". En Application → Manifest se ve el manifest válido.

- [ ] **Step 3: Probar offline**

En DevTools → Network, marcar "Offline". Recargar.
Expected: el app shell carga desde cache (logo MateRAG y login visible), el banner "Estás sin conexión" aparece. Desmarcar Offline y pulsar "Volver a intentar": la app recarga y el banner desaparece.

- [ ] **Step 4: Verificar que no se cachean datos autenticados**

Con el DevTools abierto en la pestaña Cache → Cache Storage → `ngsw:/` solo deben aparecer los archivos del app shell (no respuestas de `/api/chat` ni `POST`).

- [ ] **Step 5: Verificar backend intacto**

Run: `go build ./... && go test ./...` (desde `api/`)
Expected: build y tests OK (no debería haber cambios Go).
