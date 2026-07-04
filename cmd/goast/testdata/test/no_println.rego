package goast

fail contains res if {
	input.Kind == "CallExpr"
	input.Node.Fun.X.Name == "fmt"
	input.Node.Fun.Sel.Name == "Println"

	res := {
		"msg": "do not use fmt.Println",
		"pos": input.Node.Fun.X.NamePos,
		"sev": "ERROR",
	}
}
