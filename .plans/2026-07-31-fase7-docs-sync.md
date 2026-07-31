# Sincronización de Documentación (README) con Fases 5 y 6 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Actualizar el README para documentar el Agente Pedagógico (Fase 5) y el Motor de Aprendizaje Adaptativo (Fase 6) que ya están implementados en el código pero ausentes de la documentación.

**Architecture:** El código ya contiene los paquetes `api/agent/` (orquestador pedagógico con 8 tools), `api/adaptive/` (8 engines: mastery, knowledge map, error analyzer, prerequisites, difficulty, recommendation, learning path, events), los endpoints `/api/agent/*`, `/api/learning/*` adaptativos, `/api/dashboard/student` y `/api/dashboard/teacher`, analíticas y exportación. El README solo documenta hasta Fase 4. Este plan alinea Características, Estructura, API Endpoints, Variables de Entorno y Tests con la realidad del repo.

**Tech Stack:** Markdown, Go, Angular.

## Global Constraints

- No tocar código fuente: este plan solo modifica `README.md`. El `manual-usuario.md` ya está al día y NO se toca.
- Copy en español rioplatense con acentos (`Agente Pedagógico`, `Motor de Aprendizaje Adaptativo`, `Analíticas`, `Auditoría`, `Configuración`).
- Toda ruta citada debe existir y verificarse contra `cmd/server/main.go:91-142` y los archivos de rutas.
- Verificación: `go build ./...` NO es aplicable (no se toca Go); verificar que las rutas documentadas existen con `grep` y que el README renderiza sin bloques de código rotos.

---
### Task 1: Agregar secciones de características Fase 5 y Fase 6 al README

**Files:**
- Modify: `README.md:55-69` (insertar entre `### Fase 4` y `### Generales`)

**Interfaces:**
- Consumes: nada. Produce: documentación de características.
- Realidad verificada: Fase 5 = `api/agent/` (tool registry con RAG, math, verify, student, exercise, grading, hint + decision engine, planner, citation manager, response generator, learning updater, audit logger, intent classifier, `POST /api/agent/chat`). Fase 6 = `api/adaptive/` (8 engines) + endpoints en `api/learning.go:50-208`.

- [ ] **Step 1: Insertar bloque de Fase 5**

Insertar después de la línea `### Fase 4 — Sistema de Evaluacion y Calificacion Inteligente` (bloque de bullets de Fase 4), antes de `### Generales`:

```markdown
### Fase 5 — Agente Pedagógico
- **Orquestador de agente**: agente con loop de tools que analiza el problema y ejecuta herramientas en cadena
- **Registro de 8 herramientas**: RAG, math, verify, student, exercise, grading, hint, + eval
- **Clasificador de intenciones**: detecta resolver, verificar, pedir pista, explicar error, evaluar
- **Decision engine**: planifica la secuencia de tools según la intención y el contexto del alumno
- **Citation manager**: respuestas con citas a fuentes académicas recuperadas por RAG
- **Response generator**: genera la respuesta final consolidando resultados de las tools
- **Learning updater**: actualiza mastery y events de aprendizaje tras cada interacción
- **Hint tool**: hasta 3 pistas progresivas por ejercicio
- **Evaluate tool**: evaluación de ejercicios con el motor adaptativo
- **Guardrails**: max tool calls, max retries, umbral de intención configurables
- **Audit logger**: registra cada ejecución del agente en `agent_execution_log`
- **Chat frontend**: componente `agent-chat` con steps, citas y display de aprendizaje
```

- [ ] **Step 2: Insertar bloque de Fase 6**

```markdown
### Fase 6 — Motor de Aprendizaje Adaptativo
- **Knowledge Map**: arbol de conceptos con prerrequisitos y niveles
- **Mastery Engine**: cálculo de dominio por concepto con evidencia, recency weighting y status mapping (not_started, learning, developing, mastered)
- **Error Analyzer**: clasifica errores en 10 categorías, detecta recurrencia y severidad
- **Prerequisite Engine**: analiza prerrequisitos faltantes y recomienda remediación
- **Difficulty Engine**: selecciona dificultad 1-5 según mastery, con clamp adaptativo
- **Recommendation Engine**: genera recomendaciones personalizadas con explicación
- **Learning Path Engine**: construye rutas de aprendizaje y sugiere el siguiente paso
- **Learning Event Service**: pipeline completo de eventos (record, process, recent)
- **Learner State Loader**: carga estado completo del alumno y progreso por concepto
- **Búsqueda Qdrant adaptativa**: parámetros de búsqueda ajustados al estado del alumno
- **Integración con el Agente**: el agente consulta el motor adaptativo para personalizar
- **Analíticas de curso**: matrices de competencias y errores comunes para el docente
- **Dashboard adaptativo frontend**: componente `adaptive-dashboard` con mapa de dominio
```

- [ ] **Step 3: Verificar inserción**

Run: `grep -n 'Fase 5\|Fase 6' README.md`
Expected: ambas líneas presentes, con `### Fase 5 — Agente Pedagógico` y `### Fase 6 — Motor de Aprendizaje Adaptativo`.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add Fase 5 and Fase 6 feature sections to README"
```

### Task 2: Actualizar la sección Estructura con los paquetes adaptativos

**Files:**
- Modify: `README.md:85-183` (sección `## Estructura`)

**Interfaces:**
- Consumes: estructura real de `api/` (flat files + `api/adaptive/` + `api/agent/`).
- Produce: árbol actualizado.

- [ ] **Step 1: Verificar el árbol real**

Run: `ls api/ && ls api/adaptive/ && ls api/agent/`
Expected: los archivos flat de `api/` (`auth.go`, `chat.go`, ...), `api/adaptive/*.go` (engine, mastery, knowledge_map, errors, prerequisites, difficulty, recommendations, learning_path, events, state, analytics, qdrant), `api/agent/*.go` (agent, decision_engine, planner, tool_registry, rag_tool, math_tool, verify_tool, student_tool, exercise_tool, grading_tool, hint_tool, evaluate_tool, citation_manager, response_generator, learning_updater, audit_logger, intent_classifier, context_manager, state).

- [ ] **Step 2: Reemplazar el bloque de árbol de `api/`**

En `## Estructura`, reemplazar el bloque actual del árbol `api/` por:

```text
├── api/
│   ├── auth.go                     # Register, Login, Refresh Token
│   ├── chat.go                     # Chat con integracion RAG automatica
│   ├── documents.go                # Subida, listado y borrado de documentos
│   ├── rag.go / qdrant.go          # RAG hibrido + embeddings + Qdrant
│   ├── math.go / mathclient.go     # Proxy al math-service (SymPy)
│   ├── math_evaluator.go           # Pipeline de evaluacion matematica estructurada
│   ├── tutor.go                    # POST /api/tutor/solve (orquestador Fase 2)
│   ├── intent.go                   # Clasificador de intencion
│   ├── sessions.go                 # Sesiones tutor adaptativo
│   ├── exercises.go                # Banco de ejercicios + generacion adaptativa
│   ├── learning.go                 # Endpoints /api/learning/* (perfil, mastery, eventos, rutas)
│   ├── student.go / teacher.go     # Dashboards de estudiante y profesor
│   ├── dashboards.go               # /api/dashboard/student y /api/dashboard/teacher
│   ├── assessments.go              # Evaluaciones (Fase 4)
│   ├── questions.go / grading.go   # Banco de preguntas + calificacion
│   ├── adaptive_assessment.go      # Evaluaciones adaptativas
│   ├── analytics.go / analytics_v2.go  # Analiticas
│   ├── critical_concepts.go        # Analitica de conceptos criticos
│   ├── recovery.go                 # Planes de recuperacion
│   ├── alerts.go                   # Alertas academicas
│   ├── export.go                   # Exportacion CSV
│   ├── audit.go                    # Auditoria
│   ├── agent_routes.go             # POST /api/agent/chat
│   ├── settings.go / stats.go      # Panel admin
│   ├── middleware.go / circuit_breaker.go / metrics.go   # Resiliencia y observabilidad
│   ├── adaptive/                   # Motor de Aprendizaje Adaptativo (Fase 6)
│   │   ├── engine.go               # Core + wiring de los 8 engines
│   │   ├── mastery.go              # Mastery engine
│   │   ├── knowledge_map.go        # Knowledge map + prerrequisitos
│   │   ├── errors.go               # Error analyzer
│   │   ├── prerequisites.go        # Prerequisite engine
│   │   ├── difficulty.go           # Difficulty engine
│   │   ├── recommendations.go      # Recommendation engine
│   │   ├── learning_path.go        # Learning path engine
│   │   ├── events.go               # Learning event service
│   │   ├── state.go                # Learner state loader
│   │   ├── analytics.go            # Progreso y analiticas de curso
│   │   └── qdrant.go               # Búsqueda adaptativa en Qdrant
│   └── agent/                      # Agente Pedagogico (Fase 5)
│       ├── agent.go                # Orquestador + loop de tools
│       ├── tool_registry.go        # Registro de 8 herramientas
│       ├── decision_engine.go      # Planificacion de secuencia de tools
│       ├── planner.go              # Plan de ejecucion
│       ├── rag_tool.go / math_tool.go / verify_tool.go / student_tool.go
│       ├── exercise_tool.go / grading_tool.go / hint_tool.go / evaluate_tool.go
│       ├── citation_manager.go     # Citas a fuentes
│       ├── response_generator.go   # Generacion de respuesta final
│       ├── learning_updater.go     # Actualizacion de mastery y eventos
│       ├── audit_logger.go         # Registro de ejecuciones
│       └── intent_classifier.go    # Clasificacion de intencion del agente
```

- [ ] **Step 3: Verificar**

Run: `grep -c 'adaptive/\|agent/' README.md`
Expected: al menos 4 coincidencias (bloque de estructura).

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: update structure section with adaptive and agent packages"
```

### Task 3: Agregar tablas de API Endpoints faltantes

**Files:**
- Modify: `README.md:401-629` (sección `## API Endpoints`)

**Interfaces:**
- Consumes: rutas verificadas en `api/agent_routes.go:66`, `api/learning.go:50-208`, `api/dashboards.go:109-115`, `api/recovery.go:31-79`, `api/export.go:15-17`, `api/alerts.go`, `api/critical_concepts.go`, `api/analytics_v2.go`.
- Produce: tablas completas.

- [ ] **Step 1: Verificar rutas reales**

Run: `grep -n 'r\.Post\|r\.Get\|r\.Put\|r\.Delete' api/agent_routes.go api/learning.go api/alerts.go | head -40`
Expected: `POST /api/agent/chat`, `GET /api/learning/profile|progress|mastery|errors|learner-profile|recommendation|path|course-analytics|errors/common`, `POST /api/learning/events`, `POST /api/learning/suggest`, `GET /api/alerts`, `PUT /api/alerts/{alertID}/acknowledge`, `POST /api/alerts/check`.

- [ ] **Step 2: Insertar tabla Agente Pedagógico**

Insertar después de la tabla `### Analytics` (fin de la sección API Endpoints):

```markdown
### Agente Pedagógico (requiere auth)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/agent/chat` | Procesa un mensaje con el agente: planifica tools, ejecuta, genera respuesta con citas |
| GET | `/api/agent/sessions/{id}` | Recupera una sesion del agente |

### Learning Adaptativo (requiere auth)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/learning/profile` | Perfil del estudiante |
| GET | `/api/learning/progress` | Progreso global y por concepto |
| GET | `/api/learning/mastery` | Mapa de dominio por concepto |
| GET | `/api/learning/errors` | Errores del estudiante |
| POST | `/api/learning/events` | Registra un evento de aprendizaje |
| GET | `/api/learning/learner-profile` | Estado completo del alumno (learner state) |
| GET | `/api/learning/recommendation` | Recomendacion personalizada |
| GET | `/api/learning/path` | Ruta de aprendizaje construida |
| POST | `/api/learning/suggest` | Registrar sugerencia pedagógica |
| GET | `/api/learning/course-analytics` | Analiticas del curso (docentes) |
| GET | `/api/learning/errors/common` | Errores comunes del curso (docentes) |

### Dashboards (requiere auth)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/dashboard/student` | Dashboard del estudiante: progreso, actividad, stats |
| GET | `/api/dashboard/teacher` | Dashboard del profesor: overview del curso (TEACHER/ADMIN) |

### Alertas Academicas (requiere auth)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/alerts` | Alertas del estudiante (filtrables por severity) |
| PUT | `/api/alerts/{alertID}/acknowledge` | Marcar alerta como reconocida |
| POST | `/api/alerts/check` | Verifica y genera alertas academicas |

### Recuperacion (requiere auth)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/recovery` | Crear plan de recuperacion |
| GET | `/api/recovery` | Planes del estudiante |
| PUT | `/api/recovery/{planID}/complete` | Completar plan |
| PUT | `/api/recovery/{planID}/cancel` | Cancelar plan |

### Exportacion (requiere auth)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/export/assessment/{assessmentID}/csv` | Exportar evaluacion a CSV |
| GET | `/api/export/student/{studentID}/csv` | Exportar resultados del estudiante |
| GET | `/api/export/course/{courseID}/csv` | Exportar curso a CSV |

### Conceptos Criticos (requiere TEACHER/ADMIN)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/teacher/critical-concepts` | Conceptos criticos del curso (por debajo del umbral) |

### Metricas (publico)
| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/metrics` | Metricas Prometheus |
| GET | `/health` | Health check con estado de db, math-service y qdrant |
```

- [ ] **Step 3: Verificar**

Run: `grep -c 'api/agent/chat\|api/learning/recommendation\|api/alerts/check\|api/export/assessment' README.md`
Expected: ≥ 4 coincidencias.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add agent, adaptive learning, alerts, recovery, export, metrics endpoint tables"
```

### Task 4: Actualizar Variables de Entorno y Tests

**Files:**
- Modify: `README.md:630-682` (Variables de Entorno) y `README.md:842-864` (Tests)

**Interfaces:**
- Consumes: `internal/config/config.go` (campos adaptativos y del agente) y `api/adaptive/adaptive_test.go`, `api/agent/agent_test.go`.
- Produce: secciones actualizadas.

- [ ] **Step 1: Agregar env vars del motor adaptativo y agente**

Agregar al bloque de env vars del Backend:

```markdown
### Motor Adaptativo y Agente
| Variable | Default | Descripcion |
|----------|---------|-------------|
| MASTERY_OLD_WEIGHT | `0.30` | Peso del mastery previo |
| MASTERY_EVIDENCE_WEIGHT | `0.60` | Peso de la evidencia nueva |
| MASTERY_HINT_PENALTY | `0.10` | Penalizacion por usar pista |
| MASTERY_ERROR_PENALTY | `0.15` | Penalizacion por error |
| MASTERY_RECENCY_FACTOR | `0.80` | Factor de recencia |
| LEARNING_CRITICAL_THRESHOLD | `0.35` | Umbral de concepto critico |
| LEARNING_BEGINNER_THRESHOLD | `0.40` | Umbral de nivel beginner |
| LEARNING_DEVELOPING_THRESHOLD | `0.60` | Umbral de nivel developing |
| LEARNING_COMPETENT_THRESHOLD | `0.80` | Umbral de nivel competent |
| ADAPTIVE_QDRANT_TOP_K | `5` | Top-K en búsqueda adaptativa |
| AGENT_MAX_TOOL_CALLS | `8` | Max tool calls por turno del agente |
| AGENT_MAX_RETRIES | `2` | Reintentos del agente |
| AGENT_INTENT_THRESHOLD | `0.6` | Umbral del clasificador de intencion |
| AGENT_LOW_MASTERY | `0.4` | Mastery bajo (personalizar a este alumno) |
| AGENT_HIGH_MASTERY | `0.8` | Mastery alto |
| AGENT_MANUAL_REVIEW_THRESH | `0.5` | Umbral para revision manual |
```

- [ ] **Step 2: Actualizar la sección Tests**

En `## Tests`, agregar:

```markdown
### Adaptive Engine (Go)
Run: `cd api && go test ./adaptive/ -v`
Cobertura: mastery engine, difficulty, prerequisites, recommendations, learning path, events.

### Agent (Go)
Run: `cd api && go test ./agent/ -v`
Cobertura: decision engine, planner, tool registry, response generator.
```

- [ ] **Step 3: Verificar**

Run: `grep -c 'MASTERY_EVIDENCE_WEIGHT\|AGENT_MAX_TOOL_CALLS\|Adaptive Engine (Go)' README.md`
Expected: ≥ 3 coincidencias.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: document adaptive/agent env vars and test coverage"
```

### Task 5: Verificación final

- [ ] **Step 1: Chequear que las rutas documentadas existen en el código**

Run: `cd /home/proyecto/matematicarag && grep -rn 'agent/chat\|learning/profile\|dashboard/student\|alerts/check\|recovery/complete\|critical-concepts' cmd/server/main.go api/ | head -20`
Expected: cada ruta citada en README aparece en `cmd/server/main.go` o en los archivos de rutas.

- [ ] **Step 2: Verificar que no se toca código**

Run: `git status`
Expected: solo `README.md` modificado.

- [ ] **Step 3: Verificar que el README renderiza (sin corchetes rotos en las tablas)**

Run: `grep -c '|' README.md`
Expected: decenas de coincidencias; abrir el README y revisar que las 4 tablas nuevas tienen encabezados `| Metodo | Ruta | Descripcion |`.
