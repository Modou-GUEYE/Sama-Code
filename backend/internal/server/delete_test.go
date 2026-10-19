package server

import (
	"context"
	"net/http"
	"testing"
)

func TestDeleteQuestionsAndSeries(t *testing.T) {
	e := newAdminEnv(t)
	editor, validator := e.user("editor"), e.user("validator")

	// Brouillon : l'éditeur peut le supprimer.
	status, body := editor.do("POST", "/api/v1/admin/questions", question("brouillon"))
	mustStatus(t, status, body, http.StatusCreated, "création")
	draft := body["id"].(string)
	status, body = editor.do("DELETE", "/api/v1/admin/questions/"+draft, nil)
	mustStatus(t, status, body, http.StatusNoContent, "suppression d'un brouillon")
	status, body = editor.do("GET", "/api/v1/admin/questions/"+draft, nil)
	mustStatus(t, status, body, http.StatusNotFound, "question supprimée")

	// Série publiée, passée par un candidat.
	seriesID, optA, _, order := publishedSeries(t, e, 3)
	q := order[0]

	// Question publiée : l'éditeur ne peut pas ; dans une série : refusé même au validateur.
	status, body = editor.do("DELETE", "/api/v1/admin/questions/"+q, nil)
	mustStatus(t, status, body, http.StatusForbidden, "éditeur sur question publiée")
	status, body = validator.do("DELETE", "/api/v1/admin/questions/"+q, nil)
	mustStatus(t, status, body, http.StatusConflict, "question dans une série")

	candidate := e.user("candidate")
	status, body = candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": e.licenseID("C"), "series_id": seriesID})
	mustStatus(t, status, body, http.StatusCreated, "examen")
	sid := body["id"].(string)
	candidate.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+q, map[string]any{"option_ids": []string{optA[q]}})
	candidate.do("POST", "/api/v1/exam-sessions/"+sid+"/submit", nil)

	// Série passée par un candidat : suppression refusée, l'historique reste lisible.
	status, body = validator.do("DELETE", "/api/v1/admin/exam-series/"+seriesID, nil)
	mustStatus(t, status, body, http.StatusConflict, "série déjà passée")

	// Retirée de la série, la question reste protégée car un candidat y a répondu.
	validator.do("PUT", "/api/v1/admin/exam-series/"+seriesID+"/questions", map[string]any{"question_ids": order[1:]})
	status, body = validator.do("DELETE", "/api/v1/admin/questions/"+q, nil)
	mustStatus(t, status, body, http.StatusConflict, "question déjà répondue")
	status, body = candidate.do("GET", "/api/v1/exam-sessions/"+sid+"/review", nil)
	mustStatus(t, status, body, http.StatusOK, "correction toujours disponible")

	// Série jamais passée (même publiée) : le validateur peut la supprimer, versions comprises.
	other, _, _, _ := publishedSeries(t, e, 4)
	status, body = editor.do("DELETE", "/api/v1/admin/exam-series/"+other, nil)
	mustStatus(t, status, body, http.StatusForbidden, "éditeur sur série publiée")
	status, body = validator.do("DELETE", "/api/v1/admin/exam-series/"+other, nil)
	mustStatus(t, status, body, http.StatusNoContent, "suppression d'une série jamais passée")
	var snapshots int
	e.db.QueryRow(context.Background(), `SELECT count(*) FROM exam_series_snapshots WHERE series_id = $1`, other).Scan(&snapshots)
	if snapshots != 0 {
		t.Errorf("instantanés restants = %d", snapshots)
	}

	var deletions int
	e.db.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE action LIKE '%.deleted'`).Scan(&deletions)
	if deletions != 2 {
		t.Errorf("suppressions journalisées = %d ; attendu 2", deletions)
	}
}

func TestDeleteCourses(t *testing.T) {
	e := newAdminEnv(t)
	editor, validator := e.user("editor"), e.user("validator")
	tr := func(title string) map[string]any {
		return map[string]any{"fr": map[string]any{"title": title}}
	}

	status, body := editor.do("POST", "/api/v1/admin/course-categories", map[string]any{"code": "panneaux", "translations": tr("Panneaux")})
	mustStatus(t, status, body, http.StatusCreated, "thème")
	cat := body["id"].(string)
	status, body = editor.do("POST", "/api/v1/admin/courses", map[string]any{"category_id": cat, "translations": tr("Les panneaux de danger")})
	mustStatus(t, status, body, http.StatusCreated, "cours")
	course := body["id"].(string)
	status, body = editor.do("POST", "/api/v1/admin/courses/"+course+"/lessons", map[string]any{"translations": tr("Le triangle rouge")})
	mustStatus(t, status, body, http.StatusCreated, "leçon")
	lesson := body["id"].(string)
	text := "Le triangle à bord rouge annonce un danger."
	status, body = editor.do("PUT", "/api/v1/admin/lessons/"+lesson+"/contents", map[string]any{
		"fr": []map[string]any{{"content_type": "text", "text": text}},
	})
	mustStatus(t, status, body, http.StatusOK, "contenu de la leçon")

	// Publier la leçon puis le cours (circuit complet).
	for _, path := range []string{"/api/v1/admin/lessons/" + lesson, "/api/v1/admin/courses/" + course} {
		status, body = editor.do("POST", path+"/status", map[string]string{"status": "in_review"})
		mustStatus(t, status, body, http.StatusOK, path+" → in_review")
		status, body = validator.do("POST", path+"/status", map[string]string{"status": "published"})
		mustStatus(t, status, body, http.StatusOK, path+" → published")
	}

	// Thème non vide : refusé. Thème : réservé au validateur.
	status, body = validator.do("DELETE", "/api/v1/admin/course-categories/"+cat, nil)
	mustStatus(t, status, body, http.StatusConflict, "thème non vide")
	status, body = editor.do("DELETE", "/api/v1/admin/courses/"+course, nil)
	mustStatus(t, status, body, http.StatusForbidden, "éditeur sur cours publié")

	// Supprimer le cours supprime aussi ses leçons ; le thème vide peut alors partir.
	status, body = validator.do("DELETE", "/api/v1/admin/courses/"+course, nil)
	mustStatus(t, status, body, http.StatusNoContent, "suppression du cours")
	status, body = editor.do("GET", "/api/v1/admin/lessons/"+lesson, nil)
	mustStatus(t, status, body, http.StatusNotFound, "leçon supprimée avec le cours")
	status, body = editor.do("DELETE", "/api/v1/admin/course-categories/"+cat, nil)
	mustStatus(t, status, body, http.StatusForbidden, "éditeur sur thème")
	status, body = validator.do("DELETE", "/api/v1/admin/course-categories/"+cat, nil)
	mustStatus(t, status, body, http.StatusNoContent, "suppression du thème vide")
}
