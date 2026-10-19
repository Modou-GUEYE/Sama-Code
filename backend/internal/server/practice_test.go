package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestPractice(t *testing.T) {
	e := newAdminEnv(t)
	seriesID, optA, optB, order := publishedSeries(t, e, 5)
	candidate := e.user("candidate")
	licenseC := e.licenseID("C")

	// Questions de la série, dans l'ordre, sans bonnes réponses ni explications.
	status, body := candidate.do("GET", "/api/v1/practice/series/"+seriesID+"/questions", nil)
	mustStatus(t, status, body, http.StatusOK, "questions d'entraînement")
	items := body["items"].([]any)
	if len(items) != 10 || items[0].(map[string]any)["question"].(map[string]any)["question_id"] != order[0] {
		t.Fatalf("questions = %d", len(items))
	}
	if raw := rawJSON(t, candidate, "GET", "/api/v1/practice/series/"+seriesID+"/questions"); strings.Contains(raw, "is_correct") ||
		strings.Contains(raw, "Parce que") {
		t.Fatal("les bonnes réponses sont exposées avant la correction")
	}

	check := func(q, opt string) map[string]any {
		t.Helper()
		status, body := candidate.do("POST", "/api/v1/practice/series/"+seriesID+"/questions/"+q+"/check",
			map[string]any{"option_ids": []string{opt}})
		mustStatus(t, status, body, http.StatusOK, "correction")
		return body
	}

	// Mauvaise réponse : correction, bonne réponse et explication renvoyées.
	res := check(order[0], optB[order[0]])
	if res["outcome"] != "incorrect" || res["correct_option_ids"].([]any)[0] != optA[order[0]] ||
		res["explanations"].(map[string]any)["fr"] == nil {
		t.Fatalf("correction = %v", res)
	}
	if check(order[1], optA[order[1]])["outcome"] != "correct" {
		t.Fatal("bonne réponse non reconnue")
	}

	// Reprise : les questions déjà faites portent leur dernier résultat.
	_, body = candidate.do("GET", "/api/v1/practice/series/"+seriesID+"/questions", nil)
	got := body["items"].([]any)
	if got[0].(map[string]any)["last_outcome"] != "incorrect" || got[1].(map[string]any)["last_outcome"] != "correct" ||
		got[2].(map[string]any)["last_outcome"] != nil {
		t.Fatalf("derniers résultats = %v / %v / %v", got[0].(map[string]any)["last_outcome"],
			got[1].(map[string]any)["last_outcome"], got[2].(map[string]any)["last_outcome"])
	}

	// Réponse vide refusée.
	status, body = candidate.do("POST", "/api/v1/practice/series/"+seriesID+"/questions/"+order[2]+"/check", map[string]any{"option_ids": []string{}})
	mustStatus(t, status, body, http.StatusBadRequest, "réponse vide")

	// Bilan : 2 questions travaillées, 1 maîtrisée, 1 erreur à revoir.
	status, body = candidate.do("GET", "/api/v1/practice", nil)
	mustStatus(t, status, body, http.StatusOK, "bilan")
	var c map[string]any
	for _, l := range body["items"].([]any) {
		if l.(map[string]any)["code"] == "C" {
			c = l.(map[string]any)
		}
	}
	s := c["series"].([]any)[0].(map[string]any)
	if c["mistakes"] != 1.0 || s["practiced"] != 2.0 || s["mastered"] != 1.0 {
		t.Fatalf("bilan C = %v", c)
	}

	// Revoir mes erreurs : la question ratée, puis plus rien une fois corrigée.
	status, body = candidate.do("GET", "/api/v1/practice/license-categories/"+licenseC+"/mistakes", nil)
	mustStatus(t, status, body, http.StatusOK, "erreurs")
	if m := body["items"].([]any); len(m) != 1 || m[0].(map[string]any)["question"].(map[string]any)["question_id"] != order[0] {
		t.Fatalf("erreurs = %v", m)
	}
	check(order[0], optA[order[0]])
	_, body = candidate.do("GET", "/api/v1/practice/license-categories/"+licenseC+"/mistakes", nil)
	if len(body["items"].([]any)) != 0 {
		t.Fatalf("erreurs après correction = %v", body["items"])
	}

	// Progression par série : une série qui partage les mêmes questions repart de zéro.
	editor, validator := e.user("editor"), e.user("validator")
	status, body = editor.do("POST", "/api/v1/admin/exam-series", map[string]any{"license_category_id": licenseC, "series_number": 4})
	mustStatus(t, status, body, http.StatusCreated, "seconde série")
	shared := body["id"].(string)
	editor.do("PUT", "/api/v1/admin/exam-series/"+shared+"/questions", map[string]any{"question_ids": order})
	editor.do("POST", "/api/v1/admin/exam-series/"+shared+"/status", map[string]string{"status": "in_review"})
	status, body = validator.do("POST", "/api/v1/admin/exam-series/"+shared+"/status", map[string]string{"status": "published"})
	mustStatus(t, status, body, http.StatusOK, "publication seconde série")

	_, body = candidate.do("GET", "/api/v1/practice", nil)
	for _, l := range body["items"].([]any) {
		if l.(map[string]any)["code"] != "C" {
			continue
		}
		for _, ser := range l.(map[string]any)["series"].([]any) {
			m := ser.(map[string]any)
			switch m["id"] {
			case seriesID:
				if m["practiced"] != 2.0 || m["mastered"] != 2.0 {
					t.Errorf("série d'origine = %v", m)
				}
			case shared:
				if m["practiced"] != 0.0 || m["mastered"] != 0.0 {
					t.Errorf("la série qui partage les questions ne doit pas hériter de la progression : %v", m)
				}
			}
		}
	}
}
