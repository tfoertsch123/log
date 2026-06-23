package log

import (
	"testing"
)

func TestLevelValues(t *testing.T) {
	var i int = -1
	if int(NOTICE) != i {t.Errorf(`NOTICE != %d`, i)}
	i++
	if int(PANIC) != i {t.Errorf(`PANIC != %d`, i)}
	i++
	if int(ERROR) != i {t.Errorf(`ERROR != %d`, i)}
	i++
	if int(WARN) != i {t.Errorf(`WARN != %d`, i)}
	i++
	if int(INFO) != i {t.Errorf(`INFO != %d`, i)}
	i++
	if int(DEBUG) != i {t.Errorf(`DEBUG != %d`, i)}
	i++
	if int(DEBG2) != i {t.Errorf(`DEBG2 != %d`, i)}
	i++
	if int(DEBG3) != i {t.Errorf(`DEBG3 != %d`, i)}
	i++
	if int(DEBG4) != i {t.Errorf(`DEBG4 != %d`, i)}
	i++
	if int(DEBG5) != i {t.Errorf(`DEBG5 != %d`, i)}
	i++
}

func TestToLevel(t *testing.T) {
	var i int = -2
	if ToLevel(i) != PANIC {t.Errorf(`ToLevel(%d) != PANIC`, i)}
	i++
	if ToLevel(i) != PANIC {t.Errorf(`ToLevel(%d) != PANIC`, i)}
	i++
	if ToLevel(i) != PANIC {t.Errorf(`ToLevel(%d) != PANIC`, i)}
	i++
	if ToLevel(i) != ERROR {t.Errorf(`ToLevel(%d) != ERROR`, i)}
	i++
	if ToLevel(i) != WARN {t.Errorf(`ToLevel(%d) != WARN`, i)}
	i++
	if ToLevel(i) != INFO {t.Errorf(`ToLevel(%d) != INFO`, i)}
	i++
	if ToLevel(i) != DEBUG {t.Errorf(`ToLevel(%d) != DEBUG`, i)}
	i++
	if ToLevel(i) != DEBG2 {t.Errorf(`ToLevel(%d) != DEBG2`, i)}
	i++
	if ToLevel(i) != DEBG3 {t.Errorf(`ToLevel(%d) != DEBG2`, i)}
	i++
	if ToLevel(i) != DEBG4 {t.Errorf(`ToLevel(%d) != DEBG3`, i)}
	i++
	if ToLevel(i) != DEBG5 {t.Errorf(`ToLevel(%d) != DEBG5`, i)}
	i++
	if ToLevel(i) != DEBG5 {t.Errorf(`ToLevel(%d) != DEBG5`, i)}
	i++
}

func TestString(t *testing.T) {
	if NOTICE.String() != `NOTICE` {t.Error(`NOTICE.String()`)}
	if PANIC.String() != `PANIC` {t.Error(`PANIC.String()`)}
	if ERROR.String() != `ERROR` {t.Error(`ERROR.String()`)}
	if WARN.String() != `WARN` {t.Error(`WARN.String()`)}
	if INFO.String() != `INFO` {t.Error(`INFO.String()`)}
	if DEBUG.String() != `DEBUG` {t.Error(`DEBUG.String()`)}
	if DEBG2.String() != `DEBG2` {t.Error(`DEBG2.String()`)}
	if DEBG3.String() != `DEBG3` {t.Error(`DEBG3.String()`)}
	if DEBG4.String() != `DEBG4` {t.Error(`DEBG4.String()`)}
	if DEBG5.String() != `DEBG5` {t.Error(`DEBG5.String()`)}
}

func TestParseLevel(t *testing.T) {
	if l, _ := ParseLevel(`NOTICE`); l != NOTICE {t.Error(`ParseLevel(NOTICE)`)}
	if l, _ := ParseLevel(`notice`); l != NOTICE {t.Error(`ParseLevel(notice)`)}
	if l, _ := ParseLevel(`PANIC`); l != PANIC {t.Error(`ParseLevel(PANIC)`)}
	if l, _ := ParseLevel(`panic`); l != PANIC {t.Error(`ParseLevel(panic)`)}
	if l, _ := ParseLevel(`0`); l != PANIC {t.Error(`ParseLevel(0)`)}
	if l, _ := ParseLevel(`ERROR`); l != ERROR {t.Error(`ParseLevel(ERROR)`)}
	if l, _ := ParseLevel(`error`); l != ERROR {t.Error(`ParseLevel(error)`)}
	if l, _ := ParseLevel(`1`); l != ERROR {t.Error(`ParseLevel(1)`)}
	if l, _ := ParseLevel(`WARN`); l != WARN {t.Error(`ParseLevel(WARN)`)}
	if l, _ := ParseLevel(`warn`); l != WARN {t.Error(`ParseLevel(warn)`)}
	if l, _ := ParseLevel(`2`); l != WARN {t.Error(`ParseLevel(2)`)}
	if l, _ := ParseLevel(`INFO`); l != INFO {t.Error(`ParseLevel(INFO)`)}
	if l, _ := ParseLevel(`info`); l != INFO {t.Error(`ParseLevel(info)`)}
	if l, _ := ParseLevel(`3`); l != INFO {t.Error(`ParseLevel(3)`)}
	if l, _ := ParseLevel(`DEBUG`); l != DEBUG {t.Error(`ParseLevel(DEBUG)`)}
	if l, _ := ParseLevel(`debug`); l != DEBUG {t.Error(`ParseLevel(debug)`)}
	if l, _ := ParseLevel(`4`); l != DEBUG {t.Error(`ParseLevel(4)`)}
	if l, _ := ParseLevel(`DEBG2`); l != DEBG2 {t.Error(`ParseLevel(DEBG2)`)}
	if l, _ := ParseLevel(`debg2`); l != DEBG2 {t.Error(`ParseLevel(debg2)`)}
	if l, _ := ParseLevel(`5`); l != DEBG2 {t.Error(`ParseLevel(5)`)}
	if l, _ := ParseLevel(`DEBG3`); l != DEBG3 {t.Error(`ParseLevel(DEBG3)`)}
	if l, _ := ParseLevel(`debg3`); l != DEBG3 {t.Error(`ParseLevel(debg3)`)}
	if l, _ := ParseLevel(`6`); l != DEBG3 {t.Error(`ParseLevel(6)`)}
	if l, _ := ParseLevel(`DEBG4`); l != DEBG4 {t.Error(`ParseLevel(DEBG4)`)}
	if l, _ := ParseLevel(`debg4`); l != DEBG4 {t.Error(`ParseLevel(debg4)`)}
	if l, _ := ParseLevel(`7`); l != DEBG4 {t.Error(`ParseLevel(7)`)}
	if l, _ := ParseLevel(`DEBG5`); l != DEBG5 {t.Error(`ParseLevel(DEBG5)`)}
	if l, _ := ParseLevel(`debg5`); l != DEBG5 {t.Error(`ParseLevel(debg5)`)}
	if l, _ := ParseLevel(`8`); l != DEBG5 {t.Error(`ParseLevel(8)`)}

	if l, e := ParseLevel(`invalid`); l != PANIC && e != ErrInvalidLevel {
		t.Error(`ParseLevel(8)`)
	}
}

// Local Variables:
// tab-width: 4
// End:
