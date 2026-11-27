package dbenv

import (
	"strings"
	"testing"
)

func TestRefuseProductionBranch(t *testing.T) {
	cases := []struct {
		name     string
		pol      EndpointPolicy
		wantErr  bool
		wantFrag string
	}{
		{"dev branch allowed", EndpointPolicy{BranchName: "dev"}, false, ""},
		{"feature branch allowed", EndpointPolicy{BranchName: "feat-x", ProjectName: "p"}, false, ""},
		{"production refused", EndpointPolicy{BranchName: "production"}, true, "denylisted branch name"},
		{"PROD refused case-insensitive", EndpointPolicy{BranchName: "PROD"}, true, "denylisted branch name"},
		{"whitespace-padded name refused", EndpointPolicy{BranchName: "  production  "}, true, "denylisted branch name"},
		{"Prod mixed case refused", EndpointPolicy{BranchName: "Prod"}, true, "denylisted branch name"},
		{"platform-protected refused", EndpointPolicy{BranchName: "analytics", Protected: true}, true, "platform-protected"},
		{"protected wins with project context", EndpointPolicy{BranchName: "analytics", ProjectName: "main", Protected: true}, true, "in project main"},
		{"empty name refused", EndpointPolicy{}, true, "unknown identity"},
		{"whitespace name refused", EndpointPolicy{BranchName: "  "}, true, "unknown identity"},
		// Near-misses must NOT refuse (exact match, no substrings).
		{"production-2 allowed", EndpointPolicy{BranchName: "production-2"}, false, ""},
		{"myprod allowed", EndpointPolicy{BranchName: "myprod"}, false, ""},
		{"staging allowed", EndpointPolicy{BranchName: "staging"}, false, ""},
		{"main allowed", EndpointPolicy{BranchName: "main"}, false, ""},
	}
	for _, tc := range cases {
		err := RefuseProductionBranch(tc.pol)
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err=%v, wantErr=%v", tc.name, err, tc.wantErr)
			continue
		}
		if err != nil && !strings.Contains(err.Error(), tc.wantFrag) {
			t.Errorf("%s: error %q lacks %q", tc.name, err.Error(), tc.wantFrag)
		}
	}
}
