package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/auth"
	"samacode/backend/internal/media"
	"samacode/backend/internal/testutil"
)

type adminEnv struct {
	t      *testing.T
	db     *pgxpool.Pool
	h      http.Handler
	tokens *auth.Tokens
	seq    int
}

func newAdminEnv(t *testing.T) *adminEnv {
	db := testutil.DB(t)
	secret := []byte("test-secret-test-secret-test-secret!")
	tokens := auth.NewTokens(secret, 15*time.Minute)
	storage, err := media.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := NewRouter(Deps{
		DB: db, Tokens: tokens, Media: storage,
		Auth: auth.NewService(db, tokens, &testutil.SMS{}, secret, auth.DefaultConfig()),
	})
	return &adminEnv{t: t, db: db, h: h, tokens: tokens}
}

// user crée un compte avec les rôles donnés et renvoie un client authentifié.
func (e *adminEnv) user(roles ...string) *client {
	e.t.Helper()
	e.seq++
	var id string
	phone := fmt.Sprintf("+22177000%04d", e.seq)
	if err := e.db.QueryRow(context.Background(),
		`INSERT INTO users (display_name, phone_number) VALUES ($1, $2) RETURNING id`, "User "+phone, phone).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.db.Exec(context.Background(),
		`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE code = ANY($2)`, id, roles); err != nil {
		e.t.Fatal(err)
	}
	token, _, err := e.tokens.IssueAccess(id, roles, time.Now())
	if err != nil {
		e.t.Fatal(err)
	}
	return &client{t: e.t, h: e.h, token: token}
}

func (e *adminEnv) licenseID(code string) string {
	var id string
	if err := e.db.QueryRow(context.Background(), `SELECT id FROM license_categories WHERE code = $1`, code).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

// question : corps complet (fr + wo), réponse unique, bonne réponse A.
func question(label string) map[string]any {
	return map[string]any{
		"question_type": "single",
		"contents": map[string]any{
			"fr": map[string]any{"text": "Question " + label, "explanation": "Parce que " + label},
			"wo": map[string]any{"text": "Laaj " + label},
		},
		"options": []map[string]any{
			{"code": "A", "is_correct": true, "contents": map[string]any{"fr": map[string]any{"text": "Oui"}, "wo": map[string]any{"text": "Waaw"}}},
			{"code": "B", "is_correct": false, "contents": map[string]any{"fr": map[string]any{"text": "Non"}, "wo": map[string]any{"text": "Déedéet"}}},
		},
	}
}

func mustStatus(t *testing.T, status int, body map[string]any, want int, what string) {
	t.Helper()
	if status != want {
		t.Fatalf("%s = %d %v ; attendu %d", what, status, body, want)
	}
}

func TestAdminContentWorkflow(t *testing.T) {
	e := newAdminEnv(t)
	candidate := e.user("candidate")
	editor := e.user("editor")
	validator := e.user("validator")

	// Un candidat n'accède pas à l'administration.
	status, body := candidate.do("GET", "/api/v1/admin/questions", nil)
	mustStatus(t, status, body, http.StatusForbidden, "candidat → admin")

	// Brouillon en français seul : accepté.
	draft := question("1")
	delete(draft["contents"].(map[string]any), "wo")
	status, body = editor.do("POST", "/api/v1/admin/questions", draft)
	mustStatus(t, status, body, http.StatusCreated, "création brouillon")
	qid := body["id"].(string)

	// Soumettre sans le wolof : refusé (D10).
	status, body = editor.do("POST", "/api/v1/admin/questions/"+qid+"/status", map[string]string{"status": "in_review"})
	mustStatus(t, status, body, http.StatusUnprocessableEntity, "soumission sans wolof")

	// Réponse unique avec deux bonnes réponses : refusée.
	bad := question("x")
	bad["options"].([]map[string]any)[1]["is_correct"] = true
	status, body = editor.do("PUT", "/api/v1/admin/questions/"+qid, bad)
	mustStatus(t, status, body, http.StatusBadRequest, "deux bonnes réponses")

	status, body = editor.do("PUT", "/api/v1/admin/questions/"+qid, question("1"))
	mustStatus(t, status, body, http.StatusOK, "ajout du wolof")
	status, body = editor.do("POST", "/api/v1/admin/questions/"+qid+"/status", map[string]string{"status": "in_review"})
	mustStatus(t, status, body, http.StatusOK, "soumission")

	// Un éditeur ne publie pas ; un validateur si.
	status, body = editor.do("POST", "/api/v1/admin/questions/"+qid+"/status", map[string]string{"status": "published"})
	mustStatus(t, status, body, http.StatusForbidden, "publication par éditeur")
	status, body = validator.do("POST", "/api/v1/admin/questions/"+qid+"/status", map[string]string{"status": "published"})
	mustStatus(t, status, body, http.StatusOK, "publication par validateur")

	// La liste filtre par statut.
	status, body = editor.do("GET", "/api/v1/admin/questions?status=published", nil)
	mustStatus(t, status, body, http.StatusOK, "liste")
	if body["total"] != 1.0 {
		t.Errorf("questions publiées = %v", body["total"])
	}
}

func TestAdminSeriesPublication(t *testing.T) {
	e := newAdminEnv(t)
	editor := e.user("editor")
	validator := e.user("validator")
	licenseC := e.licenseID("C") // 10 questions par série

	publish := func(c *client, path string) {
		t.Helper()
		for _, s := range []string{"in_review", "published"} {
			who := editor
			if s == "published" {
				who = validator
			}
			status, body := who.do("POST", path+"/status", map[string]string{"status": s})
			mustStatus(t, status, body, http.StatusOK, path+" → "+s)
		}
	}

	var ids []string
	for i := range 10 {
		status, body := editor.do("POST", "/api/v1/admin/questions", question(fmt.Sprint(i)))
		mustStatus(t, status, body, http.StatusCreated, "création question")
		id := body["id"].(string)
		ids = append(ids, id)
		publish(editor, "/api/v1/admin/questions/"+id)
	}

	status, body := editor.do("POST", "/api/v1/admin/exam-series", map[string]any{"license_category_id": licenseC, "series_number": 6})
	mustStatus(t, status, body, http.StatusBadRequest, "série 6 du permis C (5 max)")
	status, body = editor.do("POST", "/api/v1/admin/exam-series", map[string]any{"license_category_id": licenseC, "series_number": 1})
	mustStatus(t, status, body, http.StatusCreated, "création série")
	sid := body["id"].(string)
	status, body = editor.do("POST", "/api/v1/admin/exam-series", map[string]any{"license_category_id": licenseC, "series_number": 1})
	mustStatus(t, status, body, http.StatusConflict, "série en double")

	// Série incomplète : soumission refusée.
	status, body = editor.do("PUT", "/api/v1/admin/exam-series/"+sid+"/questions", map[string]any{"question_ids": ids[:9]})
	mustStatus(t, status, body, http.StatusOK, "9 questions")
	status, body = editor.do("POST", "/api/v1/admin/exam-series/"+sid+"/status", map[string]string{"status": "in_review"})
	mustStatus(t, status, body, http.StatusUnprocessableEntity, "série incomplète")

	status, body = editor.do("PUT", "/api/v1/admin/exam-series/"+sid+"/questions", map[string]any{"question_ids": ids})
	mustStatus(t, status, body, http.StatusOK, "10 questions")
	if qs := body["questions"].([]any); len(qs) != 10 || qs[0].(map[string]any)["question_id"] != ids[0] {
		t.Fatalf("ordre des questions = %v", qs)
	}
	publish(editor, "/api/v1/admin/exam-series/"+sid)

	// Instantané v1 : 10 questions, ordre et bonnes réponses figés.
	var payload []byte
	if err := e.db.QueryRow(context.Background(),
		`SELECT payload FROM exam_series_snapshots WHERE series_id = $1 AND version = 1`, sid).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var snap struct {
		Version   int `json:"version"`
		MaxFaults int `json:"max_faults"`
		Questions []struct {
			QuestionID string `json:"question_id"`
			Options    []struct {
				Code      string `json:"code"`
				IsCorrect bool   `json:"is_correct"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(payload, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Version != 1 || snap.MaxFaults != 4 || len(snap.Questions) != 10 || snap.Questions[0].QuestionID != ids[0] ||
		!snap.Questions[0].Options[0].IsCorrect {
		t.Fatalf("instantané = %+v", snap)
	}

	// Une question d'une série publiée ne se modifie pas.
	status, body = editor.do("PUT", "/api/v1/admin/questions/"+ids[0], question("modifiée"))
	mustStatus(t, status, body, http.StatusConflict, "modification d'une question en série publiée")

	// Réordonner repasse la série en brouillon ; republier crée la v2 sans toucher la v1.
	reversed := make([]string, len(ids))
	for i, id := range ids {
		reversed[len(ids)-1-i] = id
	}
	status, body = editor.do("PUT", "/api/v1/admin/exam-series/"+sid+"/questions", map[string]any{"question_ids": reversed})
	mustStatus(t, status, body, http.StatusOK, "réordonner")
	if body["status"] != "draft" {
		t.Errorf("statut après modification = %v", body["status"])
	}
	publish(editor, "/api/v1/admin/exam-series/"+sid)

	var versions int
	var firstOfV1 string
	if err := e.db.QueryRow(context.Background(),
		`SELECT (SELECT count(*) FROM exam_series_snapshots WHERE series_id = $1),
		        (SELECT payload->'questions'->0->>'question_id' FROM exam_series_snapshots WHERE series_id = $1 AND version = 1)`,
		sid).Scan(&versions, &firstOfV1); err != nil {
		t.Fatal(err)
	}
	if versions != 2 || firstOfV1 != ids[0] {
		t.Errorf("versions = %d, 1re question de v1 = %s (attendu %s)", versions, firstOfV1, ids[0])
	}
}

func TestAdminMediaAndUsers(t *testing.T) {
	e := newAdminEnv(t)
	editor := e.user("editor")
	admin := e.user("admin")

	upload := func(c *client, name string, content []byte) (int, map[string]any) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		fw.Write(content)
		mw.Close()
		req := httptest.NewRequest("POST", "/api/v1/admin/media", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+c.token)
		rec := httptest.NewRecorder()
		e.h.ServeHTTP(rec, req)
		var out map[string]any
		json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00")
	status, body := upload(editor, "panneau.png", png)
	mustStatus(t, status, body, http.StatusCreated, "envoi PNG")
	if body["media_type"] != "image" || body["mime_type"] != "image/png" {
		t.Errorf("média = %v", body)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, httptest.NewRequest("GET", body["url"].(string), nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(rec.Body.Bytes(), png) {
		t.Errorf("lecture du fichier = %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}

	status, body = upload(editor, "script.png", []byte("<script>alert(1)</script>"))
	mustStatus(t, status, body, http.StatusUnsupportedMediaType, "faux PNG")

	// Gestion des comptes : admin seulement, pas d'auto-rétrogradation.
	status, body = editor.do("GET", "/api/v1/admin/users", nil)
	mustStatus(t, status, body, http.StatusForbidden, "éditeur → utilisateurs")
	status, body = admin.do("GET", "/api/v1/admin/users", nil)
	mustStatus(t, status, body, http.StatusOK, "admin → utilisateurs")
	users := body["items"].([]any)
	var adminID, editorID string
	for _, u := range users {
		m := u.(map[string]any)
		roles := fmt.Sprint(m["roles"])
		switch roles {
		case "[admin]":
			adminID = m["id"].(string)
		case "[editor]":
			editorID = m["id"].(string)
		}
	}
	status, body = admin.do("PUT", "/api/v1/admin/users/"+adminID+"/roles", map[string]any{"roles": []string{"editor"}})
	mustStatus(t, status, body, http.StatusConflict, "auto-rétrogradation")
	status, body = admin.do("PUT", "/api/v1/admin/users/"+editorID+"/roles", map[string]any{"roles": []string{"editor", "validator"}})
	mustStatus(t, status, body, http.StatusOK, "promotion en validateur")

	status, body = admin.do("PATCH", "/api/v1/admin/license-categories/"+e.licenseID("B"), map[string]any{"exam_duration_minutes": 25})
	mustStatus(t, status, body, http.StatusOK, "durée d'examen")
	if body["exam_duration_minutes"] != 25.0 {
		t.Errorf("durée = %v", body["exam_duration_minutes"])
	}
	// Remise à la valeur par défaut pour les autres tests.
	admin.do("PATCH", "/api/v1/admin/license-categories/"+e.licenseID("B"), map[string]any{"exam_duration_minutes": 20})

	status, body = editor.do("GET", "/api/v1/admin/statistics", nil)
	mustStatus(t, status, body, http.StatusOK, "statistiques")
}
