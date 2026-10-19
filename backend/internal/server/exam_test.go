package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// publishedSeries publie une série complète du permis C (10 questions, bonne réponse A)
// et renvoie son id ainsi que, par question, les ids des propositions A et B.
func publishedSeries(t *testing.T, e *adminEnv, number int) (seriesID string, optA, optB map[string]string, order []string) {
	t.Helper()
	editor := e.user("editor")
	validator := e.user("validator")
	optA, optB = map[string]string{}, map[string]string{}
	for i := range 10 {
		status, body := editor.do("POST", "/api/v1/admin/questions", question(fmt.Sprintf("s%d-%d", number, i)))
		mustStatus(t, status, body, http.StatusCreated, "question")
		id := body["id"].(string)
		for _, o := range body["options"].([]any) {
			m := o.(map[string]any)
			if m["code"] == "A" {
				optA[id] = m["id"].(string)
			} else {
				optB[id] = m["id"].(string)
			}
		}
		editor.do("POST", "/api/v1/admin/questions/"+id+"/status", map[string]string{"status": "in_review"})
		validator.do("POST", "/api/v1/admin/questions/"+id+"/status", map[string]string{"status": "published"})
		order = append(order, id)
	}
	status, body := editor.do("POST", "/api/v1/admin/exam-series", map[string]any{"license_category_id": e.licenseID("C"), "series_number": number})
	mustStatus(t, status, body, http.StatusCreated, "série")
	seriesID = body["id"].(string)
	editor.do("PUT", "/api/v1/admin/exam-series/"+seriesID+"/questions", map[string]any{"question_ids": order})
	editor.do("POST", "/api/v1/admin/exam-series/"+seriesID+"/status", map[string]string{"status": "in_review"})
	status, body = validator.do("POST", "/api/v1/admin/exam-series/"+seriesID+"/status", map[string]string{"status": "published"})
	mustStatus(t, status, body, http.StatusOK, "publication série")
	return seriesID, optA, optB, order
}

func rawJSON(t *testing.T, c *client, method, path string) string {
	t.Helper()
	_, body := c.do(method, path, nil)
	b, _ := json.Marshal(body)
	return string(b)
}

func TestExamFlow(t *testing.T) {
	e := newAdminEnv(t)
	seriesID, optA, optB, order := publishedSeries(t, e, 1)
	candidate := e.user("candidate")
	other := e.user("candidate")
	licenseC, licenseB := e.licenseID("C"), e.licenseID("B")

	// Le candidat voit la série publiée de son permis.
	status, body := candidate.do("GET", "/api/v1/license-categories/"+licenseC+"/series", nil)
	mustStatus(t, status, body, http.StatusOK, "séries")
	if items := body["items"].([]any); len(items) != 1 || items[0].(map[string]any)["question_count"] != 10.0 {
		t.Fatalf("séries = %v", body)
	}

	// CDC : choix d'une série d'une autre catégorie → requête rejetée.
	status, body = candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": licenseB, "series_id": seriesID})
	mustStatus(t, status, body, http.StatusBadRequest, "série d'un autre permis")

	status, body = candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": licenseC, "series_id": seriesID})
	mustStatus(t, status, body, http.StatusCreated, "démarrage")
	sid := body["id"].(string)
	if body["duration_minutes"] != 20.0 || len(body["questions"].([]any)) != 10 || body["status"] != "in_progress" {
		t.Fatalf("session = %v", body)
	}
	// Ordre fixe de la série.
	for i, q := range body["questions"].([]any) {
		if q.(map[string]any)["question_id"] != order[i] {
			t.Fatalf("ordre des questions modifié à la position %d", i+1)
		}
	}

	// CDC : correction avant soumission → bonnes réponses non exposées.
	sessionJSON := rawJSON(t, candidate, "GET", "/api/v1/exam-sessions/"+sid)
	for _, leak := range []string{"is_correct", "correct_option_ids", "explanation", "Parce que"} {
		if strings.Contains(sessionJSON, leak) {
			t.Fatalf("la session expose %q avant soumission", leak)
		}
	}
	status, body = candidate.do("GET", "/api/v1/exam-sessions/"+sid+"/review", nil)
	mustStatus(t, status, body, http.StatusConflict, "correction avant soumission")

	// Un seul examen en cours à la fois.
	status, body = candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": licenseC, "series_id": seriesID})
	mustStatus(t, status, body, http.StatusConflict, "second examen")

	// CDC : accès à la session d'un autre candidat → refusé (introuvable).
	status, body = other.do("GET", "/api/v1/exam-sessions/"+sid, nil)
	mustStatus(t, status, body, http.StatusNotFound, "session d'un autre")
	status, body = other.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+order[0], map[string]any{"option_ids": []string{optA[order[0]]}})
	mustStatus(t, status, body, http.StatusNotFound, "réponse dans la session d'un autre")

	// Réponses : 5 justes, 3 fausses, 2 sans réponse → 5 fautes → ajourné.
	answer := func(q, opt string) {
		t.Helper()
		status, body := candidate.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+q, map[string]any{"option_ids": []string{opt}})
		mustStatus(t, status, body, http.StatusNoContent, "réponse")
	}
	for i, q := range order[:8] {
		if i < 5 {
			answer(q, optA[q])
		} else {
			answer(q, optB[q])
		}
	}
	// Option d'une autre question, deux réponses à une question unique : refusés.
	status, body = candidate.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+order[8], map[string]any{"option_ids": []string{optA[order[0]]}})
	mustStatus(t, status, body, http.StatusBadRequest, "proposition d'une autre question")
	status, body = candidate.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+order[8], map[string]any{"option_ids": []string{optA[order[8]], optB[order[8]]}})
	mustStatus(t, status, body, http.StatusBadRequest, "deux réponses à une question unique")

	// CDC : fermeture pendant la session → reprise sans changer série ni ordre ni réponses.
	status, body = candidate.do("GET", "/api/v1/exam-sessions/current", nil)
	mustStatus(t, status, body, http.StatusOK, "reprise")
	if body["id"] != sid || len(body["answers"].(map[string]any)) != 8 {
		t.Fatalf("reprise = %v réponses", body["answers"])
	}

	status, body = candidate.do("POST", "/api/v1/exam-sessions/"+sid+"/submit", nil)
	mustStatus(t, status, body, http.StatusOK, "soumission")
	result := body["result"].(map[string]any)
	if result["fault_count"] != 5.0 || result["unanswered_count"] != 2.0 || result["correct_count"] != 5.0 || result["passed"] != false {
		t.Fatalf("résultat = %v", result)
	}

	// CDC : soumission répétée → pas de double évaluation.
	status, body = candidate.do("POST", "/api/v1/exam-sessions/"+sid+"/submit", nil)
	mustStatus(t, status, body, http.StatusOK, "seconde soumission")
	var results int
	e.db.QueryRow(context.Background(), `SELECT count(*) FROM exam_results WHERE session_id = $1`, sid).Scan(&results)
	if results != 1 || body["result"].(map[string]any)["fault_count"] != 5.0 {
		t.Fatalf("résultats en base = %d", results)
	}

	// Session soumise : plus de modification.
	status, body = candidate.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+order[9], map[string]any{"option_ids": []string{optA[order[9]]}})
	mustStatus(t, status, body, http.StatusConflict, "réponse après soumission")

	// Correction disponible après soumission.
	status, body = candidate.do("GET", "/api/v1/exam-sessions/"+sid+"/review", nil)
	mustStatus(t, status, body, http.StatusOK, "correction")
	qs := body["questions"].([]any)
	first, last := qs[0].(map[string]any), qs[9].(map[string]any)
	if first["outcome"] != "correct" || last["outcome"] != "unanswered" ||
		first["correct_option_ids"].([]any)[0] != optA[order[0]] || first["explanations"].(map[string]any)["fr"] == nil {
		t.Fatalf("correction = %v / %v", first, last)
	}

	// Tentatives illimitées : nouvelle session, historique conservé.
	status, body = candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": licenseC, "series_id": seriesID})
	mustStatus(t, status, body, http.StatusCreated, "nouvelle tentative")
	status, body = candidate.do("GET", "/api/v1/exam-sessions", nil)
	mustStatus(t, status, body, http.StatusOK, "historique")
	if len(body["items"].([]any)) != 2 {
		t.Fatalf("historique = %v", body)
	}
}

func TestExamTimeoutAndVersioning(t *testing.T) {
	e := newAdminEnv(t)
	seriesID, optA, _, order := publishedSeries(t, e, 2)
	candidate := e.user("candidate")
	licenseC := e.licenseID("C")

	status, body := candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": licenseC, "series_id": seriesID})
	mustStatus(t, status, body, http.StatusCreated, "démarrage")
	sid := body["id"].(string)
	for _, q := range order {
		candidate.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+q, map[string]any{"option_ids": []string{optA[q]}})
	}

	// CDC : nouvelle version d'une série → les sessions historiques restent inchangées.
	editor, validator := e.user("editor"), e.user("validator")
	reversed := make([]string, len(order))
	for i, q := range order {
		reversed[len(order)-1-i] = q
	}
	editor.do("PUT", "/api/v1/admin/exam-series/"+seriesID+"/questions", map[string]any{"question_ids": reversed})
	editor.do("POST", "/api/v1/admin/exam-series/"+seriesID+"/status", map[string]string{"status": "in_review"})
	validator.do("POST", "/api/v1/admin/exam-series/"+seriesID+"/status", map[string]string{"status": "published"})
	_, body = candidate.do("GET", "/api/v1/exam-sessions/"+sid, nil)
	if body["series_version"] != 1.0 || body["questions"].([]any)[0].(map[string]any)["question_id"] != order[0] {
		t.Fatalf("la session en cours a changé de version : v%v", body["series_version"])
	}

	// Temps écoulé : soumission automatique, réponses tardives refusées.
	if _, err := e.db.Exec(context.Background(),
		`UPDATE exam_sessions SET expires_at = now() - interval '1 minute' WHERE id = $1`, sid); err != nil {
		t.Fatal(err)
	}
	status, body = candidate.do("PUT", "/api/v1/exam-sessions/"+sid+"/answers/"+order[0], map[string]any{"option_ids": []string{optA[order[0]]}})
	mustStatus(t, status, body, http.StatusConflict, "réponse après expiration")
	_, body = candidate.do("GET", "/api/v1/exam-sessions/"+sid, nil)
	if body["status"] != "submitted" || body["submit_reason"] != "timeout" || body["result"].(map[string]any)["passed"] != true {
		t.Fatalf("session expirée = %v %v %v", body["status"], body["submit_reason"], body["result"])
	}

	// Abandon explicite pour démarrer un nouvel examen.
	status, body = candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": licenseC, "series_id": seriesID})
	mustStatus(t, status, body, http.StatusCreated, "nouvelle session (v2)")
	if body["series_version"] != 2.0 {
		t.Errorf("nouvelle session en v%v ; attendu v2", body["series_version"])
	}
	status, body = candidate.do("POST", "/api/v1/exam-sessions", map[string]any{"license_category_id": licenseC, "series_id": seriesID, "abandon_current": true})
	mustStatus(t, status, body, http.StatusCreated, "abandon puis nouvelle session")
}
