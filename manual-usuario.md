---
title: "MatematicaRAG — Manual de Usuario"
subtitle: "Sistema Tutor Adaptativo de Matemática con RAG Híbrido e IA"
author: "Equipo MatematicaRAG"
date: "Julio 2026"
toc: true
toc-depth: 3
lang: es
---

# Introducción

**MatematicaRAG** es un sistema tutor inteligente de matemática que combina
Recuperación Aumentada por Generación (RAG), un motor de cómputo simbólico
(SymPy), un agente pedagógico orquestado por IA, y un motor de aprendizaje
adaptativo. Está diseñado para estudiantes, docentes y administradores del
ámbito universitario.

El sistema permite:

-   **Chat con IA** con respuestas fundamentadas en documentos académicos.
-   **Tutor interactivo** que resuelve ejercicios paso a paso.
-   **Agente pedagógico** que orquesta múltiples herramientas para dar
    respuestas personalizadas.
-   **Evaluaciones** con calificación automática y detección de patrones de
    error.
-   **Seguimiento adaptativo** que ajusta dificultad y contenido al nivel de
    cada estudiante.
-   **Dashboards** con analítica en tiempo real para docentes y estudiantes.

---

# Roles de Usuario

El sistema reconoce tres roles con diferentes niveles de acceso:

| Rol | Descripción |
|-----|-------------|
| **Estudiante** | Puede usar el chat, el tutor, el agente pedagógico, realizar evaluaciones, ver su progreso y recibir recomendaciones adaptativas. |
| **Docente** | Todo lo del estudiante más: crear y gestionar evaluaciones, ver dashboards de curso, analizar errores comunes, exportar datos, y monitorear el progreso de sus alumnos. |
| **Administrador** | Acceso completo: gestionar usuarios, configurar el sistema (API keys, prompts, parámetros), ver estadísticas globales, y auditar la actividad del sistema. |

---

# Primeros Pasos

## Registro e Inicio de Sesión

1.  Acceda a la URL del sistema.
2.  Haga clic en **Registrarse**.
3.  Complete sus datos: nombre, correo electrónico, contraseña.
4.  Seleccione su rol (Estudiante, Docente).
5.  Confirme el registro.
6.  Inicie sesión con su correo y contraseña.

> Los administradores son creados por otros administradores desde el panel
> de administración.

## Navegación Principal

Una vez autenticado, verá un menú lateral con las siguientes secciones
(visibles según su rol):

-   **Chat** — Consultas con IA más RAG.
-   **Tutor** — Resolución de ejercicios paso a paso.
-   **Agente Pedagógico** — Asistente orquestado con herramientas.
-   **Motor Matemático** — Cálculos simbólicos directos.
-   **Evaluaciones** — Creación y realización de exámenes.
-   **Documentos** — Gestión de material académico.
-   **Mi Progreso** — Dashboard de aprendizaje (estudiante).
-   **Dashboard Docente** — Analítica del curso (docente).
-   **Historial** — Consultas anteriores.
-   **Administración** — Configuración del sistema (admin).
-   **Configuración** — Preferencias personales.

---

# Chat con RAG

## ¿Qué es?

El chat permite hacer preguntas en lenguaje natural sobre matemática. El
sistema busca en documentos académicos indexados (RAG híbrido) y genera una
respuesta con fundamento y citas verificables.

## Cómo usar

1.  Vaya a la sección **Chat**.
2.  Escriba su pregunta en la barra de escritura matemática (puede usar el
    teclado virtual para insertar símbolos matemáticos).
3.  Presione **Enter** o haga clic en **Enviar**.
4.  La respuesta incluirá:
    -   Explicación detallada.
    -   **Citas** (`[SRC-xxx]`) con enlace a la fuente.
    -   Porcentaje de relevancia de cada fuente.
5.  Haga clic en una cita para ver el fragmento completo del documento.

## Consejos

-   Sea específico en su pregunta.
-   Use notación matemática con el teclado virtual (`x^2`, `\int`, `\sqrt`).
-   Puede hacer preguntas de seguimiento para profundizar.

---

# Tutor Interactivo

## ¿Qué es?

El tutor resuelve problemas matemáticos paso a paso. Clasifica
automáticamente el tipo de problema, busca información relevante en los
documentos, ejecuta cálculos simbólicos, y genera una explicación
estructurada.

## Cómo usar

1.  Vaya a la sección **Tutor**.
2.  Seleccione el modo:
    -   **Resolver** — Ingrese un problema y obtenga la solución paso a paso.
    -   **Tutor** — Sesión adaptativa con ejercicios según su nivel.
    -   **Práctica** — Ejercicios de práctica con verificación.
    -   **Repaso** — Repaso de conceptos con ejercicios de recuperación.
3.  En modo **Resolver**:
    -   Escriba su problema en el campo de texto y/o en el editor matemático.
    -   El sistema clasifica la intención (derivar, integrar, factorizar,
        resolver ecuación, etc.).
    -   Obtendrá: tipo de problema detectado, explicación conceptual, cálculo
        paso a paso, resultado en LaTeX, y fuentes consultadas.
4.  En modo **Tutor/Práctica/Repaso**:
    -   El sistema selecciona el concepto y dificultad según su nivel de
        maestría.
    -   Resuelva el ejercicio.
    -   El sistema verifica la respuesta, analiza sus pasos, y actualiza su
        nivel de maestría.
    -   Puede solicitar **pistas progresivas** si se queda trabado.
    -   Recibe retroalimentación detallada con corrección de errores.

## Intenciones Detectadas

El tutor reconoce automáticamente estos tipos de problema:

| Intención | Ejemplo |
|-----------|---------|
| Derivada | "Derivar x^2 + 3x" |
| Integral | "Integrar sin(x) dx" |
| Límite | "Calcular límite de (x^2-1)/(x-1) cuando x->1" |
| Ecuación | "Resolver x^2 - 4 = 0" |
| Sistema de ecuaciones | "Resolver 2x + y = 5, x - y = 1" |
| Factorización | "Factorizar x^2 - 5x + 6" |
| Simplificación | "Simplificar (x^2 - 1)/(x - 1)" |
| Matriz | "Calcular determinante de [[1,2],[3,4]]" |
| Graficar | "Graficar y = x^2" |
| Álgebra lineal | "Resolver sistema lineal" |
| Trigonometría | "Simplificar sin^2(x) + cos^2(x)" |
| Concepto | "Qué es una derivada?" |
| Ejercicio | generación de ejercicios |
| Verificación | verificar respuesta |
| Teoría | explicación conceptual |
| General | consulta general con RAG |

---

# Agente Pedagógico

## ¿Qué es?

El agente pedagógico es un asistente orquestado que utiliza múltiples
herramientas para responder consultas de forma inteligente. A diferencia del
chat simple, el agente:

1.  **Analiza su intención** y contexto (historial, nivel de maestría).
2.  **Selecciona herramientas** según lo que necesita: RAG, motor matemático,
    ejercicios, evaluaciones, etc.
3.  **Ejecuta un plan** de varios pasos.
4.  **Genera una respuesta pedagógica** con explicaciones, fuentes y
    recomendaciones de aprendizaje.
5.  **Actualiza su perfil** de aprendizaje con los resultados.

## Cómo usar

1.  Vaya a la sección **Agente Pedagógico**.
2.  Escriba su consulta en la barra de escritura matemática.
3.  El agente mostrará:
    -   Indicador de **"Analizando consulta pedagógica..."**.
    -   Una vez listo, la respuesta incluye:
        -   **Estrategia** utilizada (ej: "tutor", "exploración", "ejercicio").
        -   **Nivel de confianza** de la respuesta.
        -   **Pasos ejecutados** (desplegables con detalle de cada
            herramienta usada).
        -   **Contenido** con explicaciones.
        -   **Fuentes** consultadas.
        -   **Sección de aprendizaje** con temas cubiertos y delta de
            maestría.
4.  Puede hacer clic en cada paso para ver los detalles (input, output,
    duración).

## Herramientas del Agente

El agente puede usar:

| Herramienta | Propósito |
|-------------|-----------|
| RAG | Búsqueda en documentos académicos |
| Math | Cálculo simbólico (SymPy) |
| Verify | Verificación de respuestas |
| Student Info | Consulta de perfil y progreso del estudiante |
| Exercise | Generación de ejercicios adaptativos |
| Grading | Evaluación de respuestas |
| Hint | Pistas progresivas |
| Assessment | Información de evaluaciones |

## Sugerencias Iniciales

Al abrir el agente, verá sugerencias como:

-   "Explicame cómo derivar x² + 3x"
-   "Ayudame con la regla de la cadena"
-   "Qué es una integral definida?"
-   "Tenemos ejercicios de límites?"

---

# Motor Matemático

## ¿Qué es?

