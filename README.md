# MatematicaRAG

Tutor de matematicas universitarias con Inteligencia Artificial, RAG (Retrieval-Augmented Generation), motor matematico symbolic y aprendizaje adaptativo. Resuelve, explica y verifica ejercicios paso a paso con citations academicas, genera ejercicios personalizados, detecta patrones de error y adapta la dificultad al nivel del estudiante.

## Caracteristicas

### Fase 1 — RAG Academico
- **Chat con IA** integrado con RAG hibrido (vector + keyword): el LLM cita automaticamente las fuentes utilizadas
- **Source attribution**: cada respuesta incluye chips clickeables con nombre del documento, pagina y seccion
- **Hybrid search**: fusion ponderada de busqueda vectorial (Qdrant) + busqueda por palabras clave (PostgreSQL tsvector)
- **Reranking**: re-evaluacion de relevancia con LLM para mejorar precision
- **Busqueda de texto**: PostgreSQL full-text search con filtros por documento, curso, unidad, tema
- **Citations estructuradas**: formato [SRC-XXX] con score de confianza

### Fase 2 — Motor Matematico
- **Tutor paso a paso**: resuelve ejercicios con explicacion didactica y verificacion automatica
- **Motor SymPy**: calculo symbolic confiable (derivadas, integrales, limites, ecuaciones, matrices)
- **Clasificador de intencion**: identifica si el alumno necesita resolver, verificar, explicar, etc.
- **Verificacion automatica**: cada resultado matematico se verifica algebraicamente
- **Modos de uso**: resolver, verificar, pista, explicar error
- **Nivel de explicacion**: basico, intermedio, avanzado
- **Renderizado LaTeX**: formulas con KaTeX en el frontend
- **Separacion fuentes/calculos**: distingue entre informacion recuperada (fuentes) y calculos realizados (motor)

### Fase 3 — Tutor Adaptativo
- **Sesiones adaptativas**: crear sesiones en modos tutor, practica, repaso o examen
- **Generacion de ejercicios**: LLM genera ejercicios originales validados por SymPy
- **Correccion paso a paso**: analiza cada paso del procedimiento del alumno
- **Sistema de pistas**: hasta 3 pistas progresivas por ejercicio
- **Banco de ejercicios**: repositorio validado por concepto y dificultad
- **Motor adaptativo**: selecciona siguiente concepto y dificultad basado en mastery del alumno
- **Grafo de conocimiento**: arbol de conceptos con prerrequisitos (algebra, funciones, limites, derivadas, integrales)
- **Tracking de mastery**: nivel por concepto (not_started, learning, developing, mastered)
- **Taxonomia de errores**: clasificacion en 10 categorias (conceptual, algebraico, aritmetico, signo, formula, metodo, notacion, dominio, logico, incompleto)
- **Deteccion de patrones**: identifica errores recurrentes por estudiante
- **Dashboard del estudiante**: progreso, dominio por concepto, stats, recomendaciones
- **Dashboard del profesor**: overview del curso, mastery por tema, errores comunes, progreso individual
- **Configuracion adaptativa**: pesos de hints, errors, threshold de mastery configurables via env

### Fase 4 — Sistema de Evaluacion y Calificacion Inteligente
- **Evaluaciones multiples**: diagnosticas, formativas, sumativas, de recuperacion y practica
- **Modos de evaluacion**: fija (profesor selecciona), generada (IA crea con RAG), adaptativa (se ajusta al rendimiento)
- **Calificacion inteligente**: validacion matematica + reglas + rbricas + LLM cuando es necesario
- **Puntuacion parcial**: soporte para evaluacion paso a paso con deteccion de errores
- **Rubricas de evaluacion**: rbricas analiticas y holsticas con multiples criterios
- **Calificacion en lote**: calificacion automatica masiva de evaluaciones
- **Banco de preguntas**: repositorio de preguntas con tipos exercise, multiple_choice, true_false, numeric, algebraic, equation, step_by_step
- **Validacion matematica**: todas las preguntas validadas por Math Engine antes de almacenar
- **Guardado automatico**: autosave cada 30 segundos durante evaluacion
- **Recuperacion de sesion**: resume donde quedaste tras conexion
- **Evaluaciones adaptativas**: dificultad se ajusta automaticamente al rendimiento (niveles 1-5)
- **Exportacion**: CSV para evaluaciones, estudiantes y cursos
- **Auditoria**: registro de cambios con versionado (old/new values)
- **Analiticas de estudiantes**: nivel de competencia, tendencias de rendimiento, conceptos dbiles/fuertes
- **Matriz de competencias**: success rate por concepto para todo el curso
- **Patrones de errores colectivos**: detecta errores recurrentes en todos los estudiantes
- **Planes de recuperacion**: recomendaciones personalizadas basadas en rendimiento
- **Alertas academicas**: sistema de temprana deteccion de estudiantes en riesgo
- **Configuracion**: time limit, max attempts, passing score, auto-grade, recovery threshold, alert threshold

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

### Generales
- **Gestion documental**: subir PDF/DOCX/TXT/MD, indexacion automatica en base vectorial
- **Extraccion de metadata**: paginas (PDF), secciones (heuristica), tipos de contenido (definicion, teorema, formula, ejemplo)
- **Multi-proveedor IA**: OpenAI, Anthropic (Claude), Groq, OpenRouter
- **Responsive**: mobile-first con hamburger menu, funciona en 375px+
- **Dark/Light theme**: toggle en el sidebar
- **Auth JWT**: registro, login, refresh token, roles (Admin, Profesor, Alumno)
- **Panel admin**: estadisticas, gestion de usuarios, configuracion de prompts y API keys

## Stack

| Capa | Tecnologia |
|------|-----------|
| Backend | Go 1.25 + Chi v5 |
| Math Engine | Python 3.11 + SymPy + Flask |
| Frontend | Angular 20 + Material Design |
| Database | PostgreSQL 16 |
| Vector DB | Qdrant v1.12 |
| Embeddings | OpenAI text-embedding-3-small (1536 dim) |
| LaTeX | KaTeX + MathLive |
| IA | OpenAI GPT-4 / Anthropic Claude / Groq Llama / OpenRouter |
| Auth | JWT (HS256) + bcrypt |
| Deploy | Docker + Dokploy |

## Estructura

```
matematicarag/
├── cmd/
│   ├── server/main.go              # Entry point + routing
│   ├── migrate/main.go             # Migracion de documentos existentes
│   └── ragtest/main.go             # Tests de evaluacion RAG
├── api/
│   ├── auth.go                     # Register, Login, Refresh Token
│   ├── chat.go                     # Chat con integracion RAG automatica
│   ├── documents.go                # Subida, listado y borrado de documentos
│   ├── rag.go / qdrant.go          # RAG hibrido + embeddings + Qdrant
│   ├── openai.go                   # Multi-proveedor LLM
│   ├── math.go / mathclient.go     # Proxy al math-service (SymPy)
│   ├── math_evaluator.go           # Pipeline de evaluacion matematica estructurada
│   ├── tutor.go                    # POST /api/tutor/solve (orquestador Fase 2)
│   ├── intent.go                   # Clasificador de intencion
│   ├── sessions.go                 # Sesiones tutor adaptativo
│   ├── exercises.go                # Banco de ejercicios + generacion adaptativa
│   ├── adaptive.go                 # Recomendacion de siguiente concepto + dificultad (legacy)
│   ├── learning.go                 # Endpoints /api/learning/* (perfil, mastery, eventos, rutas)
│   ├── knowledge.go                # Grafo de conceptos + prerrequisitos
│   ├── student.go / teacher.go     # Dashboards de estudiante y profesor
│   ├── users.go                    # Gestion de usuarios
│   ├── history.go                  # Historial de sesiones
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
│   ├── indexer.go                  # Re-indexacion (ADMIN)
│   ├── migration.go                # Helpers para migracion de datos
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
├── math-service/
│   ├── app.py                      # Flask API (12 endpoints matematicos)
│   ├── Dockerfile                  # Python 3.11-slim + SymPy
│   ├── requirements.txt            # flask, sympy, gunicorn
│   ├── engine/
│   │   ├── parser.py               # Normalizacion de entrada + parsing seguro
│   │   ├── calculus.py             # Derivadas, integrales, limites
│   │   ├── equations.py            # Resolucion de ecuaciones y sistemas
│   │   ├── algebra.py              # Simplificar, factorizar, expandir
│   │   ├── matrices.py             # Determinante, inversa, rango, eigenvalores
│   │   ├── verify.py               # Verificacion symbolic de resultados
│   │   ├── arithmetic.py           # Evaluacion basica
│   │   └── trigonometry.py         # Simplificacion trigonometrica
│   └── tests/
│       ├── test_calculus.py        # Tests: derivadas, integrales, limites
│       ├── test_equations.py       # Tests: ecuaciones
│       ├── test_matrices.py        # Tests: matrices
│       └── test_verify.py          # Tests: verificacion
├── internal/
│   ├── config/config.go            # Variables de entorno (incluye adaptive engine)
│   ├── database/database.go        # PostgreSQL + 31 migraciones
│   └── middleware/middleware.go     # CORS + Rate Limiting
├── frontend/src/app/
│   ├── core/
│   │   ├── guards/                 # Auth + Role guards
│   │   ├── services/
│   │   │   ├── auth.service.ts     # Autenticacion JWT
│   │   │   ├── api.service.ts      # Cliente API (chat, tutor, math, docs)
│   │   │   ├── learning.service.ts # Cliente API de aprendizaje (sesiones, ejercicios, dashboards)
│   │   │   ├── assessment.service.ts # Cliente API de evaluaciones (CRUD, grading, analytics, recovery, alerts)
│   │   │   └── theme.service.ts    # Toggle dark/light
│   │   └── interceptors/           # HTTP interceptors (auth, errors)
│   ├── shared/
│   │   ├── layout.component.ts     # Sidebar + responsive layout
│   │   └── render-math.pipe.ts     # Pipe KaTeX para LaTeX
│   └── modules/
│       ├── auth/                   # Login + Register
│       ├── chat/                   # Chat con IA + fuentes clickeables
│       ├── tutor/                  # Tutor matematico (4 modos: resolver/tutor/practicar/repaso)
│       ├── math/                   # Calculadora matematica (MathLive)
│       ├── documents/              # Gestion de archivos + vista de chunks
│       ├── bdvectorial/            # Visor de docs en Qdrant
│       ├── history/                # Historial de sesiones
│       ├── student-progress/       # Dashboard del estudiante (mastery, stats, recomendaciones)
│       ├── teacher-dashboard/      # Dashboard del profesor (progreso, errores, estudiantes)
│       ├── assessment/              # Tomar evaluaciones (timer, navegacion, resultados)
│       ├── analytics/               # Panel de analiticas (competencia, recuperacion, alertas)
│       ├── dashboard/              # Dashboard admin (estadisticas)
│       ├── admin/                  # Panel administrativo
│       └── settings/               # Configuracion (prompts, API keys, IA)
├── scripts/
│   └── seed.go                     # Script de datos de ejemplo (usuarios + conceptos)
├── Dockerfile                      # Multi-stage (Node + Go + Alpine)
├── docker-compose.yml              # Qdrant + Backend + Frontend + Math Service
└── .env.example                    # Template de variables
```

