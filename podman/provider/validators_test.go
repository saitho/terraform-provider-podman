package provider

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"testing"
)

func TestValidateStringMatchesPattern(t *testing.T) {
	pattern := `^(pause|continue-mate|break)$`
	cases := map[string]struct {
		Value         interface{}
		ExpectedDiags diag.Diagnostics
	}{
		"pause": {
			Value:         "pause",
			ExpectedDiags: nil,
		},
		"doesnotmatch": {
			Value: "doesnotmatch",
			ExpectedDiags: diag.Diagnostics{
				{
					Severity: diag.Error,
				},
			},
		},
		"continue-mate": {
			Value:         "continue-mate",
			ExpectedDiags: nil,
		},
	}

	fn := validateStringMatchesPattern(pattern)
	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			diags := fn(tc.Value, cty.Path{})

			checkDiagnostics(t, tn, diags, tc.ExpectedDiags)
		})
	}
}

func checkDiagnostics(t *testing.T, tn string, got, expected diag.Diagnostics) {
	if len(got) != len(expected) {
		t.Fatalf("%s: wrong number of diags, expected %d, got %d", tn, len(expected), len(got))
	}
	for j := range got {
		if got[j].Severity != expected[j].Severity {
			t.Fatalf("%s: expected severity %v, got %v", tn, expected[j].Severity, got[j].Severity)
		}
		if !got[j].AttributePath.Equals(expected[j].AttributePath) {
			t.Fatalf("%s: attribute paths do not match expected: %v, got %v", tn, expected[j].AttributePath, got[j].AttributePath)
		}
	}
}
