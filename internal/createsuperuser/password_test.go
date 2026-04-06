package createsuperuser

import "testing"

func TestPasswordIssues_Strong(t *testing.T) {
	p := "GoodPassw0rd!"
	issues := PasswordIssues(p)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestPasswordIssues_Weak(t *testing.T) {
	issues := PasswordIssues("short1A!")
	if len(issues) == 0 {
		t.Fatal("expected length issue")
	}
	issues = PasswordIssues("nouppercase123!")
	if len(issues) == 0 {
		t.Fatal("expected uppercase issue")
	}
	issues = PasswordIssues("NOLOWERCASE123!")
	if len(issues) == 0 {
		t.Fatal("expected lowercase issue")
	}
	issues = PasswordIssues("NoDigitsHere!!")
	if len(issues) == 0 {
		t.Fatal("expected digit issue")
	}
	issues = PasswordIssues("NoSpecial9Chars")
	if len(issues) == 0 {
		t.Fatal("expected special issue")
	}
}