## Arquitectura

```
Browser → Go Server (8008)
            │
            ├── /api/tutor/solve        → Orquestador del tutor (Fase 2)
            │   ├── ClassifyIntent (LLM: identifica tipo de problema)
            │   ├── HybridSearch (Qdrant vectorial + PostgreSQL keyword)
            │   ├── RerankResults (LLM: re-evalua relevancia)
            │   ├── MathClient → Python/SymPy (calculo symbolic)
            │   ├── LLM (genera explicacion paso a paso)
            │   └── Verification (verifica resultado matematico)
            │
            ├── /api/sessions/*          → Tutor adaptativo (Fase 3)
            │   ├── POST /session        → Crear sesion (tutor/practice/review/exam)
            │   ├── POST /answer         → Responder ejercicio (verifica + mastery)
            │   ├── POST /hint           → Solicitar pista progresiva
            │   └── POST /feedback       → Analizar procedimiento paso a paso
            │
            ├── /api/exercises/*         → Banco de ejercicios (Fase 3)
            │   ├── POST /generate       → Generar ejercicio (LLM + validacion SymPy)
            │   ├── GET /concept/{id}    → Ejercicio por concepto
            │   └── GET /{id}/hints      → Pistas de un ejercicio
            │
            ├── /api/learning/*          → Knowledge model (Fase 3)
            │   ├── GET /courses/{id}/concepts   → Arbol de conceptos
            │   ├── GET /courses/{id}/prerequisites → Prerrequisitos
            │   ├── GET /progress       → Perfil + mastery del estudiante
            │   ├── GET /mastery        → Mapa de dominio por concepto
            │   └── GET /errors         → Errores del estudiante
            │
            ├── /api/student/*           → Dashboard estudiante (Fase 3)
            │   ├── GET /my-progress    → Progreso completo
            │   └── GET /recommendations → Siguiente concepto recomendado
            │
            ├── /api/teacher/*           → Dashboard profesor (Fase 3, requiere TEACHER/ADMIN)
            │   ├── GET /course-progress    → Resumen del curso
            │   ├── GET /topic-mastery      → Dominio por tema
            │   ├── GET /common-errors      → Errores comunes
            │   └── GET /student-progress   → Progreso individual
            │
            ├── /api/assessments/*        → Evaluaciones (Fase 4)
            │   ├── POST /                → Crear evaluacion (profesor)
            │   ├── GET /                 → Listar evaluaciones
            │   ├── GET /{id}             → Obtener evaluacion con preguntas
            │   ├── PUT /{id}             → Actualizar evaluacion
            │   ├── DELETE /{id}          → Eliminar evaluacion
            │   ├── POST /{id}/publish    → Publicar evaluacion
            │   ├── POST /{id}/start      → Iniciar evaluacion (estudiante)
            │   ├── POST /{id}/submit     → Enviar respuestas
            │   ├── POST /{id}/autosave   → Guardar respuestas automaticamente
            │   ├── POST /{id}/resume     → Recuperar sesion de evaluacion
            │   ├── POST /{id}/adaptive-next → Siguiente pregunta adaptativa
            │   ├── GET /{id}/results     → Ver resultados
            │   └── GET /{id}/student-results → Resultados de todos los estudiantes
            │
            ├── /api/questions/*          → Banco de Preguntas (Fase 4)
            │   ├── POST /                → Crear pregunta
            │   ├── GET /                 → Listar preguntas (filtros: concepto, tipo, dificultad)
            │   ├── GET /{id}             → Obtener pregunta
            │   ├── PUT /{id}             → Actualizar pregunta
            │   ├── DELETE /{id}          → Soft delete (is_active=false)
            │   └── POST /validate/{id}   → Validar con Math Engine
            │
            ├── /api/grading/*            → Calificacion (Fase 4)
            │   ├── POST /answer/{id}     → Calificacion manual
            │   ├── POST /rubric/{id}     → Crear rbrica
            │   ├── GET /rubric/{id}      → Obtener rbricas
            │   ├── POST /evaluate/{id}   → Evaluar con rbrica
            │   └── POST /batch-grade/{id} → Calificacion automatica en lote
            │
            ├── /api/analytics/v2/*       → Analiticas (Fase 4)
            │   ├── GET /student/{id}     → Analiticas del estudiante
            │   ├── GET /course/{id}      → Analiticas del curso
            │   ├── GET /student/{id}/competency → Reporte de competencia
            │   ├── GET /student/{id}/trend → Tendencia de rendimiento
            │   ├── POST /student/{id}/update → Recalcular analiticas
            │   ├── GET /course/{id}/matrix → Matriz de competencias
            │   ├── GET /course/{id}/error-patterns → Patrones de errores colectivos
            │   └── GET /student/{id}/question-history → Historial de preguntas
            │
            ├── /api/recovery/*           → Planes de recuperacion (Fase 4)
            │   ├── POST /               → Crear plan
            │   ├── GET /                → Obtener planes activos
            │   ├── PUT /{id}/complete    → Marcar como completado
            │   └── PUT /{id}/cancel      → Cancelar plan
            │
            ├── /api/alerts/*            → Alertas academicas (Fase 4)
            │   ├── GET /                → Obtener alertas del estudiante
            │   ├── PUT /{id}/acknowledge → Reconocer alerta
            │   ├── POST /check          → Verificar y crear alertas
            │   └── GET /all             → Todas las alertas (profesor/admin)
            │
            ├── /api/export/*            → Exportacion (Fase 4)
            │   ├── GET /assessment/{id}/csv → Exportar resultados de evaluacion
            │   ├── GET /student/{id}/csv    → Exportar historial de estudiante
            │   └── GET /course/{id}/csv     → Exportar analiticas del curso
            │
            ├── /api/audit/*             → Auditoria (Fase 4)
            │   └── GET /{entityType}/{entityId} → Log de auditoria
            │
            ├── /api/chat               → Chat + RAG automatico
            ├── /api/rag                → Consultas RAG directas
            ├── /api/math               → Operaciones matematicas via SymPy
            ├── /api/documents          → Upload + chunking + embeddings
            ├── /api/indexer            → Re-indexacion (ADMIN)
            └── /*                      → Angular SPA (archivos estaticos)

Go Server → PostgreSQL (users, sessions, messages, documents, chunks, settings,
                        concepts, concept_prerequisites, student_profiles,
                        concept_mastery, exercises, tutor_sessions,
                        exercise_attempts, attempt_steps, student_errors,
                        learning_recommendations, assessments, assessment_questions,
                        rubrics, student_assessments, student_answers,
                        student_analytics, recovery_plans, academic_alerts,
                        question_bank, assessment_sessions, audit_log_entries)
Go Server → Qdrant (vectores con payload enriquecido)
Go Server → OpenAI API (GPT-4 + text-embedding-3-small)
Go Server → Python Math Service (SymPy symbolic computation)
```

## Pipeline RAG Hibrido

```
1. Upload:      Usuario sube PDF → extractTextWithPages() extrae texto por pagina
2. Chunking:    chunkTextWithMetadata() crea chunks configurables (default 500 chars)
                Detecta secciones, tipos de contenido (definicion, teorema, formula, ejemplo)
3. Embeddings:  generateEmbeddings() llama a OpenAI text-embedding-3-small
4. Storage:     qdrantUpsert() guarda chunk con payload enriquecido:
                {document_id, course_id, unit_id, topic, content_type, page, section, url}
5. Index:       PostgreSQL document_chunks con tsvector para busqueda full-text
6. Query:       HybridSearch combina:
                - Vector search (Qdrant): similaridad semantica
                - Keyword search (PostgreSQL): coincidencia lexica
                Fusion ponderada (default 60% vector / 40% keyword)
7. Rerank:      RerankResults re-evalua relevancia con LLM
8. Context:     Construye contexto con citations [SRC-XXX]:
                [SRC-001] algebra.pdf — pagina 5, seccion "Polinomios"
                contenido del chunk...
9. LLM:         El system prompt exige citas: [SRC-XXX]
10. Response:   Frontend renderiza chips clickeables con expand
```

## Pipeline del Tutor Adaptativo

```
Pregunta del alumno
        ↓
┌─ Modo RESOLVER (Fase 2) ─────────────────────┐
│  Clasificacion (LLM + keyword fallback)       │
│  RAG Hibrido (Qdrant + PostgreSQL)            │
│  Math Engine (SymPy via Python service)        │
│  Verificacion automatica                       │
│  LLM genera explicacion paso a paso           │
│  Citation Engine                               │
│  Respuesta estructurada                        │
└───────────────────────────────────────────────┘
        │
        ↓
┌─ Modos TUTOR/PRACTICA/REPASO (Fase 3) ───────┐
│  Crear sesion adaptativa                       │
│  RecomendNext (motor adaptativo)               │
│  ├─ Selecciona concepto por mastery            │
│  ├─ Calcula dificultad (mastery + errors)      │
│  └─ Verifica prerrequisitos                    │
│  GenerateExercise (LLM + validacion SymPy)      │
│  Presenta ejercicio al alumno                  │
│  Alumno responde                               │
│  SubmitAnswer:                                 │
│  ├─ MathClient.Verify (respuesta correcta?)    │
│  ├─ analyzeSteps (correccion paso a paso)      │
│  ├─ UpdateMastery (ajusta nivel)               │
│  ├─ RecordError (si hay error)                 │
│  └─ DetectPatterns (errores recurrentes)       │
│  Feedback + siguiente ejercicio                │
└───────────────────────────────────────────────┘
```

## Grafo de Conocimiento

```
matematica-1 (curso)
├── algebra
│   ├── algebra.operaciones      (dificultad 1)
│   ├── algebra.factorizacion    (dificultad 2)
│   └── algebra.ecuaciones       (dificultad 2)
├── funciones
│   ├── funciones.lineal         (dificultad 1)
│   ├── funciones.cuadratica     (dificultad 2)
│   └── funciones.composicion    (dificultad 3)
├── limites
│   ├── limites.concepto         (dificultad 2)
│   ├── limites.propiedades      (dificultad 3)
│   └── limites.laterales        (dificultad 3)
├── derivadas
│   ├── derivadas.definicion     (dificultad 3)
│   ├── derivadas.potencia       (dificultad 3)
│   ├── derivadas.producto       (dificultad 4)
│   ├── derivadas.cociente       (dificultad 4)
│   └── derivadas.cadena         (dificultad 4)
└── integrales
    ├── integrales.indefinida    (dificultad 4)
    ├── integrales.definida      (dificultad 4)
    ├── integrales.sustitucion   (dificultad 5)
    └── integrales.partes        (dificultad 5)

Prerrequisitos: algebra → funciones → limites → derivadas → integrales
```

## Usuarios de Ejemplo

| Rol | Email | Contrasena |
|-----|-------|-----------|
| Admin | admin@face-unt.ar | Admin123! |
| Profesor | profesor@face-unt.ar | Profesor123! |
| Alumno | alumno@face-unt.ar | Alumno123! |

## API Endpoints

### Health (publico)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/health` | Estado del servidor |
| GET | `/health/qdrant` | Estado de Qdrant |

### Auth (publico)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/auth/register` | Registrar usuario |
| POST | `/api/auth/login` | Iniciar sesion |
| POST | `/api/auth/refresh` | Renovar token |

### Tutor (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/tutor/solve` | Resolver problema matematico con tutor paso a paso |

Request:
```json
{
  "query": "Resolver x^2 + 5x + 6 = 0",
  "explanation_level": "intermediate",
  "mode": "solve"
}
```

Response:
```json
{
  "problem": {"type": "solve", "expression": "x^2+5*x+6=0", "variable": "x"},
  "method": {"name": "factorizacion", "description": "Buscamos dos numeros..."},
  "steps": [
    {"number": 1, "title": "Identificar", "explanation": "...", "latex": "x^2+5x+6", "is_math": false},
    {"number": 2, "title": "Factorizar", "explanation": "...", "latex": "(x+2)(x+3)", "is_math": true}
  ],
  "result": {"success": true, "result": "[-2, -3]", "latex": "x=-2, x=-3"},
  "verification": {"status": "verified", "method": "symbolic_solve"},
  "citations": [{"id": "SRC-001", "document": "algebra.pdf", "page": 35}],
  "sources": [...],
  "math_computed": true,
  "confidence": "high"
}
```

Modos disponibles:
- `solve` — Resuelve el problema mostrando todo el procedimiento
- `verify` — Verifica si un resultado dado es correcto
- `hint` — Da pistas sin resolver completamente
- `explain_error` — Localiza el error en el procedimiento del alumno

### Sessions — Tutor Adaptativo (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/sessions/session` | Crear sesion adaptativa (tutor/practice/review/exam) |
| POST | `/api/sessions/answer` | Responder ejercicio (verifica + actualiza mastery) |
| POST | `/api/sessions/hint` | Solicitar pista progresiva |
| POST | `/api/sessions/feedback` | Analizar procedimiento paso a paso |

Crear sesion:
```json
{ "mode": "practice", "course_id": "matematica-1" }
```

Enviar respuesta:
```json
{
  "session_id": "uuid",
  "exercise_id": "uuid",
  "answer": "x = -2, x = -3",
  "procedure": ["x^2+5x+6=0", "(x+2)(x+3)=0", "x=-2, x=-3"]
}
```

Response:
```json
{
  "correct": true,
  "score": 1.0,
  "feedback": "¡Correcto! Excelente resolución.",
  "mastery_before": 0.35,
  "mastery_after": 0.40,
  "mastery_status": "learning",
  "math_verified": true,
  "next_exercise": { "id": "...", "statement": "...", "difficulty": 3 }
}
```

### Ejercicios (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/exercises/generate` | Generar ejercicio (LLM + validacion SymPy) |
| GET | `/api/exercises/concept/{id}` | Ejercicio por concepto |
| GET | `/api/exercises/{id}/hints` | Pistas de un ejercicio |

### Learning — Knowledge Model (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/learning/courses/{id}/concepts` | Arbol de conceptos |
| GET | `/api/learning/courses/{id}/prerequisites` | Prerrequisitos |
| GET | `/api/learning/progress` | Perfil + mastery del estudiante |
| GET | `/api/learning/mastery` | Mapa de dominio por concepto |
| GET | `/api/learning/errors` | Errores del estudiante |

### Student Dashboard (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/student/my-progress` | Progreso completo (perfil, mastery, errores, stats) |
| GET | `/api/student/recommendations` | Siguiente concepto recomendado |

### Teacher Dashboard (requiere auth + TEACHER/ADMIN)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/teacher/course-progress` | Resumen del curso |
| GET | `/api/teacher/topic-mastery` | Dominio por tema |
| GET | `/api/teacher/common-errors` | Errores comunes del curso |
| GET | `/api/teacher/student-progress` | Progreso individual de estudiantes |

### Chat (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/chat/message` | Enviar mensaje (integra RAG automaticamente) |
| GET | `/api/chat/sessions` | Listar sesiones |
| GET | `/api/chat/sessions/{id}/messages` | Mensajes de sesion (con sources) |
| DELETE | `/api/chat/sessions/{id}` | Eliminar sesion |

### RAG (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/rag/query` | Consulta RAG hibrida (retorna answer + citations + confidence) |

### Math (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/math/evaluate` | Evaluar expresion |
| POST | `/api/math/differentiate` | Derivar |
| POST | `/api/math/integrate` | Integrar |
| POST | `/api/math/solve` | Resolver ecuacion |
| POST | `/api/math/simplify` | Simplificar |
| POST | `/api/math/factor` | Factorizar |
| POST | `/api/math/expand` | Expandir |

### Math Service (interno — Python)

Endpoints del microservicio Python (no expuestos directamente al frontend):

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/health` | Health check del math service |
| POST | `/math/evaluate` | Evaluar expresion |
| POST | `/math/differentiate` | Derivar |
| POST | `/math/integrate` | Integrar (indefinida/definida) |
| POST | `/math/limit` | Calcular limite |
| POST | `/math/solve` | Resolver ecuacion o sistema |
| POST | `/math/simplify` | Simplificar |
| POST | `/math/factor` | Factorizar |
| POST | `/math/expand` | Expandir |
| POST | `/math/matrix` | Operaciones con matrices |
| POST | `/math/verify` | Verificar resultado |
| POST | `/math/validate-exercise` | Validar ejercicio generado |

### Documents (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/documents/upload` | Subir documento (PDF, DOCX, TXT, MD) |
| GET | `/api/documents` | Listar documentos del usuario |
| GET | `/api/documents/{id}/chunks` | Ver chunks vectoriales de un documento |
| DELETE | `/api/documents/{id}` | Eliminar documento + vectores |

### Indexer (requiere admin)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/indexer/reindex` | Re-indexar todos los documentos o uno especifico |

### Users (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/users` | Listar usuarios |
| POST | `/api/users` | Crear usuario |
| PUT | `/api/users/{id}/role` | Cambiar rol de usuario |
| DELETE | `/api/users/{id}` | Eliminar usuario |

### Settings (requiere admin)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/settings` | Listar configuraciones |
| GET | `/api/settings/{key}` | Obtener configuracion |
| PUT | `/api/settings/{key}` | Crear/actualizar configuracion |
| DELETE | `/api/settings/{key}` | Eliminar configuracion |
| POST | `/api/settings/verify-openai` | Verificar conexion con proveedor IA |

### Stats (requiere admin)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/stats/admin` | Estadisticas del sistema |

### History (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/history` | Historial completo de sesiones |

### Analytics

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/analytics/overview` | Resumen de uso |
| GET | `/api/analytics/daily` | Uso diario |
| GET | `/api/analytics/modelos` | Uso por modelo |
| GET | `/api/analytics/top-users` | Top usuarios |

## Variables de Entorno

### Backend (Go)

```env
# === REQUERIDAS ===
DB_URL=postgresql://user:pass@host:5432/matematica?sslmode=disable
JWT_SECRET=tu_jwt_secret_min_32_bytes
OPENAI_API_KEY=sk-your-openai-key

# === QDRANT (requerido si Qdrant tiene auth habilitada) ===
QDRANT_HOST=host-qdrant
QDRANT_PORT=6333
QDRANT_API_KEY=tu_qdrant_api_key

# === MATH SERVICE (requerido para tutor) ===
MATH_SERVICE_URL=http://math-service:5000
MATH_TIMEOUT=5

# === CORS ===
CORS_ALLOWED_ORIGINS=https://matematica.face-unt.ar

# === OPCIONALES ===
PORT=8008
EMBEDDING_TYPE=openai
APP_ENV=production

# === RAG (opcional — valores por defecto) ===
CHUNK_SIZE=500
CHUNK_OVERLAP=50
VECTOR_WEIGHT=0.60
KEYWORD_WEIGHT=0.40
RETRIEVAL_TOP_K=20
RERANK_TOP_K=5
RAG_MIN_SCORE=0.70
ENABLE_HYBRID_SEARCH=true
ENABLE_RERANKER=true
ENABLE_CITATIONS=true

# === ADAPTIVE ENGINE (opcional — valores por defecto) ===
ADAPTIVE_HINT_WEIGHT=0.1
ADAPTIVE_ERROR_WEIGHT=0.03
ADAPTIVE_MASTERY_THRESHOLD=0.8
ADAPTIVE_MAX_DIFFICULTY=5
```

### Math Service (Python)

```env
MATH_TIMEOUT=5
MATH_PORT=5000
```

## Configuracion de IA

El sistema soporta multiples proveedores de IA, configurable desde **Configuracion > API Keys**:

| Proveedor | Modelos soportados | Uso |
|-----------|-------------------|-----|
| OpenAI | gpt-4.1, gpt-4o, o3, o4-mini | Chat, RAG, Tutor, Clasificacion, Ejercicios |
| Anthropic | claude-opus-4-7, claude-sonnet-4-6, claude-haiku-4-5 | Chat, RAG |
| Groq | llama-4-scout, llama-3.3-70b, mixtral-8x7b | Chat rapido |
| OpenRouter | Multiples proveedores con una sola key | Acceso flex |

Los prompts del sistema son configurables:
- **CHAT_SYSTEM_PROMPT**: prompt para el chat general
- **RAG_SYSTEM_PROMPT**: prompt para consultas RAG (define como citar fuentes)

## Desarrollo Local

```bash
# 1. Clonar
git clone https://github.com/brandall2021/matematicarag.git
cd matematicarag

# 2. Configurar entorno
cp .env.example .env
# Editar .env con tus credenciales

# 3. Levantar servicios
docker-compose up -d qdrant math-service

# 4. Ejecutar backend
export DB_URL="postgresql://user:pass@localhost:5432/matematicarag"
export JWT_SECRET="tu_secret_aqui"
export OPENAI_API_KEY="sk-..."
export QDRANT_API_KEY="tu_qdrant_key"
export MATH_SERVICE_URL="http://localhost:5000"
go run ./cmd/server

# 5. Frontend (otra terminal)
cd frontend
npm ci --legacy-peer-deps
npx ng serve
```

## Deploy con Dokploy

### 1. Crear repositorio

```bash
gh repo create brandall2021/matematicarag --public
git add -A && git commit -m "feat: initial" && git push
```

### 2. Configurar en Dokploy

#### Servicio Backend

- **Tipo**: Custom Application (Docker)
- **Repositorio**: `github.com/brandall2021/matematicarag`
- **Branch**: `main`
- **Dockerfile**: `Dockerfile` (en la raiz)
- **Puerto**: `8008`

Variables de entorno del backend en Dokploy:

```
# === REQUERIDAS ===
DB_URL=postgresql://user:pass@host:5432/matematica?sslmode=disable
JWT_SECRET=tu_jwt_secret
OPENAI_API_KEY=sk-...

# === QDRANT ===
QDRANT_HOST=host-del-qdrant
QDRANT_PORT=6333
QDRANT_API_KEY=tu_qdrant_api_key

# === MATH SERVICE ===
MATH_SERVICE_URL=http://math-service:5000
MATH_TIMEOUT=5

# === CORS ===
CORS_ALLOWED_ORIGINS=https://matematica.face-unt.ar

# === ADAPTIVE ENGINE ===
ADAPTIVE_HINT_WEIGHT=0.1
ADAPTIVE_ERROR_WEIGHT=0.03
ADAPTIVE_MASTERY_THRESHOLD=0.8
ADAPTIVE_MAX_DIFFICULTY=5
```

#### Servicio Math Service

Crear un segundo servicio en Dokploy:

- **Tipo**: Custom Application (Docker)
- **Build context**: `./math-service`
- **Dockerfile**: `math-service/Dockerfile`
- **Puerto**: `5000`

Variables de entorno del math service:

```
MATH_TIMEOUT=5
MATH_PORT=5000
```

#### Servicio Frontend

- **Tipo**: Custom Application (Docker)
- **Build context**: `./frontend`
- **Dockerfile**: `frontend/Dockerfile`
- **Puerto**: `8008`

### 3. Red Docker

Todos los servicios deben estar en la misma red Docker para comunicarse entre si. En Dokploy, configurar la red `matematicarag-net` para todos los servicios.

### 4. Health Checks

| Servicio | Health check |
|----------|-------------|
| Backend | `GET /health` en puerto 8008 |
| Math Service | `GET /health` en puerto 5000 |
| Qdrant | TCP connection en puerto 6333 |

### 5. Deploy

Ejecutar "Deploy" en el dashboard de Dokploy para cada servicio.

### 6. Verificar

```bash
# Verificar backend
curl https://matematica.face-unt.ar/health

# Verificar tutor
curl -X POST https://matematica.face-unt.ar/api/tutor/solve \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"query": "Que es una derivada?"}'

# Verificar sesion adaptativa
curl -X POST https://matematica.face-unt.ar/api/sessions/session \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"mode": "practice", "course_id": "matematica-1"}'
```

## Seed de Datos

Para crear los usuarios de ejemplo y el grafo de conceptos:

```bash
cd scripts
export DB_URL="postgresql://user:pass@host:5432/matematica?sslmode=disable"
go run seed.go
```

Los conceptos de matematica-1 se crean automaticamente via migraciones (arbol de 22 conceptos con prerrequisitos).

## Tests

### Math Engine (Python)

```bash
cd math-service
python -m pytest tests/ -v
```

### RAG Evaluation (Go)

```bash
cd .
go run ./cmd/ragtest
```

### Learning Integration (Go)

```bash
# Requiere PostgreSQL + math-service corriendo
go test ./api/ -run TestMastery -v
```

## Footer

Desarrollado por [softgroup.com.ar](https://softgroup.com.ar) &copy; 2026
