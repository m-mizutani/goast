package goast

# goast's own coding rule, enforced on goast by `goast eval` (see [eval] in
# .goast.toml). Every byte of command output goes through an explicit io.Writer
# (fmt.Fprintf and friends), never to os.Stdout implicitly, so a stray debug
# print cannot leak into --output or the rdjson stream. The implicit-stdout
# fmt.Print family is therefore banned; fmt.Fprint*/Sprint* stay allowed.
#
# The rule governs command *output*, so it applies to production code only.
# goast has no exclude flag and walks a target directory recursively, so a bulk
# target like `cmd` also visits test files and the intentional violation fixtures
# under testdata/. Both are skipped by Path — the documented goast idiom (see
# references/patterns.md) — which lets [eval] point at whole directories. The
# rule's own good/bad samples live under .goast/samples/ (not testdata/, not
# *_test.go) so `goast test` can still make the rule fire on them.
fail contains res if {
    input.Kind == "ExprStmt"
    input.Node.X.Fun.X.Name == "fmt"
    input.Node.X.Fun.Sel.Name in {"Print", "Printf", "Println"}
    not contains(input.Path, "/testdata/")
    not endswith(input.Path, "_test.go")

    res := {
        "msg": sprintf("do not use fmt.%s; write to an explicit io.Writer via fmt.Fprint*", [input.Node.X.Fun.Sel.Name]),
        "pos": input.Node.X.Fun.X.NamePos,
        "sev": "ERROR",
    }
}
