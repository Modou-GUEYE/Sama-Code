// Commande seed-demo-courses : crée et publie des cours de DÉMONSTRATION pour tester
// l'application en développement. Textes marqués [DÉMO], wolof [À traduire] : ce ne
// sont pas des contenus validés. Réutilise les illustrations de demo-illustrations.
//
//	go run ./cmd/seed-demo-courses
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"

	"samacode/backend/internal/database"
)

type lesson struct {
	title string
	text  []string
	image string // nom du fichier d'illustration (media_assets.original_name), facultatif
}

type course struct {
	category, categoryCode, title string
	lessons                       []lesson
}

var demo = []course{
	{
		category: "Panneaux de signalisation", categoryCode: "panneaux", title: "La forme des panneaux",
		lessons: []lesson{
			{"Les panneaux de danger", []string{
				"Un panneau en forme de triangle, bordé de rouge, annonce un danger.",
				"Quand vous le voyez, ralentissez et soyez prêt à vous arrêter.",
			}, "04-danger.png"},
			{"Les panneaux d'obligation", []string{
				"Un panneau rond à fond bleu indique une obligation : vous devez faire ce qu'il montre.",
			}, "05-obligation.png"},
			{"Les panneaux d'interdiction", []string{
				"Un panneau rond bordé de rouge indique une interdiction.",
			}, "06-interdiction.png"},
			{"Le panneau STOP", []string{
				"Au panneau STOP, l'arrêt est obligatoire, même si la route paraît libre.",
				"Arrêtez-vous à la ligne, regardez à gauche et à droite, puis repartez quand c'est sûr.",
			}, "07-stop.png"},
		},
	},
	{
		category: "Les feux", categoryCode: "feux", title: "Les feux tricolores",
		lessons: []lesson{
			{"Le feu rouge", []string{"Le feu rouge impose l'arrêt. Attendez le feu vert pour repartir."}, "01-feu-rouge.png"},
			{"Le feu orange", []string{
				"Le feu orange annonce le feu rouge.",
				"Arrêtez-vous, sauf si vous êtes trop près pour le faire sans danger.",
			}, "02-feu-orange.png"},
		},
	},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL est obligatoire")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM course_translations WHERE title LIKE '[DÉMO] %')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		fmt.Println("Les cours de démonstration existent déjà : rien à faire.")
		return nil
	}

	return database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		for ci, c := range demo {
			var catID string
			err := tx.QueryRow(ctx, `SELECT id FROM course_categories WHERE code = $1`, c.categoryCode).Scan(&catID)
			if errors.Is(err, pgx.ErrNoRows) {
				if err := tx.QueryRow(ctx,
					`INSERT INTO course_categories (code, sort_order) VALUES ($1, $2) RETURNING id`, c.categoryCode, ci+1).Scan(&catID); err != nil {
					return err
				}
				if err := translate(ctx, tx, "course_category_translations", "category_id", "name", catID, c.category); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			var courseID string
			if err := tx.QueryRow(ctx,
				`INSERT INTO courses (category_id, status, sort_order) VALUES ($1, 'published', 1) RETURNING id`, catID).Scan(&courseID); err != nil {
				return err
			}
			if err := translate(ctx, tx, "course_translations", "course_id", "title", courseID, "[DÉMO] "+c.title); err != nil {
				return err
			}
			for li, l := range c.lessons {
				var lessonID string
				if err := tx.QueryRow(ctx,
					`INSERT INTO lessons (course_id, status, sort_order) VALUES ($1, 'published', $2) RETURNING id`,
					courseID, li+1).Scan(&lessonID); err != nil {
					return err
				}
				if err := translate(ctx, tx, "lesson_translations", "lesson_id", "title", lessonID, l.title); err != nil {
					return err
				}
				var imageID *string
				if l.image != "" {
					var id string
					if err := tx.QueryRow(ctx,
						`SELECT id FROM media_assets WHERE original_name = $1 ORDER BY created_at DESC LIMIT 1`, l.image).Scan(&id); err == nil {
						imageID = &id
					}
				}
				for _, lang := range []string{"fr", "wo"} {
					prefix := ""
					if lang == "wo" {
						prefix = "[À traduire] "
					}
					order := 0
					if imageID != nil {
						if _, err := tx.Exec(ctx,
							`INSERT INTO lesson_contents (lesson_id, language_code, content_type, sort_order, media_id)
							 VALUES ($1, $2, 'image', 0, $3)`, lessonID, lang, *imageID); err != nil {
							return err
						}
						order++
					}
					for _, p := range l.text {
						if _, err := tx.Exec(ctx,
							`INSERT INTO lesson_contents (lesson_id, language_code, content_type, sort_order, text_content)
							 VALUES ($1, $2, 'text', $3, $4)`, lessonID, lang, order, prefix+p); err != nil {
							return err
						}
						order++
					}
				}
			}
			fmt.Printf("Cours « %s » publié (%d leçons).\n", c.title, len(c.lessons))
		}
		return nil
	})
}

// translate écrit le libellé fr et sa version wo à traduire.
func translate(ctx context.Context, tx pgx.Tx, table, fk, col, id, title string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO `+table+` (`+fk+`, language_code, `+col+`) VALUES ($1, 'fr', $2), ($1, 'wo', $3)`,
		id, title, "[À traduire] "+title)
	return err
}
