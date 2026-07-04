package sample

import "fmt"

// This file is excluded by the FileName filter in no_fmt_print.rego, so the
// call below must NOT be reported.
func excludedGreet() {
	fmt.Println("excluded hello")
}