Es una interfaz directa al motor SymPy para cálculos simbólicos. Útil para
verificar resultados o explorar expresiones matemáticas rápidamente.

## Cómo usar

1.  Vaya a la sección **Motor Matemático**.
2.  Escriba su expresión en el editor matemático (WYSIWYG con teclado
    virtual).
3.  Vea la previsualización LaTeX en vivo.
4.  Haga clic en **Calcular**.
5.  El resultado se muestra renderizado con KaTeX.

---

# Evaluaciones

## Para Estudiantes

### Realizar una Evaluación

1.  Vaya a la sección **Evaluaciones**.
2.  Verá la lista de evaluaciones disponibles (publicadas por su docente).
3.  Haga clic en una evaluación para iniciarla.
4.  Tipos de preguntas:
    -   **Ejercicio** — Respuesta libre con verificación SymPy.
    -   **Opción múltiple** — Seleccione la respuesta correcta.
    -   **Verdadero/Falso** — Decida si la afirmación es correcta.
    -   **Numérica** — Ingrese un valor numérico.
    -   **Algebraica** — Ingrese una expresión algebraica.
    -   **Ecuación** — Resuelva la ecuación.
    -   **Paso a paso** — Ingrese cada paso de la resolución.
5.  Sus respuestas se **guardan automáticamente**.
6.  Puede pausar y **reanudar** la evaluación más tarde.
7.  Al enviar, el sistema corrige automáticamente usando:
    -   Verificación con el motor SymPy.
    -   Evaluación con rúbrica (si el docente la definió).
8.  Recibirá su calificación y retroalimentación inmediata.

## Para Docentes

### Crear una Evaluación

1.  Vaya a **Evaluaciones** → **Crear Evaluación**.
2.  Complete:
    -   **Título** de la evaluación.
    -   **Descripción** e instrucciones.
    -   **Tipo**: diagnóstico, formativo, sumativo, recuperación, práctica.
    -   **Modo**: fijo (usted elige las preguntas), generado (IA crea las
        preguntas), adaptativo (se ajusta al desempeño del estudiante).
    -   **Curso**, **unidad**, **tema**.
    -   **Duración** límite (opcional).
    -   **Fecha límite** de entrega.
    -   **Intentos** permitidos.
    -   **Modo de revisión**: inmediata o después de la fecha límite.
3.  Agregue preguntas al banco de preguntas o seleccione del banco existente.
4.  Publique la evaluación para que los estudiantes la vean.

### Tipos de Evaluación

| Tipo | Propósito |
|------|-----------|
| Diagnóstico | Evaluar conocimientos previos |
| Formativo | Seguimiento durante el aprendizaje |
| Sumativo | Evaluación final de unidad/curso |
| Recuperación | Para estudiantes que necesitan recuperar |
| Práctica | Auto-evaluación sin calificación |

### Calificación y Reportes

-   Las evaluaciones se corrigen automáticamente.
-   Puede ver los resultados agregados por curso.
-   Puede exportar las calificaciones a **CSV**.
-   El sistema genera alertas académicas para estudiantes en riesgo.

---

# Documentos

## ¿Qué es?

El módulo de documentos permite gestionar el material académico que alimenta
el RAG. Los documentos son procesados, fragmentados (chunking), y sus
vectores se indexan en Qdrant para búsqueda semántica.

## Para Estudiantes

-   Ver la lista de documentos disponibles.
-   Cada documento puede tener asociados: curso, unidad, tema.
-   Puede buscar dentro de los documentos.

## Para Docentes y Administradores

### Subir Documentos

1.  Vaya a **Documentos** → **Subir**.
2.  Formatos soportados: **PDF**, **DOCX**, **TXT**, **MD**.
3.  Complete metadatos: título, autor, curso, unidad, tema, etiquetas.
4.  El sistema extrae el texto, lo fragmenta en chunks, genera embeddings, y
    lo indexa en Qdrant.

### Gestionar Documentos

-   Ver lista de documentos subidos.
-   Editar metadatos.
-   Eliminar documentos (también elimina sus vectores).
-   Re-indexar documentos si es necesario.

---

# Mi Progreso (Estudiante)

## ¿Qué es?

Dashboard personal que muestra su evolución en el aprendizaje de matemática.

## Qué incluye

-   **Resumen general**: estadísticas de sesiones, ejercicios resueltos,
    precisión, tiempo total de estudio.
-   **Mapa de maestría por concepto**: barras de progreso para cada concepto
    (álgebra, funciones, límites, derivadas, integrales, etc.).
-   **Gráfico de tendencia**: evolución de la maestría en el tiempo.
-   **Árbol de conceptos**: visualización de prerrequisitos entre conceptos.
-   **Errores comunes**: patrones de error detectados automáticamente.
-   **Recomendaciones**: siguiente concepto a estudiar según el motor
    adaptativo.
-   **Historial de sesiones de tutor**: ejercicios realizados y resultados.

## Conceptos en el Sistema

El sistema actualmente cubre estos conceptos (ordenados por prerrequisitos):

1.  Álgebra básica
2.  Ecuaciones lineales
3.  Sistema de ecuaciones
4.  Factorización
5.  Funciones y gráficas
6.  Trigonometría
7.  Límites
8.  Continuidad
9.  Derivadas (reglas básicas)
10. Regla de la cadena
11. Derivadas de orden superior
12. Aplicaciones de la derivada
13. Introducción a integrales
14. Técnicas de integración
15. Integral definida
16. Teorema fundamental del cálculo
17. Aplicaciones de la integral
18. Matrices y determinantes
19. Vectores
20. Números complejos
21. Sucesiones y series
22. Ecuaciones diferenciales básicas

---

# Dashboard Docente

## ¿Qué es?

Panel de analítica para que el docente monitoree el progreso de su curso.

## Qué incluye

-   **Resumen del curso**: estudiantes activos, ejercicios resueltos, tasa de
    finalización.
-   **Maestría por tema**: nivel de dominio promedio del curso en cada
    concepto.
-   **Errores comunes del curso**: ranking de errores más frecuentes con su
    frecuencia y severidad.
-   **Progreso individual**: seleccione un estudiante para ver su detalle.
-   **Conceptos críticos**: identifica los conceptos donde los estudiantes
    tienen más dificultad y que bloquean el avance en otros conceptos.
-   **Análisis profundo por concepto**: desglose por concepto crítico que
    muestra:
    -   Estudiantes con dificultades.
    -   Distribución de errores.
    -   Tendencia semanal.
    -   Recomendaciones pedagógicas.
-   **Exportar datos**: descargue reportes en **CSV**.
-   **Alertas académicas**: estudiantes identificados como en riesgo.

---

# Historial

## ¿Qué es?

Registro cronológico de todas las consultas realizadas en el chat, tutor y
agente.

## Cómo usar

1.  Vaya a la sección **Historial**.
2.  Verá la lista de interacciones ordenadas por fecha.
3.  Puede filtrar por tipo (chat, tutor, agente).
4.  Haga clic en una entrada para ver el detalle completo de la conversación.

---

# Administración

> Solo visible para usuarios con rol **Administrador**.

## Gestión de Usuarios

1.  Vaya a **Administración** → **Usuarios**.
2.  Puede:
    -   **Ver** lista completa de usuarios.
    -   **Crear** nuevos usuarios (incluyendo administradores).
    -   **Editar** roles y datos de usuarios.
    -   **Desactivar** cuentas.
    -   **Eliminar** usuarios.

## Configuración del Sistema

### Proveedores de IA

Vaya a **Configuración** → **API Keys** (o Administración → Configuración).

El sistema soporta múltiples proveedores:

| Proveedor | Modelos | Uso |
|-----------|---------|-----|
| OpenAI | gpt-4.1, gpt-4o, o3, o4-mini | Chat, RAG, Tutor, Clasificación |
| Anthropic | claude-opus-4-7, claude-sonnet-4-6 | Chat, RAG |
| Groq | llama-4-scout, llama-3.3-70b | Chat rápido |
| OpenRouter | Múltiples proveedores | Acceso flexible |

Puede configurar:

-   **API Key** para cada proveedor.
-   **Modelo por defecto** para cada funcionalidad.
-   **Prompt del sistema** para Chat y RAG (personalizable).

### Parámetros Generales

-   **Límite de tokens** por respuesta.
-   **Temperatura** del modelo.
-   **Máximo de fuentes** en respuestas RAG.
-   **Timeout** del motor matemático.
-   **Umbrales del circuit breaker**.
-   **Parámetros del motor adaptativo**: pesos de maestría, umbrales de
    dificultad, etc.

### Gestión de Qdrant

-   Ver estado de la colección de vectores.
-   Re-indexar todos los documentos.
-   Ver estadísticas de chunks indexados.

## Estadísticas del Sistema

-   **Métricas en tiempo real**: requests por ruta, tasa de error, duración
    promedio.
-   **Estado de servicios**: base de datos, motor matemático, Qdrant,
    circuit breakers.
-   **Uso de memoria** y goroutines activas.
-   **Health check**: endpoint `/health` con estado de todos los servicios.

## Auditoría

-   Registro detallado de cambios en evaluaciones y configuraciones.
-   Versión anterior y nueva de cada cambio.
-   Trazabilidad de quién hizo cada cambio.

---

# Barra de Escritura Matemática

## ¿Qué es?

Es un editor matemático interactivo (mathlive) disponible en el Chat, el
Agente, el Tutor, y el Motor Matemático. Permite escribir notación
matemática de forma visual, con teclado virtual integrado.

## Cómo usar

-   **Escribir directamente**: use el teclado para escribir texto o números.
-   **Teclado virtual**: se activa automáticamente al hacer clic en el campo.
-   **Comandos LaTeX**: escriba `\` seguido del comando (ej: `\sqrt`, `\int`,
    `\frac`).
-   **Smart fence**: los paréntesis se auto-completan.
-   **Smart superscript**: escriba `^` para superíndice automático.
-   **Atajos**: `^` para exponente, `_` para subíndice, `/` para fracción.
-   **Enter** envía el mensaje.

---

# Conceptos Técnicos

## RAG Híbrido

El sistema combina dos técnicas de búsqueda:

1.  **Búsqueda vectorial** (Qdrant): encuentra fragmentos semánticamente
    similares a la consulta usando embeddings.
2.  **Búsqueda por palabras clave** (PostgreSQL tsvector): encuentra
    fragmentos con términos exactos.

Ambos resultados se fusionan y re-rankear por un LLM para dar la respuesta
más relevante.

## Motor Adaptativo

El sistema construye un **perfil de maestría** para cada estudiante por cada
concepto. Cada vez que el estudiante:

-   Resuelve un ejercicio correctamente: aumenta la maestría.
-   Se equivoca: disminuye la maestría y se registra el tipo de error.
-   Pasa tiempo sin practicar un concepto: la maestría decae
    gradualmente.

El motor recomienda el siguiente concepto y dificultad óptimos para
maximizar el aprendizaje.

## Circuit Breaker

El motor matemático tiene un **circuit breaker** que protege al sistema:

-   Si hay 5 fallos consecutivos, el circuito se **abre** por 30 segundos.
-   Durante ese tiempo, las requests se rechazan inmediatamente.
-   Después de 30 segundos, pasa a **half-open** y permite 3 requests de
    prueba.
-   Si las pruebas fallan, vuelve a abrirse. Si funcionan, se **cierra** y
    vuelve a la normalidad.

---

# Solución de Problemas

## Error: "Servicio no disponible"

Verifique que el math-service esté corriendo. Si el circuit breaker está
abierto, espere 30 segundos y reintente.

## Error: "No se encontraron fuentes"

El RAG no encontró documentos relevantes. Asegúrese de que haya documentos
subidos e indexados para el tema de su consulta.

## Error de autenticación

Su sesión puede haber expirado. Cierre sesión y vuelva a iniciar.

## Las respuestas no son precisas

-   Sea más específico en su consulta.
-   Use notación matemática correcta.
-   Verifique que los documentos subidos sean relevantes.
-   El administrador puede ajustar el prompt del sistema.

---

# Preguntas Frecuentes

**¿Qué tipos de archivos puedo subir como documentos?**
PDF, DOCX, TXT y Markdown.

**¿Cuánto tardan en indexarse los documentos?**
Depende del tamaño, pero generalmente es inmediato (unos segundos).

**¿Puedo usar el sistema sin conexión a Internet?**
No, el sistema requiere conexión a los servicios de IA y a la base de datos.

**¿Se guarda mi historial de conversaciones?**
Sí, todo el historial se guarda y puede consultarlo en la sección
**Historial**.

**¿Cómo se calcula la maestría?**
La maestría se calcula combinando el historial de respuestas correctas e
incorrectas, ponderando más los resultados recientes.

**¿Qué significa "half-open" en el circuit breaker?**
Significa que el sistema está probando si el motor matemático ya se
recuperó. Las siguientes requests serán evaluadas.

---

# Contacto y Soporte

Para soporte técnico o reportar problemas, contacte al administrador del
sistema.

Repositorio: <https://github.com/brandall2021/matematicarag>
