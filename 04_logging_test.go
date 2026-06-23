package log

import (
	"testing"
	"regexp"
	"strings"
	"fmt"
	"time"
	"runtime"
)

func TestTimeFmt(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	Root().prnt(NOTICE, `test`, ``)
	if b.String() != "1966-04-28 09:02:05.987654 NOTICE test\n" {
		t.Errorf(`got %v`, b.String())
	}
	b.Reset()

	// test default time fmt
	if Root().GetTimeFmt() != "2006-01-02 15:04:05.000000" {
		t.Error(`default time fmt unexpected`)
	}

	Root().SetTimeFmt("2006-01-02 15:04:05")
	if GetTimeFmt() != "2006-01-02 15:04:05" {
		t.Error(`time fmt unexpected after setting`)
	}

	func(){
		NewC(WithTimeFmt("2006-01-02 15:04:05.000000000"))
		defer Close()
		L().prnt(PANIC, `test`, ``)
	}()
	L().prnt(PANIC, `test2`, ``)

	if b.String() != "1966-04-28 09:02:05.987654321 PANIC test\n" +
		"1966-04-28 09:02:05 PANIC test2\n" {
		t.Errorf(`got %v`, b.String())
	}
	b.Reset()

	func(){
		NewC()
		defer Close()
		lg := L().New()
		SetTimeFmt("2006-01-02 15:04:05.000")
		L().prnt(PANIC, `test3`, ``)

		if GetTimeFmt() != "2006-01-02 15:04:05.000" {
			t.Errorf("unexpected timefmt, got %v", GetTimeFmt())
		}
		if lg.GetTimeFmt() != "2006-01-02 15:04:05.000" {
			t.Errorf("unexpected timefmt, got %v", lg.GetTimeFmt())
		}		
	}()
	SetTimeFmt("2006-01-02 15:04:05.000000")
	L().prnt(PANIC, `test4`, ``)

	if b.String() != "1966-04-28 09:02:05.987 PANIC test3\n" +
		"1966-04-28 09:02:05.987654 PANIC test4\n" {
		t.Errorf(`got %v`, b.String())
	}
	b.Reset()
}

func TestTopic(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	lnr := func() int {
		pc := make([]uintptr, 2)
		nframes := runtime.Callers(2, pc)
		if nframes <= 0 {return -1}
		frames := runtime.CallersFrames(pc)
		f, _ := frames.Next()
		return f.Line
	}

	Root().SetTimeFmt("2006-01-02")
	Root().SetLevel(DEBG5+1)
	Root().SetLocDirectories(0)
	var ln int
	func(){
		NewC(WithTopic(`tpc`))
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln = lnr()+1
			Log(lv, ls)
		}
	}()
	exp := fmt.Sprintf(
		"1966-04-28 NOTICE Setting log level from WARN to DEBG5\n"+
		"1966-04-28 NOTICE [tpc] notice\n"+
		"1966-04-28 PANIC [tpc] panic\n"+
		"1966-04-28 ERROR [tpc] error\n"+
		"1966-04-28 WARN [tpc] warn\n"+
		"1966-04-28 INFO [tpc] info\n"+
		"1966-04-28 DEBUG [tpc] (04_logging_test.go:%[1]d) debug\n"+
		"1966-04-28 DEBG2 [tpc] (04_logging_test.go:%[1]d) debg2\n"+
		"1966-04-28 DEBG3 [tpc] (04_logging_test.go:%[1]d) debg3\n"+
		"1966-04-28 DEBG4 [tpc] (04_logging_test.go:%[1]d) debg4\n"+
		"1966-04-28 DEBG5 [tpc] (04_logging_test.go:%[1]d) debg5\n",
		ln)
	if b.String() != exp {
		t.Errorf(`got %v`, b.String())
		t.Errorf(`exp %v`, exp)
	}
	b.Reset()
}

func TestLog(t *testing.T) {
	// mocking now()
	oldNow := SetNow(func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	})
	defer func(){SetNow(oldNow)}()

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(WithLevel(DEBG5), WithMinLocation(DEBG5+10))
		defer Close()
		if L().GetMinLocation() != DEBG5+1 {
			t.Errorf("MinLoc cannot exceed DEBG5+1, got %v",
				L().GetMinLocation())
		}
		lg := L().New()
		SetMinLocation(NOTICE)
		if GetMinLocation() != DEBG5+1 {
			t.Errorf("MinLoc cannot be less than PANIC, got %v",
				GetMinLocation())
		}
		if lg.GetMinLocation() != DEBG5+1 {
			t.Errorf("MinLoc cannot be less than PANIC, got %v",
				lg.GetMinLocation())
		}
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Log(lv, ls)
			if b.String() != "1966-04-28 09:02:05.987654 "+s+" "+ls+"\n" {
				t.Errorf(`got %v`, b.String())
			}
			b.Reset()
		}

		if GetLocDirectories() != -1 {
			t.Errorf("nlocdir is -1 by default, got %v",
				GetLocDirectories())
		}	

		SetLocDirectories(2)

		if GetLocDirectories() != 2 {
			t.Errorf("nlocdir should be 2, got %v", GetLocDirectories())
		}	

		if lg.GetLocDirectories() != 2 {
			t.Errorf("lg.nlocdir should be 2, got %v", lg.GetLocDirectories())
		}	
	}()

	lnr := func() int {
		pc := make([]uintptr, 2)
		nframes := runtime.Callers(2, pc)
		if nframes <= 0 {return -1}
		frames := runtime.CallersFrames(pc)
		f, _ := frames.Next()
		return f.Line
	}

	func(){
		NewC(WithLevel(DEBG5), WithTimeFmt("2006-01-02"))
		SetMinLocation(PANIC)
		SetLocDirectories(0)
		defer Close()
		for _, s := range levels[1:] { // skip NOTICE
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln := lnr()+1		// this and the next line must not be
			Log(lv, ls)			// separated
			if b.String() != fmt.Sprintf(
				"%s %s (04_logging_test.go:%d) %s\n",
				"1966-04-28", s, ln, ls) {
				t.Errorf(`got %v`, b.String())
			}
			b.Reset()
		}
	}()

	func(){
		NewC(WithLevel(NOTICE), WithTimeFmt("2006-01-02"), WithMinLocation(-1))
		defer Close()
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Log(lv, ls)
		}
		exp := ("1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(PANIC), WithTimeFmt("2006-01-02"))
		defer Close()
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Log(lv, ls)
		}
		exp := ("1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(ERROR), WithTimeFmt("2006-01-02"))
		defer Close()
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Log(lv, ls)
		}
		exp := ("1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(WARN), WithTimeFmt("2006-01-02"))
		defer Close()
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Log(lv, ls)
		}
		exp := ("1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(INFO), WithTimeFmt("2006-01-02"))
		defer Close()
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Log(lv, ls)
		}
		exp := ("1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n"+
			    "1966-04-28 INFO info\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(DEBUG), WithTimeFmt("2006-01-02"), WithLocDirectories(0))
		defer Close()
		var ln int
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln = lnr()+1		// this and the next line must not be
			Log(lv, ls)			// separated
		}
		exp := fmt.Sprintf(
			"1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n"+
			    "1966-04-28 INFO info\n"+
			    "1966-04-28 DEBUG (04_logging_test.go:%d) debug\n", ln)
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(DEBG2), WithTimeFmt("2006-01-02"), WithLocDirectories(0))
		defer Close()
		var ln int
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln = lnr()+1		// this and the next line must not be
			Log(lv, ls)			// separated
		}
		exp := fmt.Sprintf(
			"1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n"+
			    "1966-04-28 INFO info\n"+
			    "1966-04-28 DEBUG (04_logging_test.go:%[1]d) debug\n"+
			    "1966-04-28 DEBG2 (04_logging_test.go:%[1]d) debg2\n", ln)
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(DEBG3), WithTimeFmt("2006-01-02"), WithLocDirectories(0))
		defer Close()
		var ln int
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln = lnr()+1		// this and the next line must not be
			Log(lv, ls)			// separated
		}
		exp := fmt.Sprintf(
			"1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n"+
			    "1966-04-28 INFO info\n"+
			    "1966-04-28 DEBUG (04_logging_test.go:%[1]d) debug\n"+
			    "1966-04-28 DEBG2 (04_logging_test.go:%[1]d) debg2\n"+
			    "1966-04-28 DEBG3 (04_logging_test.go:%[1]d) debg3\n", ln)
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(DEBG4), WithTimeFmt("2006-01-02"), WithLocDirectories(0))
		defer Close()
		var ln int
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln = lnr()+1		// this and the next line must not be
			Log(lv, ls)			// separated
		}
		exp := fmt.Sprintf(
			"1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n"+
			    "1966-04-28 INFO info\n"+
			    "1966-04-28 DEBUG (04_logging_test.go:%[1]d) debug\n"+
			    "1966-04-28 DEBG2 (04_logging_test.go:%[1]d) debg2\n"+
			    "1966-04-28 DEBG3 (04_logging_test.go:%[1]d) debg3\n"+
			    "1966-04-28 DEBG4 (04_logging_test.go:%[1]d) debg4\n", ln)
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(WithLevel(DEBG5),
			WithTimeFmt("2006-01-02"),
			WithMinLocation(DEBG2),
			WithLocDirectories(0))
		defer Close()
		var ln int
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln = lnr()+1		// this and the next line must not be
			Log(lv, ls)			// separated
		}
		exp := fmt.Sprintf(
			"1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n"+
			    "1966-04-28 INFO info\n"+
			    "1966-04-28 DEBUG debug\n"+
			    "1966-04-28 DEBG2 (04_logging_test.go:%[1]d) debg2\n"+
			    "1966-04-28 DEBG3 (04_logging_test.go:%[1]d) debg3\n"+
			    "1966-04-28 DEBG4 (04_logging_test.go:%[1]d) debg4\n"+
			    "1966-04-28 DEBG5 (04_logging_test.go:%[1]d) debg5\n", ln)
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		// DEBG5+N is the same as DEBG5
		NewC(WithLevel(DEBG5+1),WithTimeFmt("2006-01-02"),WithLocDirectories(0))
		defer Close()

		if GetMinLocation() != DEBUG {
			t.Errorf(`MinLocation should be DEBUG`, GetMinLocation())
		}

		var ln int
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			ln = lnr()+1		// this and the next line must not be
			Log(lv, ls)			// separated
		}
		exp := fmt.Sprintf(
			"1966-04-28 NOTICE notice\n"+
			    "1966-04-28 PANIC panic\n"+
			    "1966-04-28 ERROR error\n"+
			    "1966-04-28 WARN warn\n"+
			    "1966-04-28 INFO info\n"+
			    "1966-04-28 DEBUG (04_logging_test.go:%[1]d) debug\n"+
			    "1966-04-28 DEBG2 (04_logging_test.go:%[1]d) debg2\n"+
			    "1966-04-28 DEBG3 (04_logging_test.go:%[1]d) debg3\n"+
			    "1966-04-28 DEBG4 (04_logging_test.go:%[1]d) debg4\n"+
			    "1966-04-28 DEBG5 (04_logging_test.go:%[1]d) debg5\n", ln)
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestLogf(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(WithLevel(DEBG5), WithMinLocation(DEBG5+1))
		defer Close()
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Logf(lv, "%s %d", ls, 19)
			if b.String() != "1966-04-28 09:02:05.987654 "+s+" "+ls+" 19\n" {
				t.Errorf(`got %v`, b.String())
			}
			b.Reset()
		}
	}()

	func(){
		NewC(WithLevel(DEBUG), WithTimeFmt("2006-01-02"))
		defer Close()
		Logf(INFO, "%s %d %v", "test", 21, -2)
		if b.String() != "1966-04-28 INFO test 21 -2\n" {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		Logf(DEBUG, "%s %d %v", "test2", 21, -2)
		re := regexp.MustCompile(
			`^1966-04-28 DEBUG \(/.+/04_logging_test.go:[0-9]+\) test2 21 -2\n`,
		)
		if ! re.MatchString(b.String()) {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestLogl(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(WithLevel(DEBG5), WithMinLocation(DEBG5+1))
		defer Close()
		for _, s := range levels {
			ls := strings.ToLower(s)
			lv, _ := ParseLevel(s)
			Logl(lv, "%s %d %v", ls, func()interface{}{return 21}, -1)
			if b.String() != "1966-04-28 09:02:05.987654 "+s+" "+ls+" 21 -1\n" {
				t.Errorf(`got %v`, b.String())
			}
			b.Reset()
		}
	}()

	func(){
		NewC(WithLevel(DEBUG), WithTimeFmt("2006-01-02"))
		defer Close()
		Logl(INFO, "%s %d %v", "test", func()interface{}{return 21}, -2)
		if b.String() != "1966-04-28 INFO test 21 -2\n" {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		Logl(DEBUG, "%s %d %v", "test2", func()interface{}{return 21}, -2)
		re := regexp.MustCompile(
			`^1966-04-28 DEBUG \(/.+/04_logging_test.go:[0-9]+\) test2 21 -2\n`,
		)
		if ! re.MatchString(b.String()) {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestSomeEdgeCases(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(WithLevel(DEBG5), WithTimeFmt("2006-01-02"))
		SetMinLocation(PANIC)
		// SetLocDirectories(0)
		defer Close()

		Log(DEBUG, `test`)
		re := regexp.MustCompile(
			`^1966-04-28 DEBUG \(/.+/04_logging_test.go:[0-9]+\) test\n`,
		)
		if ! re.MatchString(b.String()) {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		oldCallers := _callers
		_callers = func(skip int, pc []uintptr) int {return 0}
		Log(DEBUG, `test2`)
		if b.String() != "1966-04-28 DEBUG test2\n" {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
		_callers = oldCallers

		oldNFrames := _nframes
		_nframes = 1
		Log(DEBUG, `test2`)
		if b.String() != "1966-04-28 DEBUG test2\n" {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		_nframes = 2
		Log(DEBUG, `test2`)
		if b.String() != "1966-04-28 DEBUG test2\n" {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
		_nframes = oldNFrames
	}()
}

func TestNotice(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(WithMinLocation(DEBG5+1), WithTimeFmt("2006-01-02"))
		defer Close()
		Notice("notice")
		Noticef("<%v> <%v>", "notice", 123)
		Noticel("<%v> <%v>", "notice", func()interface{} {return 124})
		exp := ("1966-04-28 NOTICE notice\n"+
			    "1966-04-28 NOTICE <notice> <123>\n"+
			    "1966-04-28 NOTICE <notice> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Notice("notice")
		L().Noticef("<%v> <%v>", "notice", 123)
		L().Noticel("<%v> <%v>", "notice", func()interface{} {return 124})
		exp = ("1966-04-28 NOTICE notice\n"+
			   "1966-04-28 NOTICE <notice> <123>\n"+
			   "1966-04-28 NOTICE <notice> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestPanic(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	oldExit := _exit
	defer func(){_exit = oldExit}()
	exitCalled := 0
	_exit = func(i int){exitCalled++; return}

	func(){
		NewC(WithMinLocation(DEBG5+1), WithTimeFmt("2006-01-02"))
		defer Close()
		Panic("pan")
		Panicf("<%v> <%v>", "pan", 123)
		Panicl("<%v> <%v>", "pan", func()interface{} {return 124})
		exp := ("1966-04-28 PANIC pan\n"+
			    "1966-04-28 PANIC <pan> <123>\n"+
			    "1966-04-28 PANIC <pan> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Panic("pan")
		L().Panicf("<%v> <%v>", "pan", 123)
		L().Panicl("<%v> <%v>", "pan", func()interface{} {return 124})
		exp = ("1966-04-28 PANIC pan\n"+
			   "1966-04-28 PANIC <pan> <123>\n"+
			   "1966-04-28 PANIC <pan> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		if exitCalled != 6 {
			t.Errorf(`os.Exit should have been called 6 times, got %v`,
				exitCalled)
		}
	}()
}

func TestError(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(WithMinLocation(DEBG5+1), WithTimeFmt("2006-01-02"))
		defer Close()
		Error("err")
		Errorf("<%v> <%v>", "err", 123)
		Errorl("<%v> <%v>", "err", func()interface{} {return 124})
		exp := ("1966-04-28 ERROR err\n"+
			    "1966-04-28 ERROR <err> <123>\n"+
			    "1966-04-28 ERROR <err> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Error("err")
		L().Errorf("<%v> <%v>", "err", 123)
		L().Errorl("<%v> <%v>", "err", func()interface{} {return 124})
		exp = ("1966-04-28 ERROR err\n"+
			   "1966-04-28 ERROR <err> <123>\n"+
			   "1966-04-28 ERROR <err> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestWarn(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(WithMinLocation(DEBG5+1), WithTimeFmt("2006-01-02"))
		defer Close()
		Warn("wrn")
		Warnf("<%v> <%v>", "wrn", 123)
		Warnl("<%v> <%v>", "wrn", func()interface{} {return 124})
		exp := ("1966-04-28 WARN wrn\n"+
			    "1966-04-28 WARN <wrn> <123>\n"+
			    "1966-04-28 WARN <wrn> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Warn("wrn")
		L().Warnf("<%v> <%v>", "wrn", 123)
		L().Warnl("<%v> <%v>", "wrn", func()interface{} {return 124})
		exp = ("1966-04-28 WARN wrn\n"+
			   "1966-04-28 WARN <wrn> <123>\n"+
			   "1966-04-28 WARN <wrn> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestInfo(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(
			WithLevel(DEBG5),
			WithMinLocation(DEBG5+1),
			WithTimeFmt("2006-01-02"),
		)
		defer Close()
		Info("inf")
		Infof("<%v> <%v>", "inf", 123)
		Infol("<%v> <%v>", "inf", func()interface{} {return 124})
		exp := ("1966-04-28 INFO inf\n"+
			    "1966-04-28 INFO <inf> <123>\n"+
			    "1966-04-28 INFO <inf> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Info("inf")
		L().Infof("<%v> <%v>", "inf", 123)
		L().Infol("<%v> <%v>", "inf", func()interface{} {return 124})
		exp = ("1966-04-28 INFO inf\n"+
			   "1966-04-28 INFO <inf> <123>\n"+
			   "1966-04-28 INFO <inf> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestDebug(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(
			WithLevel(DEBG5),
			WithMinLocation(DEBG5+1),
			WithTimeFmt("2006-01-02"),
		)
		defer Close()
		Debug("dbg")
		Debugf("<%v> <%v>", "dbg", 123)
		Debugl("<%v> <%v>", "dbg", func()interface{} {return 124})
		exp := ("1966-04-28 DEBUG dbg\n"+
			    "1966-04-28 DEBUG <dbg> <123>\n"+
			    "1966-04-28 DEBUG <dbg> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Debug("dbg")
		L().Debugf("<%v> <%v>", "dbg", 123)
		L().Debugl("<%v> <%v>", "dbg", func()interface{} {return 124})
		exp = ("1966-04-28 DEBUG dbg\n"+
			   "1966-04-28 DEBUG <dbg> <123>\n"+
			   "1966-04-28 DEBUG <dbg> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestDebg2(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(
			WithLevel(DEBG5),
			WithMinLocation(DEBG5+1),
			WithTimeFmt("2006-01-02"),
		)
		defer Close()
		Debg2("dbg2")
		Debg2f("<%v> <%v>", "dbg2", 123)
		Debg2l("<%v> <%v>", "dbg2", func()interface{} {return 124})
		exp := ("1966-04-28 DEBG2 dbg2\n"+
			    "1966-04-28 DEBG2 <dbg2> <123>\n"+
			    "1966-04-28 DEBG2 <dbg2> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Debg2("dbg2")
		L().Debg2f("<%v> <%v>", "dbg2", 123)
		L().Debg2l("<%v> <%v>", "dbg2", func()interface{} {return 124})
		exp = ("1966-04-28 DEBG2 dbg2\n"+
			   "1966-04-28 DEBG2 <dbg2> <123>\n"+
			   "1966-04-28 DEBG2 <dbg2> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestDebg3(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(
			WithLevel(DEBG5),
			WithMinLocation(DEBG5+1),
			WithTimeFmt("2006-01-02"),
		)
		defer Close()
		Debg3("dbg3")
		Debg3f("<%v> <%v>", "dbg3", 123)
		Debg3l("<%v> <%v>", "dbg3", func()interface{} {return 124})
		exp := ("1966-04-28 DEBG3 dbg3\n"+
			    "1966-04-28 DEBG3 <dbg3> <123>\n"+
			    "1966-04-28 DEBG3 <dbg3> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Debg3("dbg3")
		L().Debg3f("<%v> <%v>", "dbg3", 123)
		L().Debg3l("<%v> <%v>", "dbg3", func()interface{} {return 124})
		exp = ("1966-04-28 DEBG3 dbg3\n"+
			   "1966-04-28 DEBG3 <dbg3> <123>\n"+
			   "1966-04-28 DEBG3 <dbg3> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestDebg4(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(
			WithLevel(DEBG5),
			WithMinLocation(DEBG5+1),
			WithTimeFmt("2006-01-02"),
		)
		defer Close()
		Debg4("dbg4")
		Debg4f("<%v> <%v>", "dbg4", 123)
		Debg4l("<%v> <%v>", "dbg4", func()interface{} {return 124})
		exp := ("1966-04-28 DEBG4 dbg4\n"+
			    "1966-04-28 DEBG4 <dbg4> <123>\n"+
			    "1966-04-28 DEBG4 <dbg4> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Debg4("dbg4")
		L().Debg4f("<%v> <%v>", "dbg4", 123)
		L().Debg4l("<%v> <%v>", "dbg4", func()interface{} {return 124})
		exp = ("1966-04-28 DEBG4 dbg4\n"+
			   "1966-04-28 DEBG4 <dbg4> <123>\n"+
			   "1966-04-28 DEBG4 <dbg4> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestDebg5(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(
			WithLevel(DEBG5),
			WithMinLocation(DEBG5+1),
			WithTimeFmt("2006-01-02"),
		)
		defer Close()
		Debg5("dbg5")
		Debg5f("<%v> <%v>", "dbg5", 123)
		Debg5l("<%v> <%v>", "dbg5", func()interface{} {return 124})
		exp := ("1966-04-28 DEBG5 dbg5\n"+
			    "1966-04-28 DEBG5 <dbg5> <123>\n"+
			    "1966-04-28 DEBG5 <dbg5> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()

		L().Debg5("dbg5")
		L().Debg5f("<%v> <%v>", "dbg5", 123)
		L().Debg5l("<%v> <%v>", "dbg5", func()interface{} {return 124})
		exp = ("1966-04-28 DEBG5 dbg5\n"+
			   "1966-04-28 DEBG5 <dbg5> <123>\n"+
			   "1966-04-28 DEBG5 <dbg5> <124>\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}

func TestClosed(t *testing.T) {
	// mocking now()
	oldNow := _now
	defer func(){_now = oldNow}()
	_now = func() time.Time {
		return time.Date(1966, 4, 28, 9, 2, 5, 987654321, time.UTC)
	}

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()		// reinit

	kid := NewK(WithTopic(`KID`))
	NewC(WithTopic(`CUR`))
	kid.Notice(`kid`)
	Notice(`current`)
	Root().Notice(`root`)

	Root().Close()

	Root().out = &b

	kid.Notice(`kid`)
	Notice(`current2`)
	Root().Notice(`root2`)
	kid.Noticef(`kid-f`)
	Noticef(`current2-f`)
	Root().Noticef(`root2-f`)
	kid.Noticel(`kid-l`)
	Noticel(`current2-l`)
	Root().Noticel(`root2-l`)

	exp := ("1966-04-28 09:02:05.987654 NOTICE [KID] kid\n"+
			"1966-04-28 09:02:05.987654 NOTICE [CUR] current\n"+
			"1966-04-28 09:02:05.987654 NOTICE root\n"+
			// here the root logger was closed
			// so, the kid should not log anything.
			// the root should still work
			// and the current logger should be the same as root
			// but it now does not have a topic.
			"1966-04-28 09:02:05.987654 NOTICE current2\n"+
			"1966-04-28 09:02:05.987654 NOTICE root2\n"+
			"1966-04-28 09:02:05.987654 NOTICE current2-f\n"+
			"1966-04-28 09:02:05.987654 NOTICE root2-f\n"+
			"1966-04-28 09:02:05.987654 NOTICE current2-l\n"+
			"1966-04-28 09:02:05.987654 NOTICE root2-l\n")

	if b.String() != exp {
		t.Errorf(`got %v`, b.String())
	}
	b.Reset()
}

// Local Variables:
// tab-width: 4
// End:
