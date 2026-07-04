// Compliant fixture for no_stdout_print.rego: all output goes through an
// explicit io.Writer, and Sprintf only builds a string. None should be flagged.
// Under .goast/samples (a dot-dir) so the Go toolchain ignores it.
package samples

import (
	"fmt"
	"io"
	"os"
)

func good(w io.Writer) {
	fmt.Fprintf(w, "ok %d\n", 1)
	fmt.Fprintln(os.Stderr, "diagnostic")
	_ = fmt.Sprintf("built %d", 2)
}
