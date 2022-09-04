# goast

Go [AST](https://pkg.go.dev/go/ast) (Abstract Syntax Tree) based static analysis tool with [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/).

![](https://user-images.githubusercontent.com/605953/187052104-03525b0d-cb7c-44b9-b395-b7b3692a0cc2.png)

## Motivation

There are a lot of static analysis tools for Go language. They inspect Go source code with best practices. However, we need to care not only common best practice but also internal rules of individual, team or organization when reviewing code. Additionally, some kind of function and resource has a rule to use them (e.g. required initialize at first). It may be difficult to check such original rules by a common static analysis tool.

`goast` is static analysis tool with [OPA]([Rego](https://www.openpolicyagent.org/docs/latest/policy-language/)) that is generic policy engine of [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/). It can separate static analysis tool to _implementation_ and _policy_ that a user can customize.

## Features

- Dump Go AST as JSON format (excluding ast.Object to avoid reference cycle)
- Evaluate Go source code with Rego policy
- Output the evaluation result as text or JSON format (compatible with [reviewdog](https://github.com/reviewdog/reviewdog))

## Usage

### Install

```
$ go install github.com/m-mizutani/goast@latest
```

### Dump source code to confirm AST

`dump` subcommand output AST as JSON format. `--line` can specify line number and `--func` can specify function name for dump.

```bash
$ goast dump --line 6 examples/println/main.go | jq
{
  "Path": "examples/println/main.go",
  "Node": {
    "X": {
      "Fun": {
        "X": {
          "NamePos": 44,
          "Name": "fmt",
          "Obj": null
        },
        "Sel": {
          "NamePos": 48,
          "Name": "Println",
          "Obj": null
        }
      },
      "Lparen": 55,
      "Args": [
        {
          "ValuePos": 56,
          "Kind": 9,
          "Value": "\"hello\""
        }
      ],
      "Ellipsis": 0,
      "Rparen": 63
    }
  },
  "Kind": "ExprStmt"
}
```

### Write Rego policy

Here is example of a policy to prohibit `fmt.Println`.

```rego
package goast

fail[res] {
    input.Kind == "ExprStmt"
    input.Node.X.Fun.X == "fmt"
    input.Node.X.Fun.Sel == "Println"

    res := {
        "msg": "do not use fmt.Println",
        "pos": input.Node.X.NamePos,
        "sev": "ERROR",
    }
}
```

`goast`'s policy rule is following.

- Package name: `goast`
- Input
  - `Path`: Source code file path
  - `Node`: Dumped AST (without *ast.Object)
  - `Kind`: Type of node
- Output
  - `fail`: A set of violation results
    - `pos`: Integer of *Pos (e.g. `NamePos`). It will be converted to line number and column of source code file
    - `msg`: Error message
    - `sev`: Severity. Choose one from `INFO`, `WARNING` or `ERROR`

### Evaluation

`eval` subcommand evaluates go source code (file or directly recursively) with Rego policy file(s).

```bash
$ goast eval -p ./policy/do_not_use_println.rego examples/println/main.go
[examples/println/main.go:6] - do not use fmt.Println

        Detected 1 violations

```

`--format, -f` option can specify output format `text` or `json`. JSON schema is according to [Reviewdog Diagnostic Format](https://github.com/reviewdog/reviewdog/tree/master/proto/rdf#rdjson).

### Debug

Also, you can use original `opa` command to debug policy. A schema of `dump` output is same with one to be evaluated. Then, `opa` command can use it with Rego file(s). An example is following.

```bash
goast . dump -l 6 examples/println/main.go | opa eval -b ./policy/ -I data
{
  "result": [
    {
      "expressions": [
        {
          "value": {
            "goast": {
              "fail": [
                {
                  "msg": "do not use fmt.Println",
                  "pos": 44,
                  "sev": "ERROR"
                }
              ]
            }
          },
          "text": "data",
          "location": {
            "row": 1,
            "col": 1
          }
        }
      ]
    }
  ]
}
```

## License

Apache License v2
