// Commande demo-illustrations : attache des images aux questions de DÉMONSTRATION
// et republie les séries qui les contiennent (nouvelle version), en suivant le
// circuit normal : dépublier la série → modifier → valider → republier.
//
// Les fichiers du dossier sont nommés « NN-description.png », NN étant le rang de
// la question de démonstration (01 = première question créée par seed-demo).
//
//	go run ./cmd/demo-illustrations -phone "77 123 45 67" -dir ./illustrations
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/auth"
	"samacode/backend/internal/database"
	"samacode/backend/internal/examseries"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/media"
	"samacode/backend/internal/questions"
)

func main() {
	phone := flag.String("phone", "", "numéro d'un compte administrateur existant")
	dir := flag.String("dir", "", "dossier des images PNG (NN-description.png)")
	flag.Parse()
	if err := run(*phone, *dir); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(rawPhone, dir string) error {
	phone, err := auth.NormalizePhone(rawPhone)
	if err != nil {
		return fmt.Errorf("numéro invalide %q", rawPhone)
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL est obligatoire")
	}
	mediaDir := os.Getenv("MEDIA_DIR")
	if mediaDir == "" {
		mediaDir = "var/media"
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	storage, err := media.NewLocalStorage(mediaDir)
	if err != nil {
		return err
	}

	var actor string
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE phone_number = $1`, phone).Scan(&actor); err != nil {
		return fmt.Errorf("aucun compte avec le numéro %s", phone)
	}
	p := httpx.Principal{UserID: actor, Roles: []string{"admin"}}

	demoIDs, err := demoQuestions(ctx, pool)
	if err != nil {
		return err
	}
	images, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil || len(images) == 0 {
		return fmt.Errorf("aucune image PNG dans %q", dir)
	}
	sort.Strings(images)

	// 1. Dépublier les séries publiées qui contiennent des questions de démonstration.
	series := examseries.NewService(pool)
	seriesIDs, err := publishedSeriesWith(ctx, pool, demoIDs)
	if err != nil {
		return err
	}
	for _, id := range seriesIDs {
		if _, err := series.SetStatus(ctx, p, id, "draft"); err != nil {
			return fmt.Errorf("dépublier la série : %w", err)
		}
	}

	// 2. Enregistrer chaque image et l'attacher (en français) à la question correspondante.
	qs := questions.NewService(pool)
	for _, path := range images {
		rank, err := strconv.Atoi(strings.SplitN(filepath.Base(path), "-", 2)[0])
		if err != nil || rank < 1 || rank > len(demoIDs) {
			fmt.Printf("ignoré : %s (rang invalide)\n", filepath.Base(path))
			continue
		}
		mediaID, err := storeImage(ctx, pool, storage, actor, path)
		if err != nil {
			return err
		}
		q, err := qs.Get(ctx, demoIDs[rank-1])
		if err != nil {
			return err
		}
		in := questions.Input{QuestionType: q.QuestionType, RelatedLessonID: q.RelatedLessonID, Contents: q.Contents, Options: q.Options}
		fr := in.Contents["fr"]
		fr.ImageMediaID = &mediaID
		in.Contents["fr"] = fr
		// D25 : le wolof ne garde que ses audios.
		if wo, ok := in.Contents["wo"]; ok {
			in.Contents["wo"] = questions.Content{AudioMediaID: wo.AudioMediaID, ExplanationAudioMediaID: wo.ExplanationAudioMediaID}
		}
		for i := range in.Options {
			if wo, ok := in.Options[i].Contents["wo"]; ok {
				in.Options[i].Contents["wo"] = questions.OptionContent{AudioMediaID: wo.AudioMediaID}
			}
		}
		if _, err := qs.Update(ctx, p, q.ID, in); err != nil {
			return fmt.Errorf("question %d : %w", rank, err)
		}
		for _, st := range []string{"in_review", "published"} {
			if _, err := qs.SetStatus(ctx, p, q.ID, st); err != nil {
				return fmt.Errorf("question %d → %s : %w", rank, st, err)
			}
		}
		fmt.Printf("Question %d illustrée (%s).\n", rank, filepath.Base(path))
	}

	// 3. Republier les séries : nouvelle version figée, avec les illustrations.
	for _, id := range seriesIDs {
		for _, st := range []string{"in_review", "published"} {
			d, err := series.SetStatus(ctx, p, id, st)
			if err != nil {
				return fmt.Errorf("republier la série : %w", err)
			}
			if st == "published" {
				fmt.Printf("Permis %s, série %d republiée en v%d.\n", d.LicenseCode, d.SeriesNumber, d.CurrentVersion)
			}
		}
	}
	return nil
}

func demoQuestions(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT q.id FROM questions q JOIN question_contents c ON c.question_id = q.id AND c.language_code = 'fr'
		  WHERE c.text_content LIKE '[DÉMO] %' AND q.status <> 'archived' ORDER BY q.created_at`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func publishedSeriesWith(ctx context.Context, pool *pgxpool.Pool, questionIDs []string) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT s.id FROM exam_series s JOIN series_questions sq ON sq.series_id = s.id
		  WHERE s.status = 'published' AND sq.question_id = ANY($1::uuid[])`, questionIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// storeImage enregistre le fichier comme un envoi depuis l'administration.
func storeImage(ctx context.Context, pool *pgxpool.Pool, storage media.Storage, actor, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	key := "image/" + id + ".png"
	if err := storage.Put(ctx, key, bytes.NewReader(data)); err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	_, err = pool.Exec(ctx,
		`INSERT INTO media_assets (id, media_type, storage_key, mime_type, size_bytes, checksum, original_name, created_by)
		 VALUES ($1, 'image', $2, 'image/png', $3, $4, $5, $6)`,
		id, key, len(data), hex.EncodeToString(sum[:]), filepath.Base(path), actor)
	return id, err
}
