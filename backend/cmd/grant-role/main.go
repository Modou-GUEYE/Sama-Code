// Commande grant-role : attribue (ou retire) un rôle à un compte existant, à partir
// de son numéro. Sert à créer le premier administrateur.
//
//	go run ./cmd/grant-role -phone "77 123 45 67" -role admin
//	go run ./cmd/grant-role -phone 771234567 -role editor -revoke
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/jackc/pgx/v5"

	"samacode/backend/internal/auth"
	"samacode/backend/internal/database"
)

var roles = []string{"candidate", "editor", "validator", "admin"}

func main() {
	phone := flag.String("phone", "", "numéro du compte (ex. 77 123 45 67)")
	role := flag.String("role", "", "rôle : candidate, editor, validator ou admin")
	revoke := flag.Bool("revoke", false, "retirer le rôle au lieu de l'attribuer")
	flag.Parse()

	if err := run(*phone, *role, *revoke); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(rawPhone, role string, revoke bool) error {
	if !slices.Contains(roles, role) {
		return fmt.Errorf("rôle inconnu %q (attendu : %v)", role, roles)
	}
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

	var userID, name string
	err = pool.QueryRow(ctx, `SELECT id, display_name FROM users WHERE phone_number = $1`, phone).Scan(&userID, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("aucun compte avec le numéro %s : inscrivez-vous d'abord dans l'application", phone)
	}
	if err != nil {
		return err
	}

	if revoke {
		_, err = pool.Exec(ctx,
			`DELETE FROM user_roles WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE code = $2)`, userID, role)
	} else {
		_, err = pool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE code = $2 ON CONFLICT DO NOTHING`, userID, role)
	}
	if err != nil {
		return err
	}
	action := "attribué à"
	if revoke {
		action = "retiré à"
	}
	fmt.Printf("Rôle %s %s %s (%s). Effectif à la prochaine connexion.\n", role, action, name, phone)
	return nil
}
