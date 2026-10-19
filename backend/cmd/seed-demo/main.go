// Commande seed-demo : crée et publie des séries de DÉMONSTRATION (permis B série 1,
// permis C série 1) pour tester l'examen en développement. Les questions sont
// marquées [DÉMO] : ce ne sont pas des contenus validés. Pas de texte wolof (D25) :
// le wolof n'a que des audios, à enregistrer dans l'administration.
//
//	go run ./cmd/seed-demo -phone "77 123 45 67"   (compte administrateur existant)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/auth"
	"samacode/backend/internal/database"
	"samacode/backend/internal/examseries"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/questions"
)

type demo struct {
	text, explanation string
	options           []string
	correct           []int // indices des bonnes réponses
}

// Questions génériques de démonstration, à remplacer par les contenus validés.
var demos = []demo{
	{"Un feu rouge fixe signifie :", "Le feu rouge impose l'arrêt.", []string{"Je m'arrête", "Je passe prudemment", "Je klaxonne"}, []int{0}},
	{"Un feu orange fixe signifie :", "L'orange annonce le rouge : on s'arrête sauf si c'est dangereux.", []string{"Je m'arrête si je peux le faire sans danger", "J'accélère", "Je passe toujours"}, []int{0}},
	{"Le port de la ceinture de sécurité est obligatoire :", "Tous les occupants doivent être attachés.", []string{"Pour le conducteur seulement", "Pour tous les occupants", "Seulement sur autoroute"}, []int{1}},
	{"Un panneau triangulaire à bord rouge indique :", "Le triangle à bord rouge annonce un danger.", []string{"Un danger", "Une obligation", "Une interdiction"}, []int{0}},
	{"Un panneau rond à fond bleu indique :", "Le rond bleu impose une obligation.", []string{"Une obligation", "Un danger", "Une indication"}, []int{0}},
	{"Un panneau rond à bord rouge indique :", "Le rond à bord rouge signale une interdiction.", []string{"Une interdiction", "Une obligation", "Un danger"}, []int{0}},
	{"Le panneau STOP impose :", "Au STOP, l'arrêt est obligatoire même si la voie est libre.", []string{"De ralentir", "De marquer un arrêt complet", "De céder le passage sans s'arrêter"}, []int{1}},
	{"Avant de dépasser, je dois :", "Il faut vérifier, signaler, puis dépasser.", []string{"Regarder dans mes rétroviseurs", "Mettre mon clignotant", "Accélérer sans prévenir"}, []int{0, 1}},
	{"Téléphoner en tenant le téléphone en conduisant est :", "Tenir son téléphone en conduisant est interdit et dangereux.", []string{"Autorisé en ville", "Interdit", "Autorisé à l'arrêt au feu"}, []int{1}},
	{"Par temps de pluie, je dois :", "Sur route mouillée, on ralentit et on augmente la distance de sécurité.", []string{"Réduire ma vitesse", "Augmenter la distance de sécurité", "Rouler plus près du véhicule de devant"}, []int{0, 1}},
	{"Un piéton est engagé sur un passage piéton :", "Le piéton engagé est prioritaire.", []string{"Je le laisse traverser", "Je klaxonne pour qu'il se dépêche", "Je passe devant lui"}, []int{0}},
	{"La distance de sécurité sert à :", "Elle laisse le temps de réagir et de freiner.", []string{"Avoir le temps de freiner", "Doubler plus facilement", "Rien de particulier"}, []int{0}},
	{"La fatigue au volant provoque :", "La fatigue allonge le temps de réaction.", []string{"Un temps de réaction plus long", "Une meilleure vigilance", "Aucun effet"}, []int{0}},
	{"L'alcool au volant :", "L'alcool diminue les réflexes et augmente le risque d'accident.", []string{"Diminue les réflexes", "Améliore la conduite", "N'a aucun effet"}, []int{0}},
	{"Un clignotant sert à :", "Le clignotant annonce un changement de direction.", []string{"Annoncer un changement de direction", "Saluer les autres conducteurs", "Signaler une panne"}, []int{0}},
	{"Les feux de détresse s'utilisent :", "Ils signalent un danger ou un véhicule immobilisé.", []string{"En cas de panne ou de danger", "Pour stationner en double file", "Pour rouler plus vite"}, []int{0}},
	{"À l'approche d'une école, je dois :", "Près d'une école, des enfants peuvent traverser à tout moment.", []string{"Ralentir", "Être attentif aux enfants", "Klaxonner pour les prévenir"}, []int{0, 1}},
	{"Une ligne continue au sol :", "On ne franchit pas une ligne continue.", []string{"Ne doit pas être franchie", "Peut être franchie pour dépasser", "N'a pas de signification"}, []int{0}},
	{"Pour tourner à droite, je me place :", "Pour tourner à droite, on se serre à droite.", []string{"Sur la droite de la chaussée", "Au milieu", "Sur la gauche"}, []int{0}},
	{"Un agent de circulation lève le bras :", "Le bras levé signifie l'arrêt pour tous les usagers.", []string{"Je m'arrête", "Je continue", "J'accélère"}, []int{0}},
	{"Avant de démarrer, je vérifie :", "Rétroviseurs et angle mort avant de s'insérer.", []string{"Mes rétroviseurs", "L'angle mort", "La radio"}, []int{0, 1}},
	{"Un enfant en bas âge doit voyager :", "Les jeunes enfants voyagent dans un siège adapté.", []string{"Dans un siège adapté", "Sur les genoux d'un adulte", "Debout entre les sièges"}, []int{0}},
	{"Les pneus usés :", "Des pneus usés adhèrent moins, surtout sous la pluie.", []string{"Allongent la distance de freinage", "Améliorent l'adhérence", "Ne changent rien"}, []int{0}},
	{"La nuit, en croisant un véhicule, j'utilise :", "On passe en feux de croisement pour ne pas éblouir.", []string{"Les feux de croisement", "Les feux de route", "Aucun feu"}, []int{0}},
	{"En cas d'accident, je dois d'abord :", "Protéger, alerter, puis secourir.", []string{"Protéger les lieux", "Quitter les lieux", "Déplacer les blessés immédiatement"}, []int{0}},
}

func main() {
	phone := flag.String("phone", "", "numéro d'un compte administrateur existant")
	flag.Parse()
	if err := run(*phone); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(rawPhone string) error {
	phone, err := auth.NormalizePhone(rawPhone)
	if err != nil {
		return fmt.Errorf("numéro invalide %q", rawPhone)
	}
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

	var actor string
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE phone_number = $1`, phone).Scan(&actor); err != nil {
		return fmt.Errorf("aucun compte avec le numéro %s", phone)
	}
	// Principal technique : les services vérifient les rôles et journalisent l'acteur.
	p := httpx.Principal{UserID: actor, Roles: []string{"admin"}}

	qs := questions.NewService(pool)
	// Relancer la commande réutilise les questions de démonstration déjà publiées.
	ids, err := existingDemoQuestions(ctx, pool)
	if err != nil {
		return err
	}
	if len(ids) >= len(demos) {
		ids = ids[:len(demos)]
		fmt.Printf("%d questions de démonstration déjà publiées : réutilisées.\n", len(ids))
	} else {
		ids = nil
	}
	for i, d := range demos {
		if len(ids) == len(demos) {
			break
		}
		in := questions.Input{
			QuestionType: "single",
			Contents: map[string]questions.Content{
				"fr": {Text: "[DÉMO] " + d.text, Explanation: &demos[i].explanation},
			},
		}
		if len(d.correct) > 1 {
			in.QuestionType = "multiple"
		}
		for j, opt := range d.options {
			o := questions.Option{
				Code: string(rune('A' + j)),
				Contents: map[string]questions.OptionContent{
					"fr": {Text: opt},
				},
			}
			for _, c := range d.correct {
				o.IsCorrect = o.IsCorrect || c == j
			}
			in.Options = append(in.Options, o)
		}
		q, err := qs.Create(ctx, p, in)
		if err != nil {
			return fmt.Errorf("question %d : %w", i+1, err)
		}
		for _, s := range []string{"in_review", "published"} {
			if _, err := qs.SetStatus(ctx, p, q.ID, s); err != nil {
				return fmt.Errorf("question %d → %s : %w", i+1, s, err)
			}
		}
		ids = append(ids, q.ID)
	}
	if len(ids) == len(demos) {
		fmt.Printf("%d questions de démonstration prêtes.\n", len(ids))
	}

	series := examseries.NewService(pool)
	for _, lic := range []struct {
		code  string
		count int
	}{{"B", 25}, {"C", 10}} {
		var licenseID string
		if err := pool.QueryRow(ctx, `SELECT id FROM license_categories WHERE code = $1`, lic.code).Scan(&licenseID); err != nil {
			return err
		}
		seriesID, err := emptySeries(ctx, pool, series, p, licenseID)
		if err != nil {
			return fmt.Errorf("permis %s, série 1 : %w", lic.code, err)
		}
		if _, err := series.SetQuestions(ctx, p, seriesID, ids[:lic.count]); err != nil {
			return err
		}
		for _, st := range []string{"in_review", "published"} {
			if _, err := series.SetStatus(ctx, p, seriesID, st); err != nil {
				return fmt.Errorf("permis %s → %s : %w", lic.code, st, err)
			}
		}
		fmt.Printf("Permis %s, série 1 publiée (%d questions).\n", lic.code, lic.count)
	}
	return nil
}

func existingDemoQuestions(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT q.id FROM questions q JOIN question_contents c ON c.question_id = q.id AND c.language_code = 'fr'
		  WHERE q.status = 'published' AND c.text_content LIKE '[DÉMO] %' ORDER BY q.created_at`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// emptySeries renvoie la série 1 du permis : créée si besoin, réutilisée si elle est vide.
func emptySeries(ctx context.Context, pool *pgxpool.Pool, svc *examseries.Service, p httpx.Principal, licenseID string) (string, error) {
	var id string
	var count int
	err := pool.QueryRow(ctx,
		`SELECT s.id, (SELECT count(*) FROM series_questions sq WHERE sq.series_id = s.id)
		   FROM exam_series s WHERE s.license_category_id = $1 AND s.series_number = 1`, licenseID).Scan(&id, &count)
	if errors.Is(err, pgx.ErrNoRows) {
		d, err := svc.Create(ctx, p, examseries.CreateInput{LicenseCategoryID: licenseID, SeriesNumber: 1})
		return d.ID, err
	}
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", errors.New("la série existe déjà et contient des questions : elle n'est pas modifiée")
	}
	return id, nil
}
