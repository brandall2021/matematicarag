# MatematicaRAG

Tutor de matematicas universitarias con Inteligencia Artificial y RAG (Retrieval-Augmented Generation). Incluye source attribution con cites a documentos fuente, extraccion de metadata por pagina y seccion, y disenio responsive mobile-first.

## Caracteristicas

- **Chat con IA** integrado con RAG: el LLM cita automaticamente las fuentes utilizadas
- **Source attribution**: cada respuesta incluye chips clickeables con nombre del documento, pagina y seccion
- **Motor matematico**: evaluar, derivar, integrar, limites, matrices, ecuaciones (via LLM)
- **Gestion documental**: subir PDF/DOCX/TXT/MD, indexacion automatica en base vectorial
- **Extraccion de metadata**: paginas (PDF), secciones (heuristica), URLs de procedencia
- **Multi-proveedor IA**: OpenAI, Anthropic (Claude), Groq, OpenRouter
- **Responsive**: mobile-first con hamburger menu, funciona en 375px+
- **Dark/Light theme**: toggle en el sidebar
- **Auth JWT**: registro, login, refresh token, roles (Admin, Profesor, Alumno)
- **Panel admin**: estadisticas, gestion de usuarios, configuracion de prompts y API keys

## Stack

| Capa | Tecnologia |
|------|-----------|
| Backend | Go 1.25 + Chi v5 |
| Frontend | Angular 20 + Material Design |
| Database | PostgreSQL 16 |
| Vector DB | Qdrant v1.12 |
| Embeddings | OpenAI text-embedding-3-small (1536 dim) |
| IA | OpenAI GPT-4 / Anthropic Claude / Groq Llama / OpenRouter |
| Auth | JWT (HS256) + bcrypt |
| Deploy | Docker + Dokploy |

## Estructura

```
matematicarag/
├── cmd/server/main.go              # Entry point + routing
├── api/
│   ├── auth.go                     # Register, Login, Refresh Token
│   ├── chat.go                     # Chat con integracion RAG automatica
│   ├── math.go                     # Operaciones matematicas via LLM
│   ├── rag.go                      # Consultas RAG con source attribution
│   ├── documents.go                # Upload, chunking, embeddings, metadata
│   ├── qdrant.go                   # Cliente Qdrant (vectores + payload)
│   ├── openai.go                   # Multi-proveedor LLM
│   ├── settings.go                 # CRUD configuracion
│   ├── users.go                    # Gestion de usuarios
│   ├── stats.go                    # Estadisticas admin
│   ├── history.go                  # Historial de sesiones
│   ├── analytics.go                # Analiticas de uso
│   ├── indexer.go                  # Re-indexacion
│   └── middleware.go               # JWT Auth + Role middleware
├── internal/
│   ├── config/config.go            # Variables de entorno
│   ├── database/database.go        # PostgreSQL + Migraciones
│   └── middleware/middleware.go     # CORS + Rate Limiting
├── frontend/src/app/
│   ├── core/
│   │   ├── guards/                 # Auth + Role guards
│   │   ├── services/               # Auth + API services
│   │   └── interceptors/           # HTTP interceptors (auth, errors)
│   └── modules/
│       ├── auth/                   # Login + Register
│       ├── chat/                   # Chat con IA + fuentes clickeables
│       ├── math/                   # Calculadora matematica (MathLive + KaTeX)
│       ├── documents/              # Gestion de archivos + vista de chunks
│       ├── history/                # Historial de sesiones
│       ├── dashboard/              # Dashboard (Admin/Teacher)
│       ├── admin/                  # Panel administrativo
│       └── settings/               # Configuracion (prompts, API keys, IA)
├── scripts/
│   └── seed.go                     # Script de datos de ejemplo
├── Dockerfile                      # Multi-stage (Node + Go + Alpine)
├── docker-compose.yml              # Qdrant + Backend + Frontend
└── .env.example                    # Template de variables
```

## Pipeline RAG

```
1. Upload:     Usuario sube PDF → extractTextWithPages() extrae texto por pagina
2. Chunking:   chunkTextWithMetadata() crea chunks de 500 runes con overlap de 50
               Detecta secciones con regex (capitulos, numeracion, headings)
3. Embeddings: generateEmbeddings() llama a OpenAI text-embedding-3-small
4. Storage:    qdrantUpsert() guarda chunk con payload:
               {document_id, chunk_index, content, filename, page, section, url}
5. Search:     VectorSearch() genera embedding de la query y busca en Qdrant
6. Context:    Construye contexto enriquecido:
               [Fuente: algebra.pdf, pagina 5, seccion "Polinomios"]
               contenido del chunk...
7. LLM:        El system prompt exige citas: [Fuente: nombre, pag X, sec "Y"]
8. Response:   Frontend renderiza chips clickeables que navegan al documento
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
| POST | `/api/rag/query` | Consulta RAG directa ( retorna answer + sources ) |

### Math (publico)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/math/evaluate` | Evaluar expresion |
| POST | `/api/math/plot` | Graficar funcion |
| POST | `/api/math/derive` | Derivar |
| POST | `/api/math/integrate` | Integrar |
| POST | `/api/math/solve` | Ecuaciones |
| POST | `/api/math/simplify` | Simplificar |
| POST | `/api/math/factor` | Factorizar |
| POST | `/api/math/expand` | Expandir |
| POST | `/api/math/limit` | Limites |
| POST | `/api/math/sum` | Sumatorias |
| POST | `/api/math/product` | Productorias |
| POST | `/api/math/roots` | Raices |
| POST | `/api/math/matrix-determinant` | Determinante |
| POST | `/api/math/matrix-inverse` | Inversa |
| POST | `/api/math/matrix-rank` | Rango |

### Documents (requiere auth)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/documents/upload` | Subir documento (PDF, DOCX, TXT, MD) |
| GET | `/api/documents` | Listar documentos del usuario |
| GET | `/api/documents/{id}/chunks` | Ver chunks vectoriales de un documento |
| DELETE | `/api/documents/{id}` | Eliminar documento + vectores |

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

### Analytics (publico)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| GET | `/api/analytics/overview` | Resumen de uso |
| GET | `/api/analytics/daily` | Uso diario |
| GET | `/api/analytics/modelos` | Uso por modelo |
| GET | `/api/analytics/top-users` | Top usuarios |

### Indexer (publico)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| POST | `/api/indexer/reindex` | Re-indexar documentos |

## Variables de Entorno

```env
# Database (requerido)
DB_URL=postgresql://user:pass@host:5432/matematica

# Auth (requerido)
JWT_SECRET=base64_encoded_secret_min_32_bytes

# OpenAI / Embeddings (requerido para RAG y chat)
OPENAI_API_KEY=sk-your-openai-key

# CORS (requerido)
CORS_ALLOWED_ORIGINS=https://matematica.face-unt.ar

# Qdrant (opcional - valores por defecto)
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=your-qdrant-api-key

# Server (opcional)
PORT=8008
```

## Configuracion de IA

El sistema soporta multiples proveedores de IA, configurable desde **Configuracion > API Keys**:

| Proveedor | Modelos soportados | Uso |
|-----------|-------------------|-----|
| OpenAI | gpt-4.1, gpt-4o, o3, o4-mini | Chat, RAG, Math |
| Anthropic | claude-opus-4-7, claude-sonnet-4-6, claude-haiku-4-5 | Chat, RAG |
| Groq | llama-4-scout, llama-3.3-70b, mixtral-8x7b | Chat rapido |
| OpenRouter | Multiples proveedores con una sola key | Acceso flex |

Los prompts del sistema son configurables para cada modo:
- **CHAT_SYSTEM_PROMPT**: prompt para el chat general
- **MATH_SYSTEM_PROMPT**: prompt para operaciones matematicas
- **RAG_SYSTEM_PROMPT**: prompt para consultas RAG (define como citar fuentes)

## Desarrollo Local

```bash
# 1. Clonar
git clone https://github.com/brandall2021/matematicarag.git
cd matematicarag

# 2. Configurar entorno
cp .env.example .env
# Editar .env con tus credenciales

# 3. Levantar Qdrant
docker-compose up -d qdrant

# 4. Ejecutar backend
export DB_URL="postgresql://user:pass@localhost:5432/matematicarag"
export JWT_SECRET="tu_secret_aqui"
export OPENAI_API_KEY="sk-..."
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

- **Tipo**: Custom Application
- **Repositorio**: `github.com/brandall2021/matematicarag`
- **Branch**: `main`

### 3. Variables de entorno en Dokploy

```
DB_URL=postgresql://user:pass@host:5432/matematica?sslmode=disable
JWT_SECRET=tu_jwt_secret
OPENAI_API_KEY=sk-...
CORS_ALLOWED_ORIGINS=https://matematica.face-unt.ar
```

### 4. Puerto

Configurar en Dokploy el puerto `8008`.

### 5. Deploy

Ejecutar "Deploy" en el dashboard de Dokploy.

## Seed de Datos

Para crear los usuarios de ejemplo:

```bash
cd scripts
export DB_URL="postgresql://user:pass@host:5432/matematica?sslmode=disable"
go run seed.go
```

## Arquitectura

```
Browser → Go Server (8008)
            ├── /api/*      → API handlers
            │   ├── auth    → JWT register/login/refresh
            │   ├── chat    → Mensajes + integracion RAG automatica
            │   ├── rag     → Consultas RAG con source attribution
            │   ├── math    → Operaciones matematicas via LLM
            │   ├── docs    → Upload + chunking + embeddings
            │   ├── users   → Gestion de usuarios
            │   └── settings→ Configuracion (prompts, API keys, IA)
            ├── /health     → Health check
            └── /*          → Angular SPA (archivos estaticos embebidos)

Go Server → PostgreSQL (users, sessions, messages, documents, settings)
Go Server → Qdrant (vectores con metadata: page, section, url)
Go Server → OpenAI API (GPT-4 + text-embedding-3-small)
```

## Footer

Desarrollado por [softgroup.com.ar](https://softgroup.com.ar) &copy; 2026
