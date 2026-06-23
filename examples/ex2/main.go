package main

import (
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

func main() {
    log.SetTimeFmt("2006-01-02 15:04:05")
    log.SetLevel(log.DEBUG)

	subsys1()
	subsys2()
}

// Local Variables:
// tab-width: 4
// End:
