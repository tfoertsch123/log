package log

import (
	"testing"
	"io"
	"github.com/dlclark/regexp2"
)

type NamedWriter struct {
	w io.Writer
	name string
}
func (w *NamedWriter) Name() string {return w.name}
func (w *NamedWriter) String() string {return w.name}
func (w *NamedWriter) Write(b []byte) (int, error) {return w.w.Write(b)}

type StringWriter struct {
	w io.Writer
	name string
}
func (w *StringWriter) String() string {return w.name}
func (w *StringWriter) Write(b []byte) (int, error) {return w.w.Write(b)}

func TestLoggerString(t *testing.T) {
	out1 := io.Discard
	out2 := &NamedWriter{
		w: io.Discard,
		name: `NAMEDWRITER`,
	}
	out3 := &StringWriter{
		w: io.Discard,
		name: `STRINGWRITER`,
	}
	lg1 := L().New(WithOutput(out1), WithTopic(`lg1`), WithLevel(DEBUG))
	lg2 := L().New(WithOutput(out2), WithTopic(`lg2`), WithLevel(DEBG2))
	lg3 := L().New(WithOutput(out3), WithTopic(`lg3`), WithLevel(DEBG3))
	lg3.New(WithLevel(DEBG4))
	lg5 := lg3.New(WithLevel(DEBG5))

	lg2.Close()

	re1 := regexp2.MustCompile(
		`0x[0-9a-f]+:{prev:0x[0-9a-f]+, out:\{\}, topic:`+"`"+
			` \[lg1\]`+"`" +`, level:DEBUG, tmfmt:`+
			"`"+`2006-01-02 15:04:05.000000`+"`"+`, kids\[\]}`,
		0,
	)
	re2 := regexp2.MustCompile(
		`(0x[0-9a-f]+)\(closed\):{prev:\1, out:NAMEDWRITER, topic:`+"`"+
			` \[lg2\]` + "`" + `, level:DEBG2, tmfmt:`+
			"`"+`2006-01-02 15:04:05.000000`+"`"+`, kids\[\]}`,
		0,
	)
	re3 := regexp2.MustCompile(
		`0x[0-9a-f]+:{prev:0x[0-9a-f]+, out:STRINGWRITER, topic:`+"`"+
			` \[lg3\]` + "`" + `, level:DEBG3, tmfmt:`+
			"`"+`2006-01-02 15:04:05.000000`+"`"+
			`, kids\[0x[0-9a-f]+, 0x[0-9a-f]+\]}`,
		0,
	)
	if ok, _ := re1.MatchString(lg1.String()); !ok {
		t.Errorf(`lg1 (%v) does not match regex`, lg1)
	}
	if ok, _ := re2.MatchString(lg2.String()); !ok {
		t.Errorf(`lg2 (%v) does not match regex`, lg2)
	}
	if ok, _ := re3.MatchString(lg3.String()); !ok {
		t.Errorf(`lg3 (%v) does not match regex`, lg3)
	}

	var found *Logger
	for _, lg := range Root().Kids(true) {
		if lg.GetLevel() == PANIC {
			found = lg
			break
		}
	}
	if found != nil {
		t.Errorf(`Kids() found %v`, found)
	}

	found = nil
	for _, lg := range Root().Kids(true) {
		if lg.GetLevel() == DEBG5 {
			found = lg
			break
		}
	}
	if found == nil {
		t.Error(`Kids(): no DEBG5 kid???`)
	}
	if found != lg5 {t.Errorf(`found != lg5: %v`, found)}
}

func TestSetter(t *testing.T) {
	out := &NamedWriter{
		w: io.Discard,
		name: `DEFAULT`,
	}
	out2 := &StringWriter{
		w: io.Discard,
		name: `Modified`,
	}
	NewC(
		WithOutput(io.Discard),
		WithTopic(`lg`),
		WithLevel(DEBUG),
	)

	lg := L()
	if !lg.IsCurrent() {t.Error(`lg should be current`)}

	lg2 := lg.New(WithTopic(`lg2`))

	NewC(WithTopic(`lg3`))
	lg3 := L()
	lg4 := lg3.New(WithTopic(`lg4`))
	lg5 := lg3.New(WithTopic(`lg5`))
	lg6 := lg5.New(WithTopic(`lg6`))

	// Now we have this tree:
	// Root
	//  \--> lg
	//        \--> lg2
	//        \--> lg3 (current)
	//              \--> lg4
	//              \--> lg5
	//                    \--> lg6

	if !lg3.IsCurrent() {t.Error(`lg3 should be current`)}
	lg5.SetCurrent()
	if !lg5.IsCurrent() {t.Error(`lg5 should be current now`)}

	// Now we have this tree:
	// Root
	//  \--> lg
	//        \--> lg2
	//        \--> lg3
	//              \--> lg4
	//              \--> lg5 (current)
	//                    \--> lg6

	if lg4.GetLevel() != DEBUG {t.Error(`lg4 level should be DEBUG`)}
	if lg5.GetLevel() != DEBUG {t.Error(`lg5 level should be DEBUG`)}
	if lg6.GetLevel() != DEBUG {t.Error(`lg6 level should be DEBUG`)}
	if lg4.GetOutput() != io.Discard {t.Error(`lg4 output does not match`)}
	if lg5.GetOutput() != io.Discard {t.Error(`lg5 output does not match`)}
	if lg6.GetOutput() != io.Discard {t.Error(`lg6 output does not match`)}

	SetLevel(INFO)
	if GetLevel() != INFO {t.Error(`lg5 level should be INFO`)}
	if lg6.GetLevel() != INFO {t.Error(`lg6 level should be INFO`)}
	if lg4.GetLevel() != DEBUG {t.Error(`lg4 level should still be DEBUG`)}

	SetOutput(out2)
	SetMultiLine(true)
	if GetOutput() != out2 {t.Error(`lg5 output should be Modified`)}
	if lg6.GetOutput() != out2 {t.Error(`lg6 output should be Modified`)}
	if lg4.GetOutput() != io.Discard {t.Error(`lg4 must not be modified`)}

	if !GetMultiLine() {t.Error(`lg5 MultiLine`)}
	if !lg6.GetMultiLine() {t.Error(`lg6 MultiLine`)}
	if lg4.GetMultiLine() {t.Error(`lg4 MultiLine`)}

	Close()
	if !lg3.IsCurrent() {t.Error(`lg3 should be current`)}
	SetOutput(out)
	if lg3.GetOutput() != out {t.Error(`lg3 output modified`)}

	lg.SetLevel(PANIC)
	if Root().GetLevel() != WARN {t.Error(`ROOT level should still be WARN`)}
	if lg.GetLevel() != PANIC {t.Error(`lg level should be PANIC`)}
	if lg2.GetLevel() != PANIC {t.Error(`lg2 level should be PANIC`)}
	if lg3.GetLevel() != PANIC {t.Error(`lg3 level should be PANIC`)}
	if lg4.GetLevel() != PANIC {t.Error(`lg4 level should be PANIC`)}
	if lg5.GetLevel() != INFO {t.Error(`lg5 level should still be INFO`)}
	if lg6.GetLevel() != INFO {t.Error(`lg6 level should still be INFO`)}

	Root().Close()
}

// Local Variables:
// tab-width: 4
// End:
