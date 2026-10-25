package server

import "testing"

func TestLoginBudgetIsSharedAcrossCalls(t *testing.T) {
	ip := "203.0.113.77"
	for i := 0; i < 10; i++ {
		if !AllowLoginAttempt(ip) {
			t.Fatalf("attempt %d refused, want 10 allowed per minute", i+1)
		}
	}
	if AllowLoginAttempt(ip) {
		t.Fatal("11th attempt allowed; the GraphQL SingIn path must share the REST budget")
	}
}
