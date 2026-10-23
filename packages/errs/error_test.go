package errs

import "testing"

func TestExitCodeFromInt(t *testing.T) {
	cases := []struct {
		in   int
		want ExitCode
	}{
		{0, ExitCodeSuccess},
		{1, ExitCodeFailure},
		{2, ExitCodeUsage},
		{42, ExitCodeFailure},
	}
	for _, c := range cases {
		if got := ExitCodeFromInt(c.in); got != c.want {
			t.Errorf("ExitCodeFromInt(%d) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestExitCodeInt(t *testing.T) {
	if got := ExitCodeUsage.Int(); got != 2 {
		t.Errorf("ExitCodeUsage.Int() = %d, want 2", got)
	}
}

func TestError(t *testing.T) {
	err := UsageError("missing --name")
	if err.Code != ExitCodeUsage {
		t.Errorf("Code = %v, want %v", err.Code, ExitCodeUsage)
	}
	if err.Error() != "missing --name" {
		t.Errorf("Error() = %q, want %q", err.Error(), "missing --name")
	}
	if err.ExitCode() != ExitCodeUsage {
		t.Errorf("ExitCode() = %v, want %v", err.ExitCode(), ExitCodeUsage)
	}
	fail := FailureError("internal crash")
	if fail.Code != ExitCodeFailure {
		t.Errorf("Code = %v, want %v", fail.Code, ExitCodeFailure)
	}
}
