package goast

fail[result] {
    func := input.File.Decls[_]
    func.Body
    func.Name.Name != "main" # ignore func main()

    count(func.Type.Params.List) == 0
    result := {
        "msg": sprintf("%s has no arguments, ctx is required at least", [func.Name.Name]),
        "pos": func.Name.NamePos,
        "sev": "error",
    }
}

fail[result] {
    func := input.File.Decls[_]
    func.Body
    func.Type.Params.List[0].Type.X.Name != "context.Context"

    result := {
        "msg": sprintf("%s first argument must be context.Context, actual is %s",
        [func.Name.Name, func.Type.Params.List[0].Type.X.Name]),
        "pos": func.Name.NamePos,
        "sev": "error",
    }
}
