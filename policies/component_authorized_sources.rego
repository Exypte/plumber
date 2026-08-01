# component-authorized-sources — flag `include: component:` references
# whose source doesn't match componentMustComeFromAuthorizedSources.
# trustedUrls. GitLab CI/CD Components execute arbitrary third-party code
# with the job's full context (variables, secrets, CI_JOB_TOKEN) — the
# GitLab analogue of a GitHub Actions "pwn request" supply-chain risk.
#
# `$VAR` / `${VAR}` references in both the component's source and each
# trustedUrls pattern are resolved against input.pipeline.globalVariables
# before comparison, so the shipped default
# `$CI_SERVER_FQDN/$CI_PROJECT_PATH/*` implicitly trusts the scanned
# project's own namespace once GlobalVariables carries those predefined
# variables. A name with no matching entry in globalVariables is left
# untouched (matches the Go-side ReplaceVariable fallback behaviour), so
# it will not incidentally match a wildcard pattern.
package component_authorized_sources

import rego.v1

deny contains finding if {
	input.config.componentAuthorizedSources
	some i
	inc := input.pipeline.includes[i]
	inc.kind == "component"
	inc.source != ""
	not _is_authorized(inc.source)
	finding := {
		"code":     "ISSUE-414",
		"severity": "high",
		"message":  sprintf("component %q comes from an untrusted source", [inc.source]),
		"job":      inc.source,
		"link":     inc.source,
		"status":   "unauthorized",
		"file":     object.get(inc, "originFile", ""),
		"line":     object.get(inc, "originLine", 0),
	}
}

_is_authorized(source) if {
	pattern := input.config.componentAuthorizedSources.trustedUrls[_]
	glob.match(_resolve_vars(pattern), null, _resolve_vars(source))
}

# _resolve_vars replaces every `$VAR` / `${VAR}` reference in s with its
# value from input.pipeline.globalVariables, in a single pass via
# strings.replace_n so multiple distinct variables in the same string are
# each substituted correctly.
_resolve_vars(s) := strings.replace_n(_var_patterns(s), s)

_var_patterns(s) := {p: v |
	some m in regex.find_all_string_submatch_n(`\$\{?([a-zA-Z_][a-zA-Z0-9_]*)\}?`, s, -1)
	name := m[1]
	v := input.pipeline.globalVariables[name]
	some p in {sprintf("$%s", [name]), sprintf("${%s}", [name])}
}
