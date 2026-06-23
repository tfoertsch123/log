package main

import (
	"github.com/tfoertsch123/log"
)

var base *log.Logger = log.NewR(log.WithTimeFmt("2006-01-02 15:04:05"))
var tpc1 *log.Logger = base.New(log.WithTopic("TOPIC1"))
var tpc2 *log.Logger = base.New(log.WithTopic("TOPIC2"))

func subsys1() {
    tpc1.Info(`message`)
}

func subsys2() {
    tpc2.Info(`message`)
}

func main() {
    base.SetLevel(log.DEBUG)

	subsys1()
	subsys2()
}

// Local Variables:
// tab-width: 4
// End:
