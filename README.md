# MatematicaRAG

Tutor de matematicas universitarias con Inteligencia Artificial, RAG (Retrieval-Augmented Generation) y motor matematico symbolic. Resuelve, explica y verifica ejercicios paso a paso con citations academicas.

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
- **Separacion fuentes/calculos**: distingue entre informacion recuperada (📚) y calculos realizados (🧮)

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
│   ├── tutor.go                    # POST /api/tutor/solve (orquestador)
│   ├── intent.go                   # Clasificador de intencion (LLM + keywords)
│   ├── mathclient.go               # Cliente HTTP para Python math service
│   ├── math.go                     # Operaciones matematicas via SymPy
│   ├── rag.go                      # Consultas RAG con source attribution
│   ├── reranker.go                 # Hybrid search + reranking LLM
│   ├── textsearch.go               # PostgreSQL full-text search
│   ├── documents.go                # Upload, chunking, embeddings, metadata
│   ├── qdrant.go                   # Cliente Qdrant (vectores + payload)
│   ├── openai.go                   # Multi-proveedor LLM
│   ├── settings.go                 # CRUD configuracion
│   ├── users.go                    # Gestion de usuarios
│   ├── stats.go                    # Estadisticas admin
│   ├── history.go                  # Historial de sesiones
│   ├── analytics.go                # Analiticas de uso
│   ├── indexer.go                  # Re-indexacion (ADMIN)
│   ├── migration.go                # Helpers para migracion de datos
│   └── middleware.go               # JWT Auth + Role middleware
├── math-service/
│   ├── app.py                      # Flask API (11 endpoints matematicos)
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
│   ├── config/config.go            # Variables de entorno (incluye math service)
│   ├── database/database.go        # PostgreSQL + Migraciones
│   └── middleware/middleware.go     # CORS + Rate Limiting
├── frontend/src/app/
│   ├── core/
│   │   ├── guards/                 # Auth + Role guards
│   │   ├── services/               # Auth + API + Tutor services
│   │   └── interceptors/           # HTTP interceptors (auth, errors)
│   ├── shared/
│   │   ├── layout.component.ts     # Sidebar + responsive layout
│   │   └── render-math.pipe.ts     # Pipe KaTeX para LaTeX
│   └── modules/
│       ├── auth/                   # Login + Register
│       ├── chat/                   # Chat con IA + fuentes clickeables
│       ├── tutor/                  # Tutor matematico paso a paso
│       ├── math/                   # Calculadora matematica (MathLive)
│       ├── documents/              # Gestion de archivos + vista de chunks
│       ├── bdvectorial/            # Visor de docs en Qdrant
│       ├── history/                # Historial de sesiones
│       ├── dashboard/              # Dashboard (Admin/Teacher)
│       ├── admin/                  # Panel administrativo
│       └── settings/               # Configuracion (prompts, API keys, IA)
├── scripts/
│   └── seed.go                     # Script de datos de ejemplo
├── Dockerfile                      # Multi-stage (Node + Go + Alpine)
├── docker-compose.yml              # Qdrant + Backend + Frontend + Math Service
└── .env.example                    # Template de variables
```

## Arquitectura

```
Browser → Go Server (8008)
            ├── /api/tutor/solve   → Orquestador del tutor
            │   ├── ClassifyIntent (LLM: identifica tipo de problema)
            │   ├── HybridSearch (Qdrant vectorial + PostgreSQL keyword)
            │   ├── RerankResults (LLM: re-evalua relevancia)
            │   ├── MathClient → Python/SymPy (calculo symbolic)
            │   ├── LLM (genera explicacion paso a paso)
            │   └── Verification (verifica resultado matematico)
            ├── /api/chat          → Chat + RAG automatico
            ├── /api/rag           → Consultas RAG directas
            ├── /api/math          → Operaciones matematicas via SymPy
            ├── /api/documents     → Upload + chunking + embeddings
            ├── /api/indexer       → Re-indexacion (ADMIN)
            ├── /health            → Health check
            └── /*                 → Angular SPA (archivos estaticos)

Go Server → PostgreSQL (users, sessions, messages, documents, chunks, settings)
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

## Pipeline del Tutor

```
Pregunta del alumno
        ↓
Clasificacion (LLM + keyword fallback)
        ↓
RAG Hibrido (Qdrant + PostgreSQL)
        ↓
Contexto academico recuperado
        ↓
Math Engine (SymPy via Python service)
        ↓
Verificacion automatica
        ↓
LLM genera explicacion paso a paso
        ↓
Citation Engine
        ↓
Respuesta estructurada:
  📚 Fuente academica
  🧮 Calculo verificado
  ✅ Resultado verificado
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
| OpenAI | gpt-4.1, gpt-4o, o3, o4-mini | Chat, RAG, Tutor, Clasificacion |
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
QDRANT_API_KEY=0ylktnefcidr4f6dvkmwfoxc4nrgtywh

# === MATH SERVICE ===
MATH_SERVICE_URL=http://math-service:5000
MATH_TIMEOUT=5

# === CORS ===
CORS_ALLOWED_ORIGINS=https://matematica.face-unt.ar
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
```

## Seed de Datos

Para crear los usuarios de ejemplo:

```bash
cd scripts
export DB_URL="postgresql://user:pass@host:5432/matematica?sslmode=disable"
go run seed.go
```

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

## Footer

Desarrollado por [softgroup.com.ar](https://softgroup.com.ar) &copy; 2026
