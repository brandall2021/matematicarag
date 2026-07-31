package api

import (
	"strings"
	"testing"
)

func TestCanManageDocuments(t *testing.T) {
	cases := map[string]bool{
		"ADMIN":   true,
		"TEACHER": true,
		"STUDENT": false,
		"":        false,
		"GUEST":   false,
	}
	for role, want := range cases {
		if got := canManageDocuments(role); got != want {
			t.Errorf("canManageDocuments(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestDocumentsListQuery(t *testing.T) {
	studentQuery := documentsListQuery("STUDENT")
	if strings.Contains(studentQuery, "WHERE uploaded_by") {
		t.Errorf("student query should not filter by owner, got: %s", studentQuery)
	}
	teacherQuery := documentsListQuery("TEACHER")
	if !strings.Contains(teacherQuery, "WHERE uploaded_by = $1") {
		t.Errorf("teacher query should filter by owner, got: %s", teacherQuery)
	}
}
