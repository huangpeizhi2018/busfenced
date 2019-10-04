package pkg

import (
	"fmt"
	"runtime"
)

const Binary = "1.0.0-20190930"

func Version(app string) string {
	return fmt.Sprintf("%s v%s (built w/%s)", app, Binary, runtime.Version())
}
