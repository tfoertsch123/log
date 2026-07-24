package log_test

import (
	"os"
	"time"
	"math"
	"github.com/tfoertsch123/log"
)

func Example_lazy() {
	// Use SetNow() to generate a fixed timestamp. This is only needed
	// to have this pass as a test so that go doc includes the example.
	defer func(orig func() time.Time){log.SetNow(orig)}(log.SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))
	log.SetOutput(os.Stdout)
	defer log.Root().Close()	// reset/close all loggers

	complex_work := func() string {
		time.Sleep(100 * time.Millisecond)
		return "done"
	}

	ms := time.Millisecond

	// The log level is WARN. So, nothing will be printed.
	// Yet it still takes time.
	now := time.Now()

	log.Infof("complex_work: %s", complex_work())

	log.Warnf("complex_work took %vms +/- 20ms",
		20*math.Round(float64(time.Now().Sub(now))/float64(20*ms)),
	)

	now = time.Now()

	// Now with lazy execution. This should take almost no time
	// since complex_work() is not executed.
	log.Infol("complex_work: %s",
		log.Lazy(func() interface{} {return complex_work()}))

	log.Warnf("complex_work with lazy execution took %vms +/- 20ms",
		20*math.Round(float64(time.Now().Sub(now))/float64(20*ms)),
	)

	// Now let's raise the log level and repeat the previous.
	// It should take ~100ms
	log.SetLevel(log.INFO)

	now = time.Now()

	log.Infol("complex_work: %s",
		log.Lazy(func() interface{} {return complex_work()}))

	log.Warnf("complex_work if actually done took %vms +/- 20ms",
		20*math.Round(float64(time.Now().Sub(now))/float64(20*ms)),
	)

	// Output:
	// 2026-10-05 09:02:05.987654 WARN complex_work took 100ms +/- 20ms
	// 2026-10-05 09:02:05.987654 WARN complex_work with lazy execution took 0ms +/- 20ms
	// 2026-10-05 09:02:05.987654 NOTICE Setting log level from WARN to INFO
	// 2026-10-05 09:02:05.987654 INFO complex_work: done
	// 2026-10-05 09:02:05.987654 WARN complex_work if actually done took 100ms +/- 20ms
}

// Local Variables:
// tab-width: 4
// End:
