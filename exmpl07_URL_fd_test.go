package log_test

import (
	"os"
	"fmt"
	"time"
	"syscall"
	"github.com/tfoertsch123/log"
)

func Example_ParseURL_fd() {
	// Use SetNow() to generate a fixed timestamp. This is only needed
	// to have this pass as a test so that go doc includes the example.
	defer func(orig func() time.Time){log.SetNow(orig)}(log.SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))
	defer log.Root().Close()	// reset/close all loggers

	// This example demonstrates the use of an open file descriptor as
	// the log destination. In Bash, that could be achieved like so:
	//
	//   program 5>logfile --logurl="file://5?timeformat&..."
	//
	// Here we just dup() stdout. We have to dup() it because the file
	// descriptor is internally assigned to an *os.File which has a cleanup
	// function attached so that the file will be closed by the next
	// run of the garbage collector when the logger becomes unreachable.
	fd, err := syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		panic("unexpected error in syscall.Dup()")
	}
	defer syscall.Close(fd)

	url := fmt.Sprintf(
		"//%d?"+
		"timeformat=2006-01-02%2015%3A04&"+
		"locdirs=1&"+
		"topic=EXAMPLE&"+
		"level=debug",
		fd,
	)
	log.Root().Noticef("Using URL %q", url)

	if opts, e := log.ParseURL(url, nil); e == nil {
		log.NewC(opts...)
	} else {
		panic(fmt.Sprintf("ParseURL: %v", e))
	}

	log.Info("Info Message")
	log.Debug("Debug Message")	// this is line 51
	log.Debg2("Debg2 Message")

	log.NewC(log.WithTopic("MAIN"))

	log.Info("Info Message")
	log.Debug("Debug Message")	// this is line 57
	log.Debg2("Debg2 Message")

	// Output:
	// 2026-10-05:02 INFO [EXAMPLE] Info Message
	// 2026-10-05:02 DEBUG [EXAMPLE] (log/exmpl07_URL_fd_test.go:51) Debug Message
	// 2026-10-05:02 INFO [MAIN] Info Message
	// 2026-10-05:02 DEBUG [MAIN] (log/exmpl07_URL_fd_test.go:57) Debug Message
}

// Local Variables:
// tab-width: 4
// End:
