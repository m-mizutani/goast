package goast

# A second, independent rule. Paired with no_println.rego it lets the engine
# cache test key on two distinct policy paths and check order-independence.
fail contains res if {
	input.Kind == "CallExpr"
	input.Node.Fun.X.Name == "fmt"
	input.Node.Fun.Sel.Name == "Print"

	res := {
		"msg": "do not use fmt.Print",
		"pos": input.Node.Fun.X.NamePos,
		"sev": "ERROR",
	}
}
