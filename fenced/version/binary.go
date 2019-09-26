package version

import (
	"fmt"
	"runtime"
)

const Binary = "1.0.0-alpha-20190926"

func String(app string) string {
	return fmt.Sprintf("%s v%s (built w/%s)", app, Binary, runtime.Version())
}
