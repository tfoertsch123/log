package main

import (
	"os"
	"github.com/tfoertsch123/log"
)

func subsys1() {
    log.NewC(log.WithTopic(`TOPIC1`))
    defer log.Close()

    log.Info(`message`)
}

func subsys2() {
    log.NewC(log.WithTopic(`TOPIC2`))
    defer log.Close()

    log.Info(`message`)
}

func fib() {
    log.NewC(log.WithTopic(`FIB`))
    defer log.Close()

	fib := func(yield func(uint64) bool) {
		var x, y uint64 = 1, 1
		for {
			if !yield(x) {return}
			x, y = y, x+y
		}
	}

    log.Debugl(
		`sum of first 1000 fibonacci numbers is %v`,
		log.Lazy(func() interface{} {
			var sum uint64
			i := 0
			for fb := range fib {
				sum += fb
				i++
				if i >= 1000 {break}
			}
			return sum
		}),
	)
}

func main() {
	if len(os.Args) >= 2 {
		opts, err := log.ParseURL(
			os.Args[1],
			func(e error) {
				log.Panicf("Could not create output for %q: %v",
					os.Args[1], e)
			},
		)
		if err != nil {
			log.Panicf("Don't understand URL %q: %v", os.Args[1], err)
		}
		log.NewC(opts...)
		defer log.Close()
	}

	subsys1()
	fib()
	subsys2()
}

// Local Variables:
// tab-width: 4
// End:
