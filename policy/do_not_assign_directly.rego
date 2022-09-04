package goast

fail[result] {
	input.Kind == "AssignStmt"
	rhs := input.Node.Rhs[_]
	rhs.Type.Name == "User"

	result := {
		"msg": "Do not assign User type directly (value)",
		"pos": rhs.Type.NamePos,
		"sev": "error",
	}
}

fail[result] {
	input.Kind == "AssignStmt"
	rhs := input.Node.Rhs[_]
	rhs.Op == 17 # AND
	rhs.X.Type.Name == "User"

	result := {
		"msg": "Do not assign User type directly (pointer)",
		"pos": rhs.X.Type.NamePos,
		"sev": "error",
	}
}
