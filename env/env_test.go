package env

import "testing"

func TestEnvs(t *testing.T) {
	envs := Envs()
	if len(envs) != 3 {
		t.Fatalf("Envs len = %d, want 3", len(envs))
	}
	if envs[0] != EnvLocal || envs[1] != EnvProduction || envs[2] != EnvTesting {
		t.Errorf("Envs = %v", envs)
	}
}

func TestParseEnv(t *testing.T) {
	cases := []struct {
		in   string
		want Env
		err  bool
	}{
		{"", EnvLocal, false},
		{"local", EnvLocal, false},
		{"LOCAL", EnvLocal, false},
		{"production", EnvProduction, false},
		{"PRODUCTION", EnvProduction, false},
		{"testing", EnvTesting, false},
		{"staging", "", true},
	}
	for _, c := range cases {
		got, err := ParseEnv(c.in)
		if (err != nil) != c.err {
			t.Errorf("ParseEnv(%q) error = %v, want err=%v", c.in, err, c.err)
		}
		if got != c.want {
			t.Errorf("ParseEnv(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
