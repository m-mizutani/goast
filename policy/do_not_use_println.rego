package goast

fail[res] {
    input.Kind == "ExprStmt"
    input.Node.X.Fun.X.Name == "fmt"
    input.Node.X.Fun.Sel.Name == "Println"

    res := {
        "msg": "do not use fmt.Println",
        "pos": input.Node.X.Fun.X.NamePos,
        "sev": "ERROR",
    }
}
