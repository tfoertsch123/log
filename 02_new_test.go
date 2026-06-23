package log

import (
	"testing"
	"os"
	"strings"
)

func TestDefaults(t *testing.T) {
	if ! L().IsRoot() {t.Error(`L() and Root() should match`)}
	if L().level != WARN {t.Errorf(`level exp: WARN got %v`, L().level)}
	if L().out != os.Stderr {t.Errorf(`out exp: Stderr got %v`, L().out)}
	if L().topic != "" {t.Errorf(`topic exp: "" got %v`, L().topic)}
	if L().prev != nil {t.Errorf(`prev exp: nil got %v`, L().prev)}
	if L().derived != nil {t.Errorf(`derived exp: nil got %v`, L().derived)}
	if L().multiln {t.Errorf(`multiln exp: false got %v`, L().multiln)}
}

func TestNew(t *testing.T) {
	var out strings.Builder
	lg := L().New(
		WithOutput(&out),
		WithTopic(`lg`),
		WithLevel(DEBUG),
		WithMultiLine(true),
	)
	if ! L().IsRoot() {t.Error(`L() and Root() should still match`)}
	if lg.IsCurrent() {t.Error(`L() and new logger should not match`)}

	if lg.level != DEBUG {t.Errorf(`level exp: DEBUG got %v`, lg.level)}
	if lg.out != &out {t.Errorf(`out unexpected got %v`, lg.out)}
	if lg.topic != " [lg]" {t.Errorf(`topic exp: " [lg]" got %v`, lg.topic)}
	if lg.prev != Root() {t.Errorf(`prev exp: %v got %v`, L(), lg.prev)}
	if lg.derived != nil {t.Errorf(`derived exp: nil got %v`, lg.derived)}
	if !lg.multiln {t.Errorf(`multiln exp: true got %v`, lg.multiln)}
	if L().derived == nil {t.Errorf(`L().derived should not be nil now`)}
	if len(L().derived) != 1 {t.Errorf(`len(L().derived) should be 1`)}

	lg2 := lg.New(WithLevel(INFO), WithTopic(`lg2`))
	if lg2.level != INFO {t.Errorf(`level exp: INFO got %v`, lg2.level)}
	if lg2.out != &out {t.Errorf(`out unexpected got %v`, lg2.out)}
	if lg2.topic != " [lg2]" {t.Errorf(`topic exp: " [lg2]" got %v`, lg2.topic)}
	if lg2.prev != lg {t.Errorf(`prev exp: %v got %v`, lg, lg2.prev)}
	if lg2.derived != nil {t.Errorf(`derived exp: nil got %v`, lg2.derived)}
	if !lg2.multiln {t.Errorf(`multiln exp: true got %v`, lg2.multiln)}
	if lg.derived == nil {t.Errorf(`lg.derived should not be nil now`)}
	if len(lg.derived) != 1 {t.Errorf(`len(lg.derived) should be 1`)}

	lg3 := lg.New(WithTopic(`lg3`))
	lg4 := lg3.New(WithTopic(`lg4`))
	lg5 := lg3.New(WithTopic(`lg5`))
	lg6 := lg5.New(WithTopic(`lg6`))

	// Now we have this tree:
	// Root
	//  \--> lg
	//        \--> lg2
	//        \--> lg3
	//              \--> lg4
	//              \--> lg5
	//                    \--> lg6

	if lg5.Close() != lg3 {t.Error(`lg5.Close() should return lg3`)}
	if !lg5.IsClosed() {t.Errorf(`lg5 not closed %v`, lg5)}
	if !lg6.IsClosed() {t.Errorf(`lg6 not closed %v`, lg6)}

	found := map[*Logger]struct{}{
		lg: struct{}{},
		lg2: struct{}{},
		lg3: struct{}{},
		lg4: struct{}{},
	}

	for _, lg := range Root().Kids(true) {
		_, ok := found[lg]
		if !ok {t.Errorf(`logger (%p: %v) not found in Kids()`, lg, lg)}
		delete(found, lg)
	}
	if len(found) != 0 {t.Errorf(`some loggers were not traversed`)}

	if Root().Close() != Root() {
		t.Errorf(`Root().Close() should return Root()`)
	}
	if ! L().IsRoot() {t.Errorf(`L(%v) should be root(%v)`, L(), Root())}
	if len(L().derived) != 0 {t.Errorf(`len(L().derived) should be 0`)}

	// check the close marker for all derived loggers
	if !lg5.IsClosed() {t.Errorf(`lg5 is not closed`)}
	if !lg4.IsClosed() {t.Errorf(`lg4 is not closed`)}
	if !lg3.IsClosed() {t.Errorf(`lg3 is not closed`)}
	if !lg2.IsClosed() {t.Errorf(`lg2 is not closed`)}
	if !lg.IsClosed() {t.Errorf(`lg is not closed`)}
	if Root().IsClosed() {t.Errorf(`Root() must not be closed`)}
}

func TestPackageNew(t *testing.T) {
	var out strings.Builder
	NewC(
		WithOutput(&out),
		WithTopic(`lg`),
		WithLevel(DEBUG),
	)
	if L().IsRoot() {t.Error(`L() and Root() should NOT match`)}

	lg := L()
	lg2 := lg.New(WithTopic(`lg2`))

	NewC(WithTopic(`lg3`))
	lg3 := L()
	lg4 := lg3.New(WithTopic(`lg4`))
	NewC(WithTopic(`lg5`))
	lg5 := L()
	NewC(WithTopic(`lg6`))
	lg6 := L()

	// Now we have this tree:
	// Root
	//  \--> lg
	//        \--> lg2
	//        \--> lg3
	//              \--> lg4
	//              \--> lg5
	//                    \--> lg6 (current)

	if lg5.Close() != lg3 {t.Error(`lg5.Close() should return lg3`)}
	if !lg5.IsClosed() {t.Errorf(`lg5 not closed %v`, lg5)}
	if !lg6.IsClosed() {t.Errorf(`lg6 not closed %v`, lg6)}
	if !lg3.IsCurrent() {t.Error(`lg3 should be current`)}

	found := map[*Logger]struct{}{
		lg: struct{}{},
		lg2: struct{}{},
		lg3: struct{}{},
		lg4: struct{}{},
	}

	for _, lg := range Root().Kids(true) {
		_, ok := found[lg]
		if !ok {t.Errorf(`logger (%p: %v) not found in Kids()`, lg, lg)}
		delete(found, lg)
	}
	if len(found) != 0 {t.Errorf(`some loggers were not traversed`)}

	if Root().Close() != Root() {
		t.Errorf(`Root().Close() should return Root()`)
	}
	if ! L().IsRoot() {t.Errorf(`L(%v) should be root(%v)`, L(), Root())}
	if len(L().derived) != 0 {t.Errorf(`len(L().derived) should be 0`)}

	// check the close marker for all derived loggers
	if !lg5.IsClosed() {t.Errorf(`lg5 is not closed`)}
	if !lg4.IsClosed() {t.Errorf(`lg4 is not closed`)}
	if !lg3.IsClosed() {t.Errorf(`lg3 is not closed`)}
	if !lg2.IsClosed() {t.Errorf(`lg2 is not closed`)}
	if !lg.IsClosed() {t.Errorf(`lg is not closed`)}
	if Root().IsClosed() {t.Errorf(`Root() must not be closed`)}
}

func TestPackageNew2(t *testing.T) {
	var out strings.Builder
	NewC(
		WithOutput(&out),
		WithTopic(`lg`),
		WithLevel(DEBUG),
	)
	if L().IsRoot() {t.Error(`L() and Root() should NOT match`)}

	lg := L()
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

	if ! lg3.IsCurrent() {t.Errorf(`lg3 should be current at this point`)}

	found := map[*Logger]struct{}{
		lg2: struct{}{},
		lg3: struct{}{},
	}
	
	for _, lg := range lg.Kids(false) {
		_, ok := found[lg]
		if !ok {t.Errorf(`logger (%p: %v) not found in Kids()`, lg, lg)}
		delete(found, lg)
	}
	if len(found) != 0 {t.Errorf(`some loggers were not traversed`)}

	found = map[*Logger]struct{}{
		lg: struct{}{},
		lg2: struct{}{},
		lg3: struct{}{},
		lg4: struct{}{},
		lg5: struct{}{},
		lg6: struct{}{},
	}
	
	for _, lg := range Root().Kids(true) {
		_, ok := found[lg]
		if !ok {t.Errorf(`logger (%p: %v) not found in Kids()`, lg, lg)}
		delete(found, lg)
	}
	if len(found) != 0 {t.Errorf(`some loggers were not traversed`)}

	Close()
	if ! lg.IsCurrent() {t.Errorf(`Close() should make lg current`)}

	// check the close marker for all derived loggers
	if !lg6.IsClosed() {t.Errorf(`lg6 is not closed`)}
	if !lg5.IsClosed() {t.Errorf(`lg5 is not closed`)}
	if !lg4.IsClosed() {t.Errorf(`lg4 is not closed`)}
	if !lg3.IsClosed() {t.Errorf(`lg3 is not closed`)}
	if lg2.IsClosed() {t.Errorf(`lg2 must not be closed`)}
	if lg.IsClosed() {t.Errorf(`lg must not be closed`)}
	if Root().IsClosed() {t.Errorf(`Root() must not be closed`)}
}

func TestPackageNew3(t *testing.T) {
	root := Root()
	NewC()
	cur := L()
	if cur.IsRoot() {t.Error(`L() and Root() should NOT match`)}
	curKid := NewK()
	rootKid := NewR()
	if curKid.prev != cur {t.Error(`curKid should be a kid of L()`)}
	if rootKid.prev != root {t.Error(`rootKid should be a kid of Root()`)}

	if root.IsClosed() {t.Error(`root should not be closed`)}
	if cur.IsClosed() {t.Error(`cur should not be closed`)}
	if rootKid.IsClosed() {t.Error(`rootKid should not be closed`)}
	if curKid.IsClosed() {t.Error(`curKid should not be closed`)}

	Root().Close()

	if root.IsClosed() {t.Error(`root should not be closed`)}
	if !cur.IsClosed() {t.Error(`cur should be closed`)}
	if !rootKid.IsClosed() {t.Error(`rootKid should be closed`)}
	if !curKid.IsClosed() {t.Error(`curKid should be closed`)}
}

// Local Variables:
// tab-width: 4
// End:
