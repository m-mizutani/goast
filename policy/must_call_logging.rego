package goast

import future.keywords.every

fail[result] {
    allowList = ["main"]

    input.Kind == "FuncDecl"
    every allowed in allowList {
        input.Node.Name.Name != allowed
    }
    every stmt in input.Node.Body.List {
        stmt.X.Fun.Name != "logging"
    }

    result := {
        "msg": sprintf("logging() is not called in %s", [input.Node.Name.Name]),
        "pos": input.Node.Name.NamePos,
        "sev": "error",
    }
}
