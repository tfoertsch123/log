package log_test

import (
	"os"
	"time"
	"github.com/tfoertsch123/log"
)

func ExampleNewC() {
	// Use SetNow() to generate a fixed timestamp. This is only needed
	// to have this pass as a test so that go doc includes the example.
	defer func(orig func() time.Time){log.SetNow(orig)}(log.SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))

	// This example uses the current logger.
	log.SetLevel(log.DEBUG)		// change the log level from WARN to DEBUG
	log.SetOutput(os.Stdout)	// change the output from Stderr to Stdout
	log.SetLocDirectories(1)	// we want 1 directory component in the location output
	log.Info(`message without a topic`)
	log.Debug(`with code location`)
	log.Debg2(`not printed`)	// because DEBUG < DEBG2

	// Create a new logger and make it the current one.
	// Everything but the topic is inherited from the root.
	log.NewC(log.WithTopic(`TPC`))
	log.Debug(`with topic and location`)

	// Close the current logger. The root logger becomes current again.
	log.Close()
	log.Warn(`get back the old logger`)

	// Close the root logger. This reinits it.
	log.Close()
	// Output:
	// 2026-10-05 09:02:05.987654 INFO message without a topic
	// 2026-10-05 09:02:05.987654 DEBUG (log/exmpl01_log_test.go:21) with code location
	// 2026-10-05 09:02:05.987654 DEBUG [TPC] (log/exmpl01_log_test.go:27) with topic and location
	// 2026-10-05 09:02:05.987654 WARN get back the old logger
}

// Local Variables:
// tab-width: 4
// End:
