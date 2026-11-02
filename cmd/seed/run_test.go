package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRunCreatesAdministratorOnce(t *testing.T) {
	dsn, db := newSchemaDSN(t, true)
	setSeedEnvironment(t, dsn)

	var first bytes.Buffer
	if err := run(context.Background(), &first); err != nil {
		t.Fatal(err)
	}
	if got := first.String(); got != "Initial administrator created.\n" {
		t.Fatalf("first run output = %q", got)
	}

	var second bytes.Buffer
	if err := run(context.Background(), &second); err != nil {
		t.Fatal(err)
	}
	if got := second.String(); got != "The user table is already initialized; seed skipped without changes.\n" {
		t.Fatalf("second run output = %q", got)
	}

	var admins int
	if err := db.QueryRow(`SELECT count(*) FROM users WHERE email = 'admin@example.com'`).Scan(&admins); err != nil || admins != 1 {
		t.Fatalf("admin rows = %d, err = %v; want exactly 1", admins, err)
	}
	var roles int
	if err := db.QueryRow(`SELECT count(*) FROM roles`).Scan(&roles); err != nil || roles == 0 {
		t.Fatalf("roles = %d, err = %v; want seeded roles", roles, err)
	}
}

func TestMainSeedsUsingEnvironment(t *testing.T) {
	dsn, db := newSchemaDSN(t, true)
	setSeedEnvironment(t, dsn)

	main()

	var users int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil || users != 1 {
		t.Fatalf("users = %d, err = %v; want 1", users, err)
	}
}
