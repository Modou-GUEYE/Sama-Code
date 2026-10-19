package server

import (
	"net/http"
	"testing"
)

func TestCandidateCourses(t *testing.T) {
	e := newAdminEnv(t)
	editor, validator := e.user("editor"), e.user("validator")
	candidate := e.user("candidate")
	tr := func(title string) map[string]any {
		return map[string]any{"fr": map[string]any{"title": title}}
	}
	publish := func(path string) {
		t.Helper()
		status, body := editor.do("POST", path+"/status", map[string]string{"status": "in_review"})
		mustStatus(t, status, body, http.StatusOK, path+" → in_review")
		status, body = validator.do("POST", path+"/status", map[string]string{"status": "published"})
		mustStatus(t, status, body, http.StatusOK, path+" → published")
	}
	lesson := func(course, title string) string {
		t.Helper()
		status, body := editor.do("POST", "/api/v1/admin/courses/"+course+"/lessons", map[string]any{"translations": tr(title)})
		mustStatus(t, status, body, http.StatusCreated, "leçon")
		id := body["id"].(string)
		editor.do("PUT", "/api/v1/admin/lessons/"+id+"/contents", map[string]any{
			"fr": []map[string]any{{"content_type": "text", "text": title + " : contenu"}},
		})
		return id
	}

	// D25 : en wolof, pas de titre écrit ni de bloc texte, seulement de l'audio.
	status, body := editor.do("POST", "/api/v1/admin/course-categories", map[string]any{"code": "x",
		"translations": map[string]any{"fr": map[string]any{"title": "X"}, "wo": map[string]any{"title": "X (wo)"}}})
	mustStatus(t, status, body, http.StatusBadRequest, "titre wolof")

	_, body = editor.do("POST", "/api/v1/admin/course-categories", map[string]any{"code": "feux", "translations": tr("Les feux")})
	cat := body["id"].(string)
	_, body = editor.do("POST", "/api/v1/admin/courses", map[string]any{"category_id": cat, "translations": tr("Feux tricolores")})
	course := body["id"].(string)
	l1, l2, l3 := lesson(course, "Le feu rouge"), lesson(course, "Le feu orange"), lesson(course, "Brouillon")
	status, body = editor.do("PUT", "/api/v1/admin/lessons/"+l1+"/contents", map[string]any{
		"wo": []map[string]any{{"content_type": "text", "text": "texte wolof"}},
	})
	mustStatus(t, status, body, http.StatusBadRequest, "bloc texte en wolof")
	publish("/api/v1/admin/lessons/" + l1)
	publish("/api/v1/admin/lessons/" + l2)

	// Cours encore en brouillon : invisible pour le candidat.
	status, body = candidate.do("GET", "/api/v1/course-categories", nil)
	mustStatus(t, status, body, http.StatusOK, "catalogue")
	if len(body["items"].([]any)) != 0 {
		t.Fatalf("un cours non publié est visible : %v", body)
	}
	publish("/api/v1/admin/courses/" + course)

	status, body = candidate.do("GET", "/api/v1/course-categories", nil)
	mustStatus(t, status, body, http.StatusOK, "catalogue")
	c := body["items"].([]any)[0].(map[string]any)["courses"].([]any)[0].(map[string]any)
	if c["lesson_count"] != 2.0 || c["completed_lessons"] != 0.0 {
		t.Fatalf("cours = %v (la leçon en brouillon ne doit pas compter)", c)
	}

	// Leçon en brouillon : introuvable ; leçon publiée : contenu fr/wo et leçon suivante.
	status, body = candidate.do("GET", "/api/v1/lessons/"+l3, nil)
	mustStatus(t, status, body, http.StatusNotFound, "leçon en brouillon")
	status, body = candidate.do("GET", "/api/v1/lessons/"+l1, nil)
	mustStatus(t, status, body, http.StatusOK, "leçon")
	if body["next_lesson_id"] != l2 || len(body["blocks"].(map[string]any)["fr"].([]any)) != 1 || body["completed"] != false {
		t.Fatalf("leçon = %v", body)
	}

	// Terminer (deux fois : idempotent), puis la progression du cours.
	for range 2 {
		status, body = candidate.do("POST", "/api/v1/lessons/"+l1+"/completion", nil)
		mustStatus(t, status, body, http.StatusNoContent, "leçon terminée")
	}
	status, body = candidate.do("GET", "/api/v1/courses/"+course, nil)
	mustStatus(t, status, body, http.StatusOK, "cours")
	lessons := body["lessons"].([]any)
	if body["completed_lessons"] != 1.0 || len(lessons) != 2 || lessons[0].(map[string]any)["completed"] != true {
		t.Fatalf("progression = %v", body)
	}
	status, body = candidate.do("POST", "/api/v1/lessons/"+l3+"/completion", nil)
	mustStatus(t, status, body, http.StatusNotFound, "terminer une leçon non publiée")
}
