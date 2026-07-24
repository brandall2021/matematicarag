# MatematicaRAG

Tutor de matemáticas universitarias con Inteligencia Artificial y RAG (Retrieval-Augmented Generation).

## Stack

| Capa | Tecnología |
|------|-----------|
| Backend | Go 1.25 + Chi v5 |
| Frontend | Angular 20 + Material Design |
| Database | PostgreSQL 16 |
| Vector DB | Qdrant v1.12 |
| IA | OpenAI GPT-4 + Embeddings |
| Auth | JWT (HS256) + bcrypt |
| Deploy | Docker + Dokploy |

## Estructura

```
matematicarag/
├── cmd/server/main.go          # Entry point
├── api/                        # Handlers HTTP
│   ├── auth.go                 # Register, Login, Refresh Token
│   ├── chat.go                 # Sesiones y mensajes de chat
│   ├── math.go                 # Operaciones matemáticas
│   ├── rag.go                  # Consultas RAG
│   ├── documents.go            # Gestión de documentos
│   ├── history.go              # Historial
│   ├── settings.go             # Configuración
│   ├── stats.go                # Estadísticas
│   ├── analytics.go            # Analíticas
│   ├── indexer.go              # Indexación de documentos
│   └── middleware.go           # JWT Auth + Role middleware
├── internal/
│   ├── config/config.go        # Variables de entorno
│   ├── database/database.go    # PostgreSQL + Migraciones
│   └── middleware/middleware.go # CORS + Rate Limiting
├── frontend/                   # Angular 20
│   └── src/app/
│       ├── core/
│       │   ├── guards/         # Auth + Role guards
│       │   └── services/       # Auth + API services
│       └── modules/
│           ├── auth/           # Login + Register
│           ├── chat/           # Chat con IA
│           ├── math/           # Calculadora matemática
│           ├── documents/      # Gestión de archivos
│           ├── history/        # Historial de sesiones
│           ├── dashboard/      # Dashboard (Admin/Teacher)
│           ├── admin/          # Panel administrativo
│           └── settings/       # Configuración
├── scripts/
│   └── seed.go                 # Script de datos de ejemplo
├── Dockerfile                  # Multi-stage (Node + Go + Alpine)
├── docker-compose.yml          # Qdrant + Backend + Frontend
└── .env.example                # Template de variables
```

## Usuarios de Ejemplo

| Rol | Email | Contraseña |
|-----|-------|-----------|
| Admin | admin@face-unt.ar | Admin123! |
| Profesor | profesor@face-unt.ar | Profesor123! |
| Alumno | alumno@face-unt.ar | Alumno123! |

## API Endpoints

### Auth (público)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/auth/register` | Registrar usuario |
| POST | `/api/auth/login` | Iniciar sesión |
| POST | `/api/auth/refresh` | Renovar token |

### Chat (requiere auth)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/chat/message` | Enviar mensaje |
| GET | `/api/chat/sessions` | Listar sesiones |
| GET | `/api/chat/sessions/{id}/messages` | Mensajes de sesión |
| DELETE | `/api/chat/sessions/{id}` | Eliminar sesión |

### Math (público)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/math/evaluate` | Evaluar expresión |
| POST | `/api/math/plot` | Graficar función |
| POST | `/api/math/derive` | Derivar |
| POST | `/api/math/integrate` | Integrar |
| POST | `/api/math/solve` | Ecuaciones |
| POST | `/api/math/simplify` | Simplificar |
| POST | `/api/math/factor` | Factorizar |
| POST | `/api/math/expand` | Expandir |
| POST | `/api/math/limit` | Límites |
| POST | `/api/math/sum` | Sumatorias |
| POST | `/api/math/product` | Productorias |
| POST | `/api/math/roots` | Raíces |
| POST | `/api/math/matrix-determinant` | Determinante |
| POST | `/api/math/matrix-inverse` | Inversa |
| POST | `/api/math/matrix-rank` | Rango |

### RAG (requiere auth)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/rag/query` | Consulta RAG |
| GET | `/api/rag/sources` | Fuentes indexadas |

### Documents (requiere auth)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/documents/upload` | Subir documento |
| GET | `/api/documents` | Listar documentos |
| DELETE | `/api/documents/{id}` | Eliminar documento |

### Otros (requiere auth)

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/history` | Historial completo |
| GET | `/api/settings` | Configuración |
| PUT | `/api/settings` | Actualizar config |
| GET | `/api/stats` | Estadísticas |
| GET | `/api/analytics` | Analíticas |

### Health

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/health` | Estado del servidor |

## Variables de Entorno

```env
# Database
DB_URL=postgresql://user:pass@host:5432/matematica

# Auth
JWT_SECRET=base64_encoded_secret_min_32_bytes

# OpenAI
OPENAI_API_KEY=sk-your-openai-key

# CORS
CORS_ALLOWED_ORIGINS=https://matematica.face-unt.ar

# Opcional
PORT=8008                          # Default: 8008
QDRANT_HOST=localhost              # Default: localhost
QDRANT_PORT=6334                   # Default: 6334
EMBEDDING_TYPE=openai              # Default: openai
APP_ENV=production                 # Default: development
STATIC_DIR=./static                # Default: ./static
```

## Desarrollo Local

```bash
# 1. Clonar
git clone https://github.com/brandall2021/matematicarag.git
cd matematicarag

# 2. Configurar entorno
cp .env.example .env
# Editar .env con tus credenciales

# 3. Levantar servicios
docker-compose up -d qdrant

# 4. Ejecutar backend
export PATH=$PATH:/usr/local/go/bin
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
DB_URL=postgresql://brandall:Hansol1974+@186.153.163.188:5432/matematica?sslmode=disable
JWT_SECRET=6r8dzbWWhhRr0FGH+Kqg8S+0K9r7gXJrR0kY7M4gLwTQy2gqzj8YvE3mQ6j5iK8wY4vYbQdR3cL7oX0Q9P2eNQ==
OPENAI_API_KEY=sk-proj-...
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
export DB_URL="postgresql://brandall:Hansol1974+@186.153.163.188:5432/matematica?sslmode=disable"
go run seed.go
```

## Arquitectura

```
Browser → Go Server (8008)
            ├── /api/*     → API handlers (auth, chat, math, rag, docs)
            ├── /health    → Health check
            └── /*         → Angular SPA (archivos estáticos embebidos)

Go Server → PostgreSQL (users, sessions, messages, documents, settings)
Go Server → Qdrant (vectores de documentos para RAG)
Go Server → OpenAI API (GPT-4 + embeddings)
```

## Licencia

Proyecto privado - FACE-UNT
