package log_test

import (
	"os"
	"time"
	"github.com/tfoertsch123/log"
)

func Example_multiline() {
	// Use SetNow() to generate a fixed timestamp. This is only needed
	// to have this pass as a test so that go doc includes the example.
	defer func(orig func() time.Time){log.SetNow(orig)}(log.SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))
	log.SetOutput(os.Stdout)
	defer log.Root().Close()	// reset/close all loggers

	log.Notice("this\nis a\nmultiline\nmessage")

	log.NewC(log.WithMultiLine(true))

	log.Notice("this\nis a\nmultiline\nmessage")
	// Output:
	// 2026-10-05 09:02 NOTICE this
	// is a
	// multiline
	// message
	// 2026-10-05 09:02 NOTICE this
	// 2026-10-05 09:02 NOTICE is a
	// 2026-10-05 09:02 NOTICE multiline
	// 2026-10-05 09:02 NOTICE message
}

// Local Variables:
// tab-width: 4
// End:
