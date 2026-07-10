package log_test

import (
	"fmt"
	"time"
	"github.com/tfoertsch123/log"
)

func ExampleParseURL_stdout() {
	// Use SetNow() to generate a fixed timestamp. This is only needed
	// to have this pass as a test so that go doc includes the example.
	defer func(orig func() time.Time){log.SetNow(orig)}(log.SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))
	defer log.Root().Close()	// reset/close all loggers

	// This URL sends the log output to Stdout. It configures os.Stdout
	// as output. Instead of stdout, stderr can also be used.
	url := "//DEBUG@stdout?"+
		"timeformat=2006-01-02%2015%3A04&"+
		"locdirs=1&"+
		"topic=EXAMPLE"

	if opts, e := log.ParseURL(url, nil); e == nil {
		log.NewC(opts...)
	} else {
		panic(fmt.Sprintf("ParseURL: %v", e))
	}

	log.Info("Info Message")
	log.Debug("Debug Message")	// this is line 31
	log.Debg2("Debg2 Message")

	log.NewC(log.WithTopic("MAIN"))

	log.Info("Info Message")
	log.Debug("Debug Message")	// this is line 37
	log.Debg2("Debg2 Message")

	// Output:
	// 2026-10-05 09:02 INFO [EXAMPLE] Info Message
	// 2026-10-05 09:02 DEBUG [EXAMPLE] (log/exmpl05_URL_stdout_test.go:31) Debug Message
	// 2026-10-05 09:02 INFO [MAIN] Info Message
	// 2026-10-05 09:02 DEBUG [MAIN] (log/exmpl05_URL_stdout_test.go:37) Debug Message
}

// Local Variables:
// tab-width: 4
// End:
