package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRunCreatesSeedUsersOnce(t *testing.T) {
	dsn, db := newSchemaDSN(t, true)
	setSeedEnvironment(t, dsn)

	var first bytes.Buffer
	if err := run(context.Background(), &first); err != nil {
		t.Fatal(err)
	}
	want := "Created admin@oficina.local (admin).\nCreated gerente@oficina.local (manager).\nCreated atendente@oficina.local (attendant).\n"
	if got := first.String(); got != want {
		t.Fatalf("first run output = %q, want %q", got, want)
	}

	var second bytes.Buffer
	if err := run(context.Background(), &second); err != nil {
		t.Fatal(err)
	}
	want = "Skipped admin@oficina.local: already exists.\nSkipped gerente@oficina.local: already exists.\nSkipped atendente@oficina.local: already exists.\n"
	if got := second.String(); got != want {
		t.Fatalf("second run output = %q, want %q", got, want)
	}

	rows, err := db.Query(`SELECT u.email, ur.role_id FROM users u JOIN user_roles ur ON ur.user_id = u.id ORDER BY u.email`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	got := map[string]string{}
	for rows.Next() {
		var email, role string
		if err := rows.Scan(&email, &role); err != nil {
			t.Fatal(err)
		}
		got[email] = role
	}
	expected := map[string]string{"admin@oficina.local": "admin", "gerente@oficina.local": "manager", "atendente@oficina.local": "attendant"}
	if len(got) != len(expected) {
		t.Fatalf("users = %v, want %v", got, expected)
	}
	for email, role := range expected {
		if got[email] != role {
			t.Errorf("%s role = %q, want %q", email, got[email], role)
		}
	}
}

func TestMainSeedsUsingEnvironment(t *testing.T) {
	dsn, db := newSchemaDSN(t, true)
	setSeedEnvironment(t, dsn)

	main()

	var users int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil || users != 3 {
		t.Fatalf("users = %d, err = %v; want 3", users, err)
	}
}
