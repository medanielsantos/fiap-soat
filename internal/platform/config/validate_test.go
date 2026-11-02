package config

import "testing"

func TestValidateAcceptsKnownEnvironments(t *testing.T) {
	for _, env := range []string{EnvLocal, EnvStaging, EnvProduction} {
		if err := (Config{AppEnv: env}).Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", env, err)
		}
	}
}

func TestValidateRejectsUnknownEnvironment(t *testing.T) {
	for _, env := range []string{"", "dev", "Production", "prod"} {
		err := (Config{AppEnv: env}).Validate()
		if err == nil {
			t.Errorf("Validate(%q) = nil, want error", env)
		}
	}
}

func TestLoadValidatedReportsInvalidEnvironment(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("APP_ENV", "dev")
	cfg, err := LoadValidated()
	if err == nil || err.Error() != `invalid APP_ENV "dev": use local, staging or production` {
		t.Fatalf("err = %v, want invalid APP_ENV error", err)
	}
	if cfg.AppEnv != "dev" {
		t.Errorf("AppEnv = %q, want dev", cfg.AppEnv)
	}
}

func TestLoadValidatedDefaultsToProduction(t *testing.T) {
	clearConfigEnvironment(t)
	cfg, err := LoadValidated()
	if err != nil || cfg.AppEnv != EnvProduction {
		t.Fatalf("LoadValidated() = %q, %v; want production, nil", cfg.AppEnv, err)
	}
}
