package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFileParsesQuotesCommentsAndInvalidLines(t *testing.T) {
	keys := []string{"FIAP_DOTENV_DOUBLE", "FIAP_DOTENV_SINGLE", "FIAP_DOTENV_PLAIN", "FIAP_DOTENV_EMPTY_QUOTE"}
	for _, key := range keys {
		unsetForTest(t, key)
	}
	contents := "# comment\n\n" +
		"FIAP_DOTENV_DOUBLE = \"double value\"\n" +
		"FIAP_DOTENV_SINGLE='single value'\n" +
		"FIAP_DOTENV_PLAIN=plain=with=equals\n" +
		"FIAP_DOTENV_EMPTY_QUOTE=\"\n" +
		"line-without-separator\n" +
		"=value-without-key\n"
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	LoadDotEnvFile(path)

	want := map[string]string{
		"FIAP_DOTENV_DOUBLE":      "double value",
		"FIAP_DOTENV_SINGLE":      "single value",
		"FIAP_DOTENV_PLAIN":       "plain=with=equals",
		"FIAP_DOTENV_EMPTY_QUOTE": `"`,
	}
	for key, value := range want {
		if got := os.Getenv(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
	if _, exists := os.LookupEnv(""); exists {
		t.Error("an empty key must not be exported")
	}
}

func TestLoadDotEnvFileIgnoresMissingFile(t *testing.T) {
	LoadDotEnvFile(filepath.Join(t.TempDir(), "missing.env"))
}
