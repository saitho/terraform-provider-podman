package provider

import (
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"regexp"
	"time"
)

func validateStringMatchesPattern(pattern string) schema.SchemaValidateDiagFunc {
	return func(v interface{}, k cty.Path) diag.Diagnostics {
		compiledRegex, err := regexp.Compile(pattern)
		var errors diag.Diagnostics
		if err != nil {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("%q regex does not compile", pattern),
				Detail:        fmt.Sprintf("%q regex does not compile", pattern),
				AttributePath: nil,
			}}
		}

		value := v.(string)
		if !compiledRegex.MatchString(value) {
			errors = append(errors, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("%q doesn't match the pattern (%q): %q", k, pattern, value),
				Detail:        fmt.Sprintf("%q doesn't match the pattern (%q): %q", k, pattern, value),
				AttributePath: nil,
			})
		}
		return errors
	}
}

func validateDockerContainerPath(v interface{}, k cty.Path) diag.Diagnostics {

	value := v.(string)
	if !regexp.MustCompile(`^[a-zA-Z]:\\|^/`).MatchString(value) {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("%q must be an absolute path", k),
			Detail:        fmt.Sprintf("%q must be an absolute path", k),
			AttributePath: nil,
		}}
	}
	return nil
}

func validateDurationGeq0() schema.SchemaValidateDiagFunc {
	return func(v interface{}, k cty.Path) diag.Diagnostics {
		value := v.(string)
		dur, err := time.ParseDuration(value)
		if err != nil {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("%q is not a valid duration", k),
				Detail:        fmt.Sprintf("%q is not a valid duration", k),
				AttributePath: nil,
			}}
		}
		if dur < 0 {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       "duration must not be negative",
				Detail:        "duration must not be negative",
				AttributePath: nil,
			}}

		}
		return nil
	}
}

func validateIntegerGeqThan(threshold int) schema.SchemaValidateDiagFunc {
	return func(v interface{}, k cty.Path) diag.Diagnostics {
		value := v.(int)
		if value < threshold {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("%q cannot be lower than %d", k, threshold),
				Detail:        fmt.Sprintf("%q cannot be lower than %d", k, threshold),
				AttributePath: nil,
			}}
		}
		return nil
	}
}
