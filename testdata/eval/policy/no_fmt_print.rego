package goast

# Ban fmt.Print* and the print/println builtins, mirroring a real-world policy
# shape: helper rules, constant sets, sprintf messages and metadata filters.

fail contains res if {
	input.Kind == "CallExpr"
	not is_excluded_file
	input.Node.Fun.X.Name == "fmt"
	startswith(input.Node.Fun.Sel.Name, "Print")

	res := {
		"msg": sprintf("do not use fmt.%s", [input.Node.Fun.Sel.Name]),
		"pos": input.Node.Fun.X.NamePos,
		"sev": "ERROR",
	}
}

fail contains res if {
	input.Kind == "CallExpr"
	not is_excluded_file
	builtin_print[input.Node.Fun.Name]

	res := {
		"msg": sprintf("do not use the builtin %s()", [input.Node.Fun.Name]),
		"pos": input.Node.Fun.NamePos,
		"sev": "ERROR",
	}
}

builtin_print := {"print", "println"}

is_excluded_file if {
	endswith(input.FileName, "_ignore.go")
}
