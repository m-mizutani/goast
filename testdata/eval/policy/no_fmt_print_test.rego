package goast

# Unit tests for no_fmt_print.rego, deliberately in the SAME package as the
# policies (mirroring real-world layouts such as hecatoncheires' .goast/).
# These rules must never influence `goast eval` output; the golden test pins
# that. Run with: opa test testdata/eval/policy

fmt_call(sel) := {
	"Path": "pkg/x/x.go",
	"FileName": "x.go",
	"DirName": "pkg/x",
	"Kind": "CallExpr",
	"Node": {"Fun": {
		"X": {"Name": "fmt", "NamePos": 100},
		"Sel": {"Name": sel, "NamePos": 104},
	}},
}

ident_call(name) := {
	"Path": "pkg/x/x.go",
	"FileName": "x.go",
	"DirName": "pkg/x",
	"Kind": "CallExpr",
	"Node": {"Fun": {"Name": name, "NamePos": 42}},
}

test_println_flagged if {
	some res in fail with input as fmt_call("Println")
	res.sev == "ERROR"
}

test_printf_flagged if {
	count(fail) == 1 with input as fmt_call("Printf")
}

test_sprintf_not_flagged if {
	count(fail) == 0 with input as fmt_call("Sprintf")
}

test_builtin_print_flagged if {
	count(fail) == 1 with input as ident_call("print")
}

test_regular_call_not_flagged if {
	count(fail) == 0 with input as ident_call("logging")
}

test_excluded_file_not_flagged if {
	excluded := json.patch(fmt_call("Println"), [{"op": "replace", "path": "/FileName", "value": "x_ignore.go"}])
	count(fail) == 0 with input as excluded
}
