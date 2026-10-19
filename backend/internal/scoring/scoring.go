// Package scoring calcule le résultat d'un examen (CDC §5.3). Fonction pure : aucune
// dépendance à la base, pour être testée exhaustivement.
package scoring

type Outcome string

const (
	Correct    Outcome = "correct"
	Incorrect  Outcome = "incorrect"
	Unanswered Outcome = "unanswered"
)

// Question : identifiant et bonnes propositions (une seule pour « single »).
type Question struct {
	ID               string
	CorrectOptionIDs []string
}

type Result struct {
	CorrectCount    int                `json:"correct_count"`
	FaultCount      int                `json:"fault_count"`
	UnansweredCount int                `json:"unanswered_count"`
	MaxFaults       int                `json:"max_faults"`
	Passed          bool               `json:"passed"`
	Outcomes        map[string]Outcome `json:"-"`
}

// Evaluate applique les règles du CDC :
//   - une question est juste si l'ensemble coché est exactement l'ensemble des bonnes réponses ;
//   - une question sans réponse est une faute ;
//   - les fautes se comptent par question, pas par proposition ;
//   - réussi si fautes ≤ maxFaults (4 pour les permis B et C).
func Evaluate(questions []Question, answers map[string][]string, maxFaults int) Result {
	r := Result{MaxFaults: maxFaults, Outcomes: make(map[string]Outcome, len(questions))}
	for _, q := range questions {
		selected := answers[q.ID]
		switch {
		case len(selected) == 0:
			r.Outcomes[q.ID] = Unanswered
			r.UnansweredCount++
			r.FaultCount++
		case sameSet(selected, q.CorrectOptionIDs):
			r.Outcomes[q.ID] = Correct
			r.CorrectCount++
		default:
			r.Outcomes[q.ID] = Incorrect
			r.FaultCount++
		}
	}
	r.Passed = r.FaultCount <= maxFaults
	return r
}

func sameSet(a, b []string) bool {
	set := make(map[string]bool, len(b))
	for _, x := range b {
		set[x] = true
	}
	seen := make(map[string]bool, len(a))
	for _, x := range a {
		if !set[x] {
			return false
		}
		seen[x] = true
	}
	return len(seen) == len(set)
}
