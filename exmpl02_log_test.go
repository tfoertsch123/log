package log_test

import (
	"os"
	"time"
	"github.com/tfoertsch123/log"
)

func Example_2() {
	// Use SetNow() to generate a fixed timestamp. This is only needed
	// to have this pass as a test so that go doc includes the example.
	defer func(orig func() time.Time){log.SetNow(orig)}(log.SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))

	var lg1 *log.Logger
	var lg2 *log.Logger

	fn1 := func() {
		lg1.Infof(`info in fn%d`, 1)
		lg1.Warnf(`info in fn%d`, 1)
	}

	fn2 := func() {
		lg2.Infof(`info in fn%d`, 2)
		lg2.Warnf(`info in fn%d`, 2)
	}

	// Create 2 loggers with different topics. Both are children of the
	// current logger. The current logger does not change. So, they are
	// also direct kids of the root logger.
	// Since at this point the root and current loggers are the same,
	// NewR() and NewK() are basically so too.
	lg1 = log.NewK(log.WithTopic(`TOPIC1`))
	lg2 = log.NewR(log.WithTopic(`TOPIC2`))

	// these changes propagate from the root/current logger to all other
	log.SetOutput(os.Stdout)
	log.SetTimeFmt("2006-01-02 15:04")

	fn1()
	fn2()

	// change the log level from WARN to INFO for all loggers
	log.SetLevel(log.DEBUG)

	fn1()
	fn2()

	// Output:
	// 2026-10-05 09:02 WARN [TOPIC1] info in fn1
	// 2026-10-05 09:02 WARN [TOPIC2] info in fn2
	// 2026-10-05 09:02 NOTICE Setting log level from WARN to DEBUG
	// 2026-10-05 09:02 INFO [TOPIC1] info in fn1
	// 2026-10-05 09:02 WARN [TOPIC1] info in fn1
	// 2026-10-05 09:02 INFO [TOPIC2] info in fn2
	// 2026-10-05 09:02 WARN [TOPIC2] info in fn2
}

// Local Variables:
// tab-width: 4
// End:
