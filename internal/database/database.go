package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseURL string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return pool, nil
}

func Migrate(db *pgxpool.Pool) error {
	ctx := context.Background()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255),
			role VARCHAR(50) NOT NULL DEFAULT 'STUDENT',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(500),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
			role VARCHAR(20) NOT NULL,
			content TEXT NOT NULL,
			model VARCHAR(100),
			tokens_used INTEGER DEFAULT 0,
			sources JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			filename VARCHAR(500) NOT NULL,
			original_name VARCHAR(500) NOT NULL,
			type VARCHAR(50) NOT NULL,
			size BIGINT NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			metadata JSONB,
			uploaded_by UUID REFERENCES users(id),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key VARCHAR(255) PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id),
			action VARCHAR(100) NOT NULL,
			entity_type VARCHAR(100),
			entity_id UUID,
			details JSONB,
			ip_address VARCHAR(45),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS usage_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id),
			operation VARCHAR(50) NOT NULL,
			model VARCHAR(100),
			tokens_input INTEGER DEFAULT 0,
			tokens_output INTEGER DEFAULT 0,
			cost_cents INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_documents_uploaded_by ON documents(uploaded_by)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_user ON usage_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_created ON usage_logs(created_at)`,
		`CREATE TABLE IF NOT EXISTS document_chunks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			chunk_index INTEGER NOT NULL,
			content TEXT NOT NULL,
			page INTEGER DEFAULT 0,
			section TEXT DEFAULT '',
			topic TEXT DEFAULT '',
			content_type TEXT DEFAULT 'theory',
			has_formula BOOLEAN DEFAULT false,
			has_example BOOLEAN DEFAULT false,
			has_exercise BOOLEAN DEFAULT false,
			has_solution BOOLEAN DEFAULT false,
			course_id TEXT DEFAULT '',
			unit_id TEXT DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_doc ON document_chunks(document_id)`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_content_type ON document_chunks(content_type)`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_course ON document_chunks(course_id)`,
		`ALTER TABLE document_chunks ADD COLUMN IF NOT EXISTS content_tsv tsvector`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_tsv ON document_chunks USING gin(content_tsv)`,
		`CREATE OR REPLACE FUNCTION document_chunks_tsv_trigger() RETURNS trigger AS $$
		BEGIN
			NEW.content_tsv := to_tsvector('spanish', COALESCE(NEW.content, ''));
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		`DROP TRIGGER IF EXISTS tsvector_update ON document_chunks`,
		`CREATE TRIGGER tsvector_update BEFORE INSERT OR UPDATE ON document_chunks
			FOR EACH ROW EXECUTE FUNCTION document_chunks_tsv_trigger()`,

		// Knowledge Model
		`CREATE TABLE IF NOT EXISTS concepts (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			parent_id VARCHAR(100) REFERENCES concepts(id),
			course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
			difficulty_base INTEGER DEFAULT 1 CHECK (difficulty_base BETWEEN 1 AND 5),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS concept_prerequisites (
			concept_id VARCHAR(100) NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
			prerequisite_id VARCHAR(100) NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
			PRIMARY KEY (concept_id, prerequisite_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_concepts_course ON concepts(course_id)`,
		`CREATE INDEX IF NOT EXISTS idx_concept_prereq ON concept_prerequisites(prerequisite_id)`,

		// Student Learning Profile
		`CREATE TABLE IF NOT EXISTS student_profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
			overall_level REAL DEFAULT 0.0 CHECK (overall_level BETWEEN 0.0 AND 1.0),
			total_attempts INTEGER DEFAULT 0,
			correct_attempts INTEGER DEFAULT 0,
			total_hints_used INTEGER DEFAULT 0,
			study_time_seconds INTEGER DEFAULT 0,
			last_active_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_student_profiles_student ON student_profiles(student_id)`,

		// Concept Mastery per Student
		`CREATE TABLE IF NOT EXISTS concept_mastery (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			concept_id VARCHAR(100) NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
			mastery REAL DEFAULT 0.0 CHECK (mastery BETWEEN 0.0 AND 1.0),
			status VARCHAR(20) NOT NULL DEFAULT 'not_started'
				CHECK (status IN ('not_started','learning','developing','mastered')),
			attempts INTEGER DEFAULT 0,
			correct INTEGER DEFAULT 0,
			hints_used INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			last_attempt_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(student_id, concept_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_concept_mastery_student ON concept_mastery(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_concept_mastery_concept ON concept_mastery(concept_id)`,

		// Exercise Bank
		`CREATE TABLE IF NOT EXISTS exercises (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			concept_id VARCHAR(100) NOT NULL REFERENCES concepts(id),
			difficulty INTEGER NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
			statement TEXT NOT NULL,
			latex TEXT DEFAULT '',
			expected_answer TEXT NOT NULL,
			solution TEXT DEFAULT '',
			solution_steps JSONB DEFAULT '[]',
			hints JSONB DEFAULT '[]',
			common_errors JSONB DEFAULT '[]',
			source VARCHAR(20) NOT NULL DEFAULT 'generated'
				CHECK (source IN ('official','generated')),
			generated_by VARCHAR(50) DEFAULT '',
			verified_by_math BOOLEAN DEFAULT false,
			status VARCHAR(20) NOT NULL DEFAULT 'validated'
				CHECK (status IN ('pending','validated','rejected')),
			embedding_id VARCHAR(100) DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_exercises_concept ON exercises(concept_id)`,
		`CREATE INDEX IF NOT EXISTS idx_exercises_difficulty ON exercises(difficulty)`,
		`CREATE INDEX IF NOT EXISTS idx_exercises_source ON exercises(source)`,

		// Tutor Sessions
		`CREATE TABLE IF NOT EXISTS tutor_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
			mode VARCHAR(20) NOT NULL DEFAULT 'tutor'
				CHECK (mode IN ('tutor','practice','review','exam','solve')),
			concept_id VARCHAR(100) DEFAULT '',
			started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			ended_at TIMESTAMP WITH TIME ZONE,
			exercise_count INTEGER DEFAULT 0,
			correct_count INTEGER DEFAULT 0,
			hints_used INTEGER DEFAULT 0,
			total_score REAL DEFAULT 0.0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tutor_sessions_student ON tutor_sessions(student_id)`,

		// Exercise Attempts
		`CREATE TABLE IF NOT EXISTS exercise_attempts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES tutor_sessions(id) ON DELETE CASCADE,
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			exercise_id UUID NOT NULL REFERENCES exercises(id),
			started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE,
			answer TEXT DEFAULT '',
			correct BOOLEAN DEFAULT false,
			score REAL DEFAULT 0.0 CHECK (score BETWEEN 0.0 AND 1.0),
			hints_used INTEGER DEFAULT 0,
			max_hints_used INTEGER DEFAULT 0,
			first_error_step INTEGER DEFAULT 0,
			time_seconds INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_exercise_attempts_session ON exercise_attempts(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_exercise_attempts_student ON exercise_attempts(student_id)`,

		// Step-by-Step Attempts
		`CREATE TABLE IF NOT EXISTS attempt_steps (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			attempt_id UUID NOT NULL REFERENCES exercise_attempts(id) ON DELETE CASCADE,
			step_index INTEGER NOT NULL,
			content TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending'
				CHECK (status IN ('pending','correct','incorrect')),
			error_type VARCHAR(50) DEFAULT '',
			error_detail TEXT DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attempt_steps_attempt ON attempt_steps(attempt_id)`,

		// Error Tracking
		`CREATE TABLE IF NOT EXISTS student_errors (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			concept_id VARCHAR(100) NOT NULL DEFAULT '',
			error_type VARCHAR(50) NOT NULL,
			error_subtype VARCHAR(100) DEFAULT '',
			count INTEGER DEFAULT 1,
			severity VARCHAR(20) DEFAULT 'low'
				CHECK (severity IN ('low','medium','high','critical')),
			last_occurred_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(student_id, concept_id, error_type, error_subtype)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_student_errors_student ON student_errors(student_id)`,

		// Learning Recommendations
		`CREATE TABLE IF NOT EXISTS learning_recommendations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			recommendation_type VARCHAR(50) NOT NULL,
			concept_id VARCHAR(100) DEFAULT '',
			message TEXT NOT NULL,
			priority INTEGER DEFAULT 1,
			dismissed BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_recommendations_student ON learning_recommendations(student_id)`,

		// Seed: Math I concept tree
		`INSERT INTO concepts (id, name, description, parent_id, course_id, difficulty_base) VALUES
		 ('algebra', 'Álgebra', 'Operaciones algebraicas y ecuaciones', NULL, 'matematica-1', 1),
		 ('algebra.operaciones', 'Operaciones algebraicas', 'Suma, resta, multiplicación, división de polinomios', 'algebra', 'matematica-1', 1),
		 ('algebra.factorizacion', 'Factorización', 'Factor común, diferencia de cuadrados, trinomio cuadrado', 'algebra', 'matematica-1', 2),
		 ('algebra.ecuaciones', 'Ecuaciones', 'Ecuaciones lineales y cuadráticas', 'algebra', 'matematica-1', 2),
		 ('funciones', 'Funciones', 'Concepto de función, dominio, imagen', NULL, 'matematica-1', 1),
		 ('funciones.lineal', 'Función lineal', 'f(x) = mx + b, pendiente, intersección', 'funciones', 'matematica-1', 1),
		 ('funciones.cuadratica', 'Función cuadrática', 'f(x) = ax² + bx + c, vértice, raíces', 'funciones', 'matematica-1', 2),
		 ('funciones.composicion', 'Composición de funciones', 'f(g(x)), función compuesta', 'funciones', 'matematica-1', 3),
		 ('limites', 'Límites', 'Concepto y cálculo de límites', NULL, 'matematica-1', 2),
		 ('limites.concepto', 'Concepto de límite', 'Definición intuitiva y formal', 'limites', 'matematica-1', 2),
		 ('limites.propiedades', 'Propiedades de límites', 'Propiedades algebraicas', 'limites', 'matematica-1', 3),
		 ('limites.laterales', 'Límites laterales', 'Límite por la izquierda y derecha', 'limites', 'matematica-1', 3),
		 ('derivadas', 'Derivadas', 'Cálculo diferencial', NULL, 'matematica-1', 3),
		 ('derivadas.definicion', 'Definición de derivada', 'Límite de la razón de incremento', 'derivadas', 'matematica-1', 3),
		 ('derivadas.potencia', 'Regla de la potencia', 'd/dx(x^n) = n·x^(n-1)', 'derivadas', 'matematica-1', 3),
		 ('derivadas.producto', 'Regla del producto', 'd/dx(f·g) = f''·g + f·g''', 'derivadas', 'matematica-1', 4),
		 ('derivadas.cociente', 'Regla del cociente', 'd/dx(f/g) = (f''·g - f·g'') / g²', 'derivadas', 'matematica-1', 4),
		 ('derivadas.cadena', 'Regla de la cadena', 'd/dx(f(g(x))) = f''(g(x))·g''(x)', 'derivadas', 'matematica-1', 4),
		 ('integrales', 'Integrales', 'Cálculo integral', NULL, 'matematica-1', 4),
		 ('integrales.indefinida', 'Integral indefinida', 'Antiderivada, familia de funciones', 'integrales', 'matematica-1', 4),
		 ('integrales.definida', 'Integral definida', 'Área bajo la curva, teorema fundamental', 'integrales', 'matematica-1', 4),
		 ('integrales.sustitucion', 'Sustitución', 'Cambio de variable', 'integrales', 'matematica-1', 5),
		 ('integrales.partes', 'Integración por partes', '∫u·dv = u·v - ∫v·du', 'integrales', 'matematica-1', 5)
		 ON CONFLICT (id) DO NOTHING`,

		// Seed: Prerequisites
		`INSERT INTO concept_prerequisites (concept_id, prerequisite_id) VALUES
		 ('algebra.operaciones', 'algebra'),
		 ('algebra.factorizacion', 'algebra.operaciones'),
		 ('algebra.ecuaciones', 'algebra.operaciones'),
		 ('funciones.lineal', 'funciones'),
		 ('funciones.cuadratica', 'funciones.lineal'),
		 ('funciones.composicion', 'funciones.cuadratica'),
		 ('limites.concepto', 'funciones'),
		 ('limites.propiedades', 'limites.concepto'),
		 ('limites.laterales', 'limites.propiedades'),
		 ('derivadas.definicion', 'limites.concepto'),
		 ('derivadas.potencia', 'derivadas.definicion'),
		 ('derivadas.producto', 'derivadas.potencia'),
		 ('derivadas.cociente', 'derivadas.producto'),
		 ('derivadas.cadena', 'derivadas.potencia'),
		 ('derivadas.cadena', 'funciones.composicion'),
		 ('integrales.indefinida', 'derivadas.potencia'),
		 ('integrales.definida', 'integrales.indefinida'),
		 ('integrales.definida', 'limites.concepto'),
		 ('integrales.sustitucion', 'integrales.indefinida'),
		 ('integrales.partes', 'integrales.indefinida'),
		 ('integrales.partes', 'derivadas.producto')
		 ON CONFLICT DO NOTHING`,

		// Assessment System
		`CREATE TABLE IF NOT EXISTS assessments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
			assessment_type VARCHAR(30) NOT NULL DEFAULT 'formative'
				CHECK (assessment_type IN ('diagnostic','formative','summative','recovery','practice')),
			mode VARCHAR(20) NOT NULL DEFAULT 'fixed'
				CHECK (mode IN ('fixed','generated','adaptive')),
			time_limit_minutes INTEGER DEFAULT 0,
			max_attempts INTEGER DEFAULT 1,
			shuffle_questions BOOLEAN DEFAULT true,
			show_results BOOLEAN DEFAULT true,
			show_solutions BOOLEAN DEFAULT false,
			passing_score REAL DEFAULT 0.6 CHECK (passing_score BETWEEN 0.0 AND 1.0),
			total_points INTEGER DEFAULT 100,
			created_by UUID REFERENCES users(id),
			status VARCHAR(20) NOT NULL DEFAULT 'draft'
				CHECK (status IN ('draft','published','archived')),
			published_at TIMESTAMP WITH TIME ZONE,
			expires_at TIMESTAMP WITH TIME ZONE,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_assessments_course ON assessments(course_id)`,
		`CREATE INDEX IF NOT EXISTS idx_assessments_type ON assessments(assessment_type)`,
		`CREATE INDEX IF NOT EXISTS idx_assessments_status ON assessments(status)`,

		`CREATE TABLE IF NOT EXISTS assessment_questions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
			exercise_id UUID REFERENCES exercises(id),
			question_order INTEGER NOT NULL DEFAULT 0,
			points INTEGER NOT NULL DEFAULT 10,
			question_type VARCHAR(20) NOT NULL DEFAULT 'exercise'
				CHECK (question_type IN ('exercise','generated','text','multiple_choice')),
			statement_override TEXT DEFAULT '',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_assessment_questions_assessment ON assessment_questions(assessment_id)`,

		`CREATE TABLE IF NOT EXISTS rubrics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			rubric_type VARCHAR(20) NOT NULL DEFAULT 'analytic'
				CHECK (rubric_type IN ('analytic','holistic')),
			max_score REAL NOT NULL DEFAULT 1.0,
			criteria JSONB NOT NULL DEFAULT '[]',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rubrics_assessment ON rubrics(assessment_id)`,

		`CREATE TABLE IF NOT EXISTS student_assessments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL DEFAULT 'in_progress'
				CHECK (status IN ('in_progress','submitted','graded','returned')),
			started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			submitted_at TIMESTAMP WITH TIME ZONE,
			graded_at TIMESTAMP WITH TIME ZONE,
			time_spent_seconds INTEGER DEFAULT 0,
			attempt_number INTEGER DEFAULT 1,
			total_score REAL DEFAULT 0.0,
			max_score REAL DEFAULT 0.0,
			percentage REAL DEFAULT 0.0,
			passed BOOLEAN DEFAULT false,
			graded_by UUID REFERENCES users(id),
			feedback TEXT DEFAULT '',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(student_id, assessment_id, attempt_number)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_student_assessments_student ON student_assessments(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_student_assessments_assessment ON student_assessments(assessment_id)`,

		`CREATE TABLE IF NOT EXISTS student_answers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_assessment_id UUID NOT NULL REFERENCES student_assessments(id) ON DELETE CASCADE,
			question_id UUID NOT NULL REFERENCES assessment_questions(id) ON DELETE CASCADE,
			answer TEXT DEFAULT '',
			procedure JSONB DEFAULT '[]',
			is_correct BOOLEAN DEFAULT false,
			score REAL DEFAULT 0.0 CHECK (score BETWEEN 0.0 AND 1.0),
			points_earned REAL DEFAULT 0.0,
			points_possible INTEGER DEFAULT 10,
			math_verified BOOLEAN DEFAULT false,
			rubric_scores JSONB DEFAULT '{}',
			feedback TEXT DEFAULT '',
			time_spent_seconds INTEGER DEFAULT 0,
			submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			graded_at TIMESTAMP WITH TIME ZONE,
			grading_method VARCHAR(20) DEFAULT 'auto'
				CHECK (grading_method IN ('auto','manual','hybrid'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_student_answers_assessment ON student_answers(student_assessment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_student_answers_question ON student_answers(question_id)`,

		`CREATE TABLE IF NOT EXISTS student_analytics (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
			total_assessments INTEGER DEFAULT 0,
			passed_assessments INTEGER DEFAULT 0,
			average_score REAL DEFAULT 0.0,
			average_time_seconds INTEGER DEFAULT 0,
			competency_level VARCHAR(20) DEFAULT 'beginner'
				CHECK (competency_level IN ('beginner','developing','proficient','advanced','exceptional')),
			competency_score REAL DEFAULT 0.0,
			weakest_concepts JSONB DEFAULT '[]',
			strongest_concepts JSONB DEFAULT '[]',
			improvement_trend REAL DEFAULT 0.0,
			study_streak_days INTEGER DEFAULT 0,
			last_assessment_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(student_id, course_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_student_analytics_student ON student_analytics(student_id)`,

		`CREATE TABLE IF NOT EXISTS recovery_plans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
			trigger_assessment_id UUID REFERENCES assessments(id),
			trigger_score REAL DEFAULT 0.0,
			status VARCHAR(20) NOT NULL DEFAULT 'active'
				CHECK (status IN ('active','completed','expired','cancelled')),
			priority INTEGER DEFAULT 1 CHECK (priority BETWEEN 1 AND 5),
			concepts_to_review JSONB DEFAULT '[]',
			recommended_activities JSONB DEFAULT '[]',
			target_date TIMESTAMP WITH TIME ZONE,
			completed_at TIMESTAMP WITH TIME ZONE,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_recovery_plans_student ON recovery_plans(student_id)`,

		`CREATE TABLE IF NOT EXISTS academic_alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			alert_type VARCHAR(50) NOT NULL,
			severity VARCHAR(20) NOT NULL DEFAULT 'warning'
				CHECK (severity IN ('info','warning','critical')),
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			concept_id VARCHAR(100) DEFAULT '',
			assessment_id UUID REFERENCES assessments(id),
			acknowledged BOOLEAN DEFAULT false,
			acknowledged_by UUID REFERENCES users(id),
			acknowledged_at TIMESTAMP WITH TIME ZONE,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_academic_alerts_student ON academic_alerts(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_academic_alerts_type ON academic_alerts(alert_type)`,
		`CREATE INDEX IF NOT EXISTS idx_academic_alerts_severity ON academic_alerts(severity)`,

		`CREATE TABLE IF NOT EXISTS assessment_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_assessment_id UUID NOT NULL REFERENCES student_assessments(id) ON DELETE CASCADE,
			answers_snapshot JSONB NOT NULL DEFAULT '{}',
			current_question_index INT DEFAULT 0,
			time_spent_seconds INT DEFAULT 0,
			last_saved_at TIMESTAMPTZ DEFAULT NOW(),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(student_assessment_id)
		)`,

		`CREATE TABLE IF NOT EXISTS audit_log_entries (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_type VARCHAR(50) NOT NULL,
			entity_id UUID NOT NULL,
			action VARCHAR(50) NOT NULL,
			user_id UUID REFERENCES users(id),
			old_value JSONB,
			new_value JSONB,
			version INT,
			ip_address INET,
			user_agent TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS question_bank (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			statement TEXT NOT NULL,
			latex TEXT,
			question_type VARCHAR(50) NOT NULL DEFAULT 'exercise',
			difficulty INT DEFAULT 1 CHECK (difficulty BETWEEN 1 AND 5),
			concept_id VARCHAR(100),
			competencies TEXT[] DEFAULT '{}',
			expected_answer TEXT,
			answer_options JSONB,
			explanation TEXT,
			explanation_latex TEXT,
			tags TEXT[] DEFAULT '{}',
			source VARCHAR(50) DEFAULT 'manual',
			created_by UUID REFERENCES users(id),
			validated_by_math BOOLEAN DEFAULT FALSE,
			version INT DEFAULT 1,
			is_active BOOLEAN DEFAULT TRUE,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		// Agent System
		`CREATE TABLE IF NOT EXISTS agent_execution_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id),
			session_id UUID,
			intent TEXT NOT NULL,
			plan JSONB,
			tools_used TEXT[],
			tool_results JSONB,
			final_response TEXT,
			duration_ms INTEGER,
			total_tokens INTEGER,
			model TEXT,
			status TEXT DEFAULT 'completed',
			error TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS agent_sessions (
			session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id),
			course_id VARCHAR(100) NOT NULL DEFAULT 'matematica-1',
			intent TEXT DEFAULT '',
			current_concept TEXT DEFAULT '',
			current_exercise_id UUID,
			messages JSONB DEFAULT '[]',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_execution_log_student ON agent_execution_log(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_execution_log_created ON agent_execution_log(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_sessions_student ON agent_sessions(student_id)`,

		// Adaptive Learning Engine — Fase 6
		`CREATE TABLE IF NOT EXISTS learning_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id),
			course_id VARCHAR(100) NOT NULL,
			concept_id VARCHAR(255) NOT NULL,
			activity_id UUID,
			event_type VARCHAR(50) NOT NULL,
			difficulty INT DEFAULT 1,
			correct BOOLEAN,
			score FLOAT DEFAULT 0,
			time_seconds INT DEFAULT 0,
			hints_used INT DEFAULT 0,
			error_type VARCHAR(50),
			error_detail TEXT,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_events_student ON learning_events(student_id, concept_id)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_events_type ON learning_events(event_type)`,
		`CREATE TABLE IF NOT EXISTS mastery_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			student_id UUID NOT NULL REFERENCES users(id),
			concept_id VARCHAR(255) NOT NULL,
			old_mastery FLOAT NOT NULL,
			new_mastery FLOAT NOT NULL,
			trigger_event_id UUID,
			reason TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mastery_history_student ON mastery_history(student_id, concept_id)`,
		`ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS independent_successes INT DEFAULT 0`,
		`ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS average_time_seconds INT DEFAULT 0`,
		`ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMPTZ`,
		`ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMPTZ`,
		`ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS next_review_at TIMESTAMPTZ`,
		`ALTER TABLE concept_mastery ADD COLUMN IF NOT EXISTS confidence FLOAT DEFAULT 1.0`,
		`ALTER TABLE student_errors ADD COLUMN IF NOT EXISTS course_id VARCHAR(100)`,
		`ALTER TABLE student_errors ADD COLUMN IF NOT EXISTS resolved BOOLEAN DEFAULT false`,
		`ALTER TABLE student_errors ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ`,
		`ALTER TABLE student_errors ADD COLUMN IF NOT EXISTS attempt_id UUID`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w\nQuery: %s", err, m[:50])
		}
	}

	log.Println("Migrations completed")
	return nil
}
