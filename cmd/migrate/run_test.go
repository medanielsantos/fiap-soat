package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestRunAppliesMigrationsThenReportsUpToDate(t *testing.T) {
	dsn, _ := newSchemaDSN(t)
	setDatabaseURL(t, dsn)

	var first bytes.Buffer
	if err := run(context.Background(), []string{"up"}, &first); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.String(), "000001_create_customers.sql") {
		t.Fatalf("first run output = %q, want applied migrations", first.String())
	}

	var second bytes.Buffer
	if err := run(context.Background(), nil, &second); err != nil {
		t.Fatal(err)
	}
	if got := second.String(); got != "Database is already up to date.\n" {
		t.Fatalf("second run output = %q", got)
	}
}

func TestMainMigratesUsingProcessArguments(t *testing.T) {
	dsn, admin := newSchemaDSN(t)
	setDatabaseURL(t, dsn)
	oldArgs := os.Args
	os.Args = []string{"migrate", "up"}
	t.Cleanup(func() { os.Args = oldArgs })

	main()

	schema := strings.TrimPrefix(dsnOption(t, dsn), "-csearch_path=")
	var exists bool
	if err := admin.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, schema+".customers").Scan(&exists); err != nil || !exists {
		t.Fatalf("customers table exists = %v, err = %v", exists, err)
	}
}
