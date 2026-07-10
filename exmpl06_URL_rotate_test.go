package log_test

import (
	"os"
	"path/filepath"
	"fmt"
	"time"
	"github.com/tfoertsch123/log"
)

func ExampleParseURL_rotate() {
	// Use SetNow() to generate a fixed timestamp. This is only needed
	// to have this pass as a test so that go doc includes the example.
	defer func(orig func() time.Time){log.SetNow(orig)}(log.SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))

	// Setup a temp directory for log files
	dir, err := os.MkdirTemp("","Example_ParseURL_rotate")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)		// cleanup

	cat := func(fn string) {
		data, err := os.ReadFile(filepath.Join(dir, fn))
		if err != nil {
			panic(err)
		}
		fmt.Printf("file %s, len %d:\n%s\n", fn, len(data), string(data))
	}

	log.Root().Noticef("logfile is in %q", dir)
	defer log.Root().Close()	// reset/close all loggers

	logf := "test.log"
	url := "//DEBUG@"+dir+"/"+logf+"?"+
		"maxsize=100&"+
		"nbackups=5&"+
		"timeformat=2006-01-02%2015%3A04&"+
		"locdirs=1&"+
		"topic=EXAMPLE"

	if opts, e := log.ParseURL(url, nil); e == nil {
		log.NewC(opts...)
	} else {
		panic(fmt.Sprintf("ParseURL: %v", e))
	}

	log.Info("Info Message")
	log.Debug("Debug Message")	// this is line 29
	log.Debg2("Debg2 Message")

	// give it some time to rotate
	time.Sleep(100 * time.Millisecond)

	log.NewC(log.WithTopic("MAIN"))

	log.Info("Info Message")
	log.Debug("Debug Message")	// this is line 35
	log.Debg2("Debg2 Message")

	// We should have triggered a 2nd rotation, maxsize=100
	time.Sleep(100 * time.Millisecond)

	cat(logf)
	cat(logf + "~")
	cat(logf + "~2")

	// Output:
	// file test.log, len 0:
	//
	// file test.log~, len 122:
	// 2026-10-05 09:02 INFO [MAIN] Info Message
	// 2026-10-05 09:02 DEBUG [MAIN] (log/exmpl06_URL_rotate_test.go:60) Debug Message
	//
	// file test.log~2, len 128:
	// 2026-10-05 09:02 INFO [EXAMPLE] Info Message
	// 2026-10-05 09:02 DEBUG [EXAMPLE] (log/exmpl06_URL_rotate_test.go:51) Debug Message
}

// Local Variables:
// tab-width: 4
// End:
