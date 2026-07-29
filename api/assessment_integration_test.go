package api

import (
	"encoding/json"
	"testing"
)

func TestQuestionCRUD(t *testing.T) {
	// Create
	q := Question{
		ID:               "q-test-001",
		Statement:        "What is 2+2?",
		QuestionType:     "multiple_choice",
		Difficulty:       2,
		Competencies:     []string{"arithmetic"},
		Tags:             []string{"basic", "addition"},
		Source:           "teacher",
		IsActive:         true,
		Version:          1,
		ValidatedByMath:  false,
		AnswerOptions:    json.RawMessage(`[{"id":"a","text":"4"},{"id":"b","text":"5"},{"id":"c","text":"3"}]`),
		ExpectedAnswer:   "a",
	}

	// Read - verify all fields preserved after creation
	if q.ID != "q-test-001" {
		t.Errorf("expected ID 'q-test-001', got '%s'", q.ID)
	}
	if q.Statement != "What is 2+2?" {
		t.Errorf("expected statement 'What is 2+2?', got '%s'", q.Statement)
	}
	if q.QuestionType != "multiple_choice" {
		t.Errorf("expected question_type 'multiple_choice', got '%s'", q.QuestionType)
	}
	if q.Difficulty != 2 {
		t.Errorf("expected difficulty 2, got %d", q.Difficulty)
	}
	if !q.IsActive {
		t.Error("expected question to be active on creation")
	}
	if q.Version != 1 {
		t.Errorf("expected version 1, got %d", q.Version)
	}
	if len(q.Competencies) != 1 || q.Competencies[0] != "arithmetic" {
		t.Errorf("expected competencies [arithmetic], got %v", q.Competencies)
	}
	if len(q.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(q.Tags))
	}
	if q.AnswerOptions == nil {
		t.Error("expected answer_options to be set")
	}

	// Update
	q.Statement = "What is 3+3?"
	q.Difficulty = 3
	q.AnswerOptions = json.RawMessage(`[{"id":"a","text":"6"},{"id":"b","text":"7"}]`)
	q.ExpectedAnswer = "a"
	q.Version++

	if q.Statement != "What is 3+3?" {
		t.Errorf("expected updated statement 'What is 3+3?', got '%s'", q.Statement)
	}
	if q.Difficulty != 3 {
		t.Errorf("expected updated difficulty 3, got %d", q.Difficulty)
	}
	if q.Version != 2 {
		t.Errorf("expected version 2 after update, got %d", q.Version)
	}

	var options []map[string]string
	if err := json.Unmarshal(q.AnswerOptions, &options); err != nil {
		t.Fatalf("failed to unmarshal updated answer_options: %v", err)
	}
	if len(options) != 2 {
		t.Errorf("expected 2 answer options after update, got %d", len(options))
	}

	// Soft delete
	q.IsActive = false
	if q.IsActive {
		t.Error("expected question to be inactive after soft delete")
	}
	if q.ID != "q-test-001" {
		t.Error("question ID should be preserved after soft delete")
	}
	if q.Statement != "What is 3+3?" {
		t.Error("question statement should be preserved after soft delete")
	}
}

func TestQuestionValidation(t *testing.T) {
	t.Run("multiple_choice without options fails", func(t *testing.T) {
		q := Question{
			QuestionType: "multiple_choice",
			Statement:    "What is 2+2?",
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for multiple choice without answer_options")
		}
	})

	t.Run("multiple_choice with single option fails", func(t *testing.T) {
		q := Question{
			QuestionType: "multiple_choice",
			Statement:    "What is 2+2?",
			AnswerOptions: json.RawMessage(`[{"id":"a","text":"4"}]`),
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for single answer option")
		}
	})

	t.Run("multiple_choice with 2+ options passes", func(t *testing.T) {
		q := Question{
			QuestionType: "multiple_choice",
			Statement:    "What is 2+2?",
			AnswerOptions: json.RawMessage(`[{"id":"a","text":"4"},{"id":"b","text":"5"}]`),
		}
		result := ValidateQuestionByType(q)
		if result != "" {
			t.Errorf("unexpected error: %s", result)
		}
	})

	t.Run("multiple_choice with invalid JSON fails", func(t *testing.T) {
		q := Question{
			QuestionType: "multiple_choice",
			Statement:    "What is 2+2?",
			AnswerOptions: json.RawMessage(`not json`),
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for invalid JSON in answer_options")
		}
	})

	t.Run("multiple_choice with empty array fails", func(t *testing.T) {
		q := Question{
			QuestionType: "multiple_choice",
			Statement:    "What is 2+2?",
			AnswerOptions: json.RawMessage(`[]`),
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for empty answer_options array")
		}
	})

	t.Run("numeric without expected_answer fails", func(t *testing.T) {
		q := Question{
			QuestionType: "numeric",
			Statement:    "Calculate 5 * 3",
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for numeric without expected_answer")
		}
	})

	t.Run("numeric with expected_answer passes", func(t *testing.T) {
		q := Question{
			QuestionType:   "numeric",
			Statement:      "Calculate 5 * 3",
			ExpectedAnswer: "15",
		}
		result := ValidateQuestionByType(q)
		if result != "" {
			t.Errorf("unexpected error: %s", result)
		}
	})

	t.Run("exercise type passes", func(t *testing.T) {
		q := Question{
			QuestionType: "exercise",
			Statement:    "Solve the following equation step by step",
		}
		result := ValidateQuestionByType(q)
		if result != "" {
			t.Errorf("exercise type should pass validation, got: %s", result)
		}
	})

	t.Run("true_false without options fails", func(t *testing.T) {
		q := Question{
			QuestionType: "true_false",
			Statement:    "The sky is blue",
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for true_false without answer_options")
		}
	})

	t.Run("true_false with options passes", func(t *testing.T) {
		q := Question{
			QuestionType: "true_false",
			Statement:    "The sky is blue",
			AnswerOptions: json.RawMessage(`[{"id":"a","text":"True"},{"id":"b","text":"False"}]`),
		}
		result := ValidateQuestionByType(q)
		if result != "" {
			t.Errorf("unexpected error: %s", result)
		}
	})

	t.Run("algebraic_expression without expected_answer fails", func(t *testing.T) {
		q := Question{
			QuestionType: "algebraic_expression",
			Statement:    "Simplify: 2x + 3x",
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for algebraic_expression without expected_answer")
		}
	})

	t.Run("equation without expected_answer fails", func(t *testing.T) {
		q := Question{
			QuestionType: "equation",
			Statement:    "Solve for x: 2x + 4 = 10",
		}
		result := ValidateQuestionByType(q)
		if result == "" {
			t.Error("expected error for equation without expected_answer")
		}
	})
}

func TestAdaptiveAssessment(t *testing.T) {
	t.Run("level increases after correct answers", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 3,
			Performance:  0,
		}

		// 3 correct answers → ratio 1.0 → level increases from 3 to 4
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(true)
		}
		if aa.CurrentLevel != 4 {
			t.Errorf("expected level 4 after 3 correct answers, got %d", aa.CurrentLevel)
		}
		if aa.CorrectAnswers != 3 {
			t.Errorf("expected 3 correct answers, got %d", aa.CorrectAnswers)
		}
		if aa.QuestionsAnswered != 3 {
			t.Errorf("expected 3 questions answered, got %d", aa.QuestionsAnswered)
		}
	})

	t.Run("level decreases after wrong answers", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 3,
			Performance:  0,
		}

		// 3 wrong answers → ratio 0.0 → level decreases from 3 to 2
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(false)
		}
		if aa.CurrentLevel != 2 {
			t.Errorf("expected level 2 after 3 wrong answers, got %d", aa.CurrentLevel)
		}
		if aa.CorrectAnswers != 0 {
			t.Errorf("expected 0 correct answers, got %d", aa.CorrectAnswers)
		}
	})

	t.Run("level does not exceed 5", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 5,
			Performance:  0,
		}

		// 3 correct answers → should stay at 5
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(true)
		}
		if aa.CurrentLevel != 5 {
			t.Errorf("expected level 5 (max), got %d", aa.CurrentLevel)
		}
	})

	t.Run("level does not go below 1", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 1,
			Performance:  0,
		}

		// 3 wrong answers → should stay at 1
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(false)
		}
		if aa.CurrentLevel != 1 {
			t.Errorf("expected level 1 (min), got %d", aa.CurrentLevel)
		}
	})

	t.Run("performance calculation", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 3,
			Performance:  0,
		}

		aa.UpdatePerformance(true)
		aa.UpdatePerformance(false)
		aa.UpdatePerformance(true)
		aa.UpdatePerformance(false)

		expectedPerformance := 0.5
		if aa.Performance != expectedPerformance {
			t.Errorf("expected performance %f, got %f", expectedPerformance, aa.Performance)
		}
		if aa.CorrectAnswers != 2 {
			t.Errorf("expected 2 correct answers, got %d", aa.CorrectAnswers)
		}
		if aa.QuestionsAnswered != 4 {
			t.Errorf("expected 4 questions answered, got %d", aa.QuestionsAnswered)
		}
	})

	t.Run("level increases across multiple rounds", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 3,
			Performance:  0,
		}

		// 3 correct → level 4
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(true)
		}
		if aa.CurrentLevel != 4 {
			t.Errorf("expected level 4, got %d", aa.CurrentLevel)
		}

		// 3 more correct → level 5
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(true)
		}
		if aa.CurrentLevel != 5 {
			t.Errorf("expected level 5, got %d", aa.CurrentLevel)
		}

		// Even more correct → stays at 5
		aa.UpdatePerformance(true)
		if aa.CurrentLevel != 5 {
			t.Errorf("expected level 5 (max boundary), got %d", aa.CurrentLevel)
		}
	})

	t.Run("level decreases across multiple rounds", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 3,
			Performance:  0,
		}

		// 3 wrong → level 2
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(false)
		}
		if aa.CurrentLevel != 2 {
			t.Errorf("expected level 2, got %d", aa.CurrentLevel)
		}

		// 3 more wrong → level 1
		for i := 0; i < 3; i++ {
			aa.UpdatePerformance(false)
		}
		if aa.CurrentLevel != 1 {
			t.Errorf("expected level 1, got %d", aa.CurrentLevel)
		}

		// More wrong → stays at 1
		aa.UpdatePerformance(false)
		if aa.CurrentLevel != 1 {
			t.Errorf("expected level 1 (min boundary), got %d", aa.CurrentLevel)
		}
	})

	t.Run("target_level bounded correctly", func(t *testing.T) {
		aa := &AdaptiveAssessment{
			StudentID:    "student-1",
			AssessmentID: "assess-1",
			CurrentLevel: 5,
			Performance:  0,
		}

		aa.UpdatePerformance(true)
		if aa.TargetLevel != 5 {
			t.Errorf("expected TargetLevel 5 (max) when at level 5 correct, got %d", aa.TargetLevel)
		}

		aa.TargetLevel = 5
		aa.UpdatePerformance(false)
		if aa.TargetLevel != 4 {
			t.Errorf("expected TargetLevel 4 when at level 5 wrong, got %d", aa.TargetLevel)
		}

		aa.CurrentLevel = 1
		aa.UpdatePerformance(false)
		if aa.TargetLevel != 1 {
			t.Errorf("expected TargetLevel 1 (min) when at level 1 wrong, got %d", aa.TargetLevel)
		}
	})
}
