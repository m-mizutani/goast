package goast

# Exported functions must take context.Context as the first parameter, with a
# conventional exemption list — exercises FuncDecl matching, regex, negation
# and helper rules.

fail contains res if {
	input.Kind == "FuncDecl"
	regex.match("^[A-Z]", input.Node.Name.Name)
	not is_exempt
	not has_context_first_param

	res := {
		"msg": sprintf("exported function %q must take context.Context first", [input.Node.Name.Name]),
		"pos": input.Node.Name.NamePos,
		"sev": "ERROR",
	}
}

exempt_prefix := {"New", "With", "Is"}

is_exempt if {
	some p in exempt_prefix
	startswith(input.Node.Name.Name, p)
}

has_context_first_param if {
	first := input.Node.Type.Params.List[0]
	first.Type.X.Name == "context"
	first.Type.Sel.Name == "Context"
}
