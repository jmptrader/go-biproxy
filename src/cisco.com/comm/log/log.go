// Implements leveled logging but still retain pure stdlib logging
// (i.e., no external dependencies just for logging).
package log

import (
	stdlog "log"
	"fmt"
	"runtime"
	"path"
	"os"
)

func I(ft string, x ...interface{}) {
	logw(fmt.Sprintf("[ INFO ] " + ft, x...))
}

func W(ft string, x ...interface{}) {
	logw(fmt.Sprintf("[ WARN ] " + ft, x...))
}

func E(ft string, x ...interface{}) {
	logw(fmt.Sprintf("[ ERROR ] " + ft, x...))
}

func D(ft string, x ...interface{}) {
	logw(fmt.Sprintf("[ DEBUG ] " + ft, x...))
}

func F(ft string, x ...interface{}) {
	E(ft, x)
	os.Exit(1)
}

func logw(x string) {
	_, fp, ln, _ := runtime.Caller(2)
	file := path.Base(fp)
	stdlog.Println(fmt.Sprintf("%s:%d %s", file, ln, x))
}