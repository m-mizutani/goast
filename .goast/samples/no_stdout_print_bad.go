// Intentional violation fixture for no_stdout_print.rego, used by `goast test`.
// It lives under .goast/samples (NOT testdata/) because the rule skips /testdata/
// paths; keeping it here lets the rule fire on it during the policy test. As a
// dot-dir file it is ignored by the Go toolchain and never compiled.
package samples

import "fmt"

func bad() {
	fmt.Println("leaks to stdout")
	fmt.Printf("also %d\n", 1)
	fmt.Print("and this")
}
