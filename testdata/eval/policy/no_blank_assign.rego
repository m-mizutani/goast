package goast

# Flag assignments to the blank identifier — exercises AssignStmt, array
# iteration over Lhs and a non-ERROR severity.

fail contains res if {
	input.Kind == "AssignStmt"
	lhs := input.Node.Lhs[_]
	lhs.Name == "_"

	res := {
		"msg": "do not assign to the blank identifier",
		"pos": lhs.NamePos,
		"sev": "WARNING",
	}
}
