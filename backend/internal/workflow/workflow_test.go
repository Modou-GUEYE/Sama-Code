package workflow

import (
	"errors"
	"testing"

	"samacode/backend/internal/httpx"
)

func TestCheck(t *testing.T) {
	editor := httpx.Principal{Roles: []string{RoleEditor}}
	validator := httpx.Principal{Roles: []string{RoleValidator}}
	admin := httpx.Principal{Roles: []string{RoleAdmin}}

	cases := []struct {
		from, to string
		who      httpx.Principal
		want     error
	}{
		{Draft, InReview, editor, nil},
		{InReview, Published, editor, ErrForbidden},
		{InReview, Published, validator, nil},
		{Draft, Published, admin, ErrInvalidTransition}, // pas de raccourci, même pour l'admin
		{Published, Draft, validator, nil},
		{Archived, Draft, validator, ErrForbidden},
		{Archived, Draft, admin, nil},
		{Draft, "deleted", admin, ErrInvalidStatus},
	}
	for _, c := range cases {
		err := Check(c.from, c.to, c.who)
		if c.want == nil && err != nil || c.want != nil && !sameCode(err, c.want) {
			t.Errorf("%s → %s par %v = %v ; attendu %v", c.from, c.to, c.who.Roles, err, c.want)
		}
	}
}

func sameCode(err, want error) bool {
	var a, b *httpx.Error
	return errors.As(err, &a) && errors.As(want, &b) && a.Code == b.Code
}
