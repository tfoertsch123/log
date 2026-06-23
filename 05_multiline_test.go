package log

import (
	"testing"
	"time"
	"strings"
)

func TestMultiLine(t *testing.T) {
	// mocking now()
	defer func(orig func() time.Time){SetNow(orig)}(SetNow(func() time.Time {
		return time.Date(2026, 10, 5, 9, 2, 5, 987654321, time.UTC)
	}))

	var b strings.Builder
	// avoid using SetOutput() because it prints to the old output
	oldOut := Root().out
	defer func(){Root().out = oldOut}()
	Root().out = &b

	defer Root().Close()

	func(){
		NewC(
			WithTimeFmt("2006-01-02"),
		)
		defer Close()

		L().Warn("line1\nline2\nline3\n")
		exp := ("2026-10-05 WARN line1\nline2\nline3\n\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(
			WithTimeFmt("2006-01-02"),
			WithTopic("ML"),
			WithMultiLine(true),
		)
		defer Close()

		L().Warn("line1\nline2\nline3\n")
		exp := ("2026-10-05 WARN [ML] line1\n"+
			    "2026-10-05 WARN [ML] line2\n"+
				"2026-10-05 WARN [ML] line3\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(
			WithTimeFmt("2006-01-02"),
			WithTopic("ML:\\r\\n"),
			WithMultiLine(true),
		)
		defer Close()

		L().Warn("line1\r\nline2\r\nline3\r\n")
		exp := ("2026-10-05 WARN [ML:\\r\\n] line1\n"+
			    "2026-10-05 WARN [ML:\\r\\n] line2\n"+
				"2026-10-05 WARN [ML:\\r\\n] line3\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()

	func(){
		NewC(
			WithTimeFmt("2006-01-02"),
			WithTopic("ML:trailing \\r"),
			WithMultiLine(true),
		)
		defer Close()

		L().Warn("line1\r\nline2\r\nline3\r")
		exp := ("2026-10-05 WARN [ML:trailing \\r] line1\n"+
			    "2026-10-05 WARN [ML:trailing \\r] line2\n"+
				"2026-10-05 WARN [ML:trailing \\r] line3\n")
		if b.String() != exp {
			t.Errorf(`got %v`, b.String())
		}
		b.Reset()
	}()
}


// Local Variables:
// tab-width: 4
// End:
