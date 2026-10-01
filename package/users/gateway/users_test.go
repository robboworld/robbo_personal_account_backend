package gateway

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(fmt.Errorf("update: %w", &pgconn.PgError{Code: "23505"})) {
		t.Fatal("wrapped 23505 not detected")
	}
	for _, err := range []error{
		&pgconn.PgError{Code: "23503"}, // foreign key: not "email already used"
		errors.New("connection reset"),
		nil,
	} {
		if isUniqueViolation(err) {
			t.Fatalf("%v treated as a unique violation", err)
		}
	}
}
