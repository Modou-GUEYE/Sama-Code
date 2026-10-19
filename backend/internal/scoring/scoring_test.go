package scoring

import (
	"fmt"
	"testing"
)

// series construit n questions à réponse unique dont la bonne proposition est « <id>-A ».
func series(n int) []Question {
	qs := make([]Question, n)
	for i := range qs {
		id := fmt.Sprintf("q%d", i)
		qs[i] = Question{ID: id, CorrectOptionIDs: []string{id + "-A"}}
	}
	return qs
}

// answersWithFaults répond juste partout sauf aux `faults` premières questions (mauvaise réponse).
func answersWithFaults(qs []Question, faults int) map[string][]string {
	a := map[string][]string{}
	for i, q := range qs {
		if i < faults {
			a[q.ID] = []string{q.ID + "-B"}
		} else {
			a[q.ID] = []string{q.ID + "-A"}
		}
	}
	return a
}

// Critères d'acceptation du CDC §12 (barème).
func TestThresholds(t *testing.T) {
	cases := []struct {
		name      string
		questions int
		faults    int
		passed    bool
	}{
		{"Permis B avec 4 fautes", 25, 4, true},
		{"Permis B avec 5 fautes", 25, 5, false},
		{"Permis C avec 4 fautes", 10, 4, true},
		{"Permis C avec 5 fautes", 10, 5, false},
		{"Sans faute", 25, 0, true},
	}
	for _, c := range cases {
		qs := series(c.questions)
		r := Evaluate(qs, answersWithFaults(qs, c.faults), 4)
		if r.Passed != c.passed || r.FaultCount != c.faults || r.CorrectCount != c.questions-c.faults {
			t.Errorf("%s : %+v", c.name, r)
		}
	}
}

func TestMultipleAnswers(t *testing.T) {
	q := []Question{{ID: "m", CorrectOptionIDs: []string{"A", "C"}}}
	cases := []struct {
		name     string
		selected []string
		want     Outcome
	}{
		{"réponse multiple exacte", []string{"A", "C"}, Correct},
		{"ordre indifférent", []string{"C", "A"}, Correct},
		{"incomplète", []string{"A"}, Incorrect},
		{"avec un choix erroné", []string{"A", "B", "C"}, Incorrect},
		{"uniquement erronée", []string{"B"}, Incorrect},
		{"doublon ne compense pas un oubli", []string{"A", "A"}, Incorrect},
	}
	for _, c := range cases {
		r := Evaluate(q, map[string][]string{"m": c.selected}, 4)
		if r.Outcomes["m"] != c.want {
			t.Errorf("%s : %s ; attendu %s", c.name, r.Outcomes["m"], c.want)
		}
	}
}

func TestUnansweredIsAFault(t *testing.T) {
	qs := series(10)
	answers := answersWithFaults(qs, 0)
	delete(answers, "q0")
	answers["q1"] = []string{} // cochée puis décochée
	r := Evaluate(qs, answers, 4)
	if r.UnansweredCount != 2 || r.FaultCount != 2 || r.CorrectCount != 8 || !r.Passed {
		t.Errorf("%+v", r)
	}

	// Rien coché du tout : ajourné.
	r = Evaluate(qs, nil, 4)
	if r.UnansweredCount != 10 || r.FaultCount != 10 || r.Passed {
		t.Errorf("examen vide : %+v", r)
	}
}

func TestFaultsCountQuestionsNotOptions(t *testing.T) {
	q := []Question{{ID: "m", CorrectOptionIDs: []string{"A", "B"}}}
	// Trois propositions fausses sur une même question = une seule faute.
	r := Evaluate(q, map[string][]string{"m": {"C", "D", "E"}}, 4)
	if r.FaultCount != 1 {
		t.Errorf("fautes = %d ; attendu 1", r.FaultCount)
	}
}
