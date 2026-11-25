package tools

import "testing"

// Every legal edge must be allowed and every other pair refused, for all
// three state machines. The tool-level tests then prove the tables are
// actually consulted on every path.
func TestTransitionTablesExhaustive(t *testing.T) {
	builds := []string{buildAssigned, buildRunning, buildWaitingUser, buildSucceeded, buildFailed, buildCancelled}
	legalBuild := map[[2]string]bool{
		{buildAssigned, buildRunning}: true, {buildAssigned, buildCancelled}: true,
		{buildRunning, buildWaitingUser}: true, {buildRunning, buildSucceeded}: true,
		{buildRunning, buildFailed}: true, {buildRunning, buildCancelled}: true,
		{buildWaitingUser, buildRunning}: true, {buildWaitingUser, buildCancelled}: true,
	}
	for _, from := range builds {
		for _, to := range builds {
			if got, want := buildAllowed(from, to), legalBuild[[2]string{from, to}]; got != want {
				t.Errorf("buildAllowed(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
	turns := []string{turnOpen, turnWaitingUser, turnAnswered, turnExpired, turnClosed}
	legalTurn := map[[2]string]bool{
		{turnOpen, turnWaitingUser}: true, {turnOpen, turnClosed}: true, {turnOpen, turnExpired}: true,
		{turnWaitingUser, turnAnswered}: true, {turnWaitingUser, turnClosed}: true, {turnWaitingUser, turnExpired}: true,
		{turnAnswered, turnClosed}: true,
	}
	for _, from := range turns {
		for _, to := range turns {
			if got, want := turnAllowed(from, to), legalTurn[[2]string{from, to}]; got != want {
				t.Errorf("turnAllowed(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
	questions := []string{questionOpen, questionAnswered, questionExpired, questionWithdrawn}
	legalQ := map[[2]string]bool{
		{questionOpen, questionAnswered}: true, {questionOpen, questionExpired}: true,
		{questionOpen, questionWithdrawn}: true,
	}
	for _, from := range questions {
		for _, to := range questions {
			if got, want := questionAllowed(from, to), legalQ[[2]string{from, to}]; got != want {
				t.Errorf("questionAllowed(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}
