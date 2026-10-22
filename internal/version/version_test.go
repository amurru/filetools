package version

import "testing"

func TestStringFallback(t *testing.T) {
	orig := Version
	Version = ""
	defer func() { Version = orig }()
	if got := String(); got != "dev" {
		t.Fatalf("String() = %q, want dev", got)
	}
}

func TestStringSet(t *testing.T) {
	orig := Version
	Version = "1.2.3"
	defer func() { Version = orig }()
	if got := String(); got != "1.2.3" {
		t.Fatalf("String() = %q, want 1.2.3", got)
	}
}

func TestCommitStringFallback(t *testing.T) {
	orig := Commit
	Commit = ""
	defer func() { Commit = orig }()
	if got := CommitString(); got != "none" {
		t.Fatalf("CommitString() = %q, want none", got)
	}
}

func TestCommitDateStringFallback(t *testing.T) {
	orig := CommitDate
	CommitDate = ""
	defer func() { CommitDate = orig }()
	if got := CommitDateString(); got != "unknown" {
		t.Fatalf("CommitDateString() = %q, want unknown", got)
	}
}

func TestTreeStateStringFallback(t *testing.T) {
	orig := TreeState
	TreeState = ""
	defer func() { TreeState = orig }()
	if got := TreeStateString(); got != "unknown" {
		t.Fatalf("TreeStateString() = %q, want unknown", got)
	}
}
