package log

import (
	"os"
	"fmt"
	"syscall"
	"path/filepath"
	// "strings"
	"testing"

	"github.com/tfoertsch123/log/rotate"
)

// ---------------------------------------------------------------------------
// Helper: error capture handler for ParseURL
// ---------------------------------------------------------------------------
type errorCollector struct {
	errs []error
}

func (ec *errorCollector) handler(err error) {
	ec.errs = append(ec.errs, err)
}

func readFile(t *testing.T, f string) string {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func cat(t *testing.T, f string) {
    t.Logf("%q contents:\n%s", f, readFile(t, f))
}

// ---------------------------------------------------------------------------
// TestParseURL – main URL parsing tests
// ---------------------------------------------------------------------------
func TestParseURL_ValidAbsolute(t *testing.T) {
	dir := t.TempDir()
	ec := &errorCollector{}
	opts, err := ParseURL(
		"file://INFO@"+dir+"/app.log?"+
			"maxsize=10MiB&nbackups=3&timeformat=rfc3339&"+
			"multiline=on&mlprefix=&minloc=DEBG2&locdirs=2&topic=test",
		ec.handler,
	)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(ec.errs) > 0 {
		t.Fatal("unexpected error handler calls:", ec.errs)
	}

	var lo LogOpts
	for _, o := range opts {
		o(&lo)
	}
	// Check level
	if lo.level == nil || *lo.level != INFO {
		t.Errorf("expected level INFO, got %v", lo.level)
	}
	// Check timeformat
	if lo.timefmt == nil || *lo.timefmt != "rfc3339" {
		t.Errorf("expected timeformat rfc3339, got %v", lo.timefmt)
	}
	// Check multiline
	if lo.multiln == nil || *lo.multiln != true {
		t.Errorf("expected multiline true, got %v", lo.multiln)
	}
	// Check multilineprefix
	if lo.mlprfx == nil || *lo.mlprfx != "" || !lo.setprfx {
		t.Errorf("expected empty mlprfx, got (%v, %v)", lo.mlprfx, lo.setprfx)
	}
	// Check minloc
	if lo.minloc == nil || *lo.minloc != DEBG2 {
		t.Errorf("expected minloc DEBG2, got %v", lo.minloc)
	}
	// Check locdirs
	if lo.nlocdir == nil || *lo.nlocdir != 2 {
		t.Errorf("expected locdirs 2, got %v", lo.nlocdir)
	}
	// Check topic
	if lo.topic == nil || *lo.topic != " [test]" {
		t.Errorf("expected topic ' [test]', got %v", *lo.topic)
	}
	// Check that output is via factory (not file)
	if lo.out != nil {
		t.Error("expected out to be nil (factory used)")
	}
	if lo.outcr == nil {
		t.Fatal("expected output factory to be set")
	}
	// Verify factory creates a rotator
	writer, err := lo.outcr()
	if err != nil {
		t.Fatal("factory error:", err)
	}
	// The writer should be a *rotate.Rotate
	rot, ok := writer.(*rotate.Rotate)
	if !ok {
		t.Errorf("expected *rotate.Rotate, got %T", writer)
	}
	defer rot.Close()

	if rot.Name() != dir+"/app.log" {
		t.Errorf("expected name: %q, got: %q", dir+"/app.log", rot.Name())
	}

	if rot.MaxSize() != 10*1024*1024 {
		t.Errorf("expected MaxSize: %q, got: %q", 10*1024*1024, rot.MaxSize())
	}

	if rot.NBackups() != 3 {
		t.Errorf("expected NBackups: %q, got: %q", 3, rot.NBackups())
	}
}

func TestParseURL_RelativeHostDot(t *testing.T) {
	dir := t.TempDir()
	if d, err := os.Getwd(); err == nil {t.Logf("WD is %q", d)}
	t.Chdir(dir)
	if d, err := os.Getwd(); err == nil {t.Logf("changed WD to %q", d)}
	if d, err := os.Getwd(); err != nil || d != dir {
		t.Fatalf("unexpected error: %v or expected: %q != got: %q", err, dir, d)
	}
	relPath := "relative.log"
	urlStr := "file://DEBUG@./" + relPath + "?maxsize=100"
	ec := &errorCollector{}
	opts, err := ParseURL(urlStr, ec.handler)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	lg := L().New(opts...)
	defer lg.Close()
	
	// Write something to trigger file creation
	lg.Info("info message")

	if _, err := os.Stat(relPath); os.IsNotExist(err) {
		t.Errorf("expected file %q to exist", relPath)
	}

	cat(t, relPath)

	mx := lg.GetOutput().(*rotate.Rotate).MaxSize()
	if mx != 100 {
		t.Errorf("expected MaxSize: %q, got: %q", 100, mx)
	}
}

func TestParseURL_RelativeEmptyHost(t *testing.T) {
	dir := t.TempDir()
	if d, err := os.Getwd(); err == nil {t.Logf("WD is %q", d)}
	t.Chdir(dir)
	if d, err := os.Getwd(); err == nil {t.Logf("changed WD to %q", d)}
	if d, err := os.Getwd(); err != nil || d != dir {
		t.Fatalf("unexpected error: %v or expected: %q != got: %q", err, dir, d)
	}
	relPath := "relative.log"
	urlStr := "file://DEBUG@/./" + relPath
	ec := &errorCollector{}
	opts, err := ParseURL(urlStr, ec.handler)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	lg := L().New(opts...)
	defer lg.Close()
	
	// Write something to trigger file creation
	lg.Info("info message")

	if _, err := os.Stat(relPath); os.IsNotExist(err) {
		t.Errorf("expected file %q to exist", relPath)
	}

	cat(t, relPath)
}

func TestParseURL_Stdout(t *testing.T) {
	urlStr := "//stdout?level=debug&multiline=false"
	opts, err := ParseURL(urlStr, nil)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	lg := L().New(opts...)
	defer lg.Close()

	if lg.GetOutput() != os.Stdout {
		t.Error("Output is not Stdout")
	}

	if lg.GetLevel() != DEBUG {
		t.Errorf("unexpected log level %v", lg.GetLevel())
	}
}

func TestParseURL_Stderr(t *testing.T) {
	urlStr := "file://DEBUG@stderr"
	opts, err := ParseURL(urlStr, nil)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	lg := L().New(opts...)
	defer lg.Close()

	if lg.GetOutput() != os.Stderr {
		t.Error("Output is not Stderr")
	}

	if lg.GetLevel() != DEBUG {
		t.Errorf("unexpected log level %v", lg.GetLevel())
	}
}

func TestParseURL_FDopen(t *testing.T) {
	fd, err := syscall.Dup(2)
	if err != nil {
		t.Fatal("unexpected error in syscall.Dup():", err)
	}

	// ParseURL turns the fd into an *os.File. That thing then gets
	// a cleanup attached that closes the file when GC is running.
	// We don't need to close it here.
	// defer syscall.Close(fd)

	urlStr := fmt.Sprintf("file://DEBUG@%d", fd)
	t.Logf("Using URL %q", urlStr)
	opts, err := ParseURL(urlStr, nil)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	lg := L().New(opts...)
	defer lg.Close()

	if file, ok := lg.GetOutput().(*os.File); ok {
		if file.Fd() != uintptr(fd) {
			t.Errorf("expecting fd %q, got %q", fd, file.Fd())
		}
	} else {
		t.Errorf("Output is not *os.File")
	}

	if lg.GetLevel() != DEBUG {
		t.Errorf("unexpected log level %v", lg.GetLevel())
	}
}

func TestParseURL_FDclosed(t *testing.T) {
	fd, err := syscall.Dup(2)
	if err != nil {
		t.Fatal("unexpected error in syscall.Dup():", err)
	}
	syscall.Close(fd)

	urlStr := fmt.Sprintf("file://DEBUG@%d", fd)
	t.Logf("Using URL %q", urlStr)
	_, err = ParseURL(urlStr, nil)
	if err == nil {
		t.Fatal("unexpected success")
	}
}

func TestParseURL_AppendOn(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "app.log")

	if f, err := os.Create(fn); err != nil {
		t.Fatal(err)
	} else {
		f.Write([]byte("to be kept"))
		f.Close()
	}

	ec := &errorCollector{}
	opts, err := ParseURL(dir+"/app.log?append=on&maxsize=10MiB", ec.handler)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(ec.errs) > 0 {
		t.Fatal("unexpected error handler calls:", ec.errs)
	}

	var lo LogOpts
	for _, o := range opts {
		o(&lo)
	}

	if lo.outcr == nil {
		t.Fatal("expected output factory to be set")
	}

	// Verify factory creates a rotator
	writer, err := lo.outcr()
	if err != nil {
		t.Fatal("factory error:", err)
	}

	// The writer should be a *rotate.Rotate
	rot, ok := writer.(*rotate.Rotate)
	if !ok {
		t.Errorf("expected *rotate.Rotate, got %T", writer)
	}
	defer rot.Close()

	// Write something
	n, err := rot.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5", n)
	}

	// Verify content
	if got := readFile(t, fn); got != "to be kepthello" {
		t.Errorf("content = %q, want %q", got, "to be kepthello")
	}
}

func TestParseURL_AppendOff(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "app.log")

	if f, err := os.Create(fn); err != nil {
		t.Fatal(err)
	} else {
		f.Write([]byte("to be kept"))
		f.Close()
	}

	ec := &errorCollector{}
	opts, err := ParseURL(dir+"/app.log?append=off&maxsize=10MiB", ec.handler)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(ec.errs) > 0 {
		t.Fatal("unexpected error handler calls:", ec.errs)
	}

	var lo LogOpts
	for _, o := range opts {
		o(&lo)
	}

	if lo.outcr == nil {
		t.Fatal("expected output factory to be set")
	}

	// Verify factory creates a rotator
	writer, err := lo.outcr()
	if err != nil {
		t.Fatal("factory error:", err)
	}

	// The writer should be a *rotate.Rotate
	rot, ok := writer.(*rotate.Rotate)
	if !ok {
		t.Errorf("expected *rotate.Rotate, got %T", writer)
	}
	defer rot.Close()

	// Write something
	n, err := rot.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5", n)
	}

	// Verify content
	if got := readFile(t, fn); got != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
}

func TestParseURL_ErrorHandler(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test as root")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if d, err := os.Getwd(); err != nil || d != dir {
		t.Fatalf("unexpected error: %v or expected: %q != got: %q", err, dir, d)
	}
	relPath := "relative.log"

	// We use maxsize=100. That means the rotator actually rotates.
	// rotate.New() calls the first rotation. The first step then is to
	// open the dot-file. We create a dot-file here as a read-only file
	// So, that opening will fail.
	f, err := os.Create("."+relPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	os.Chmod("."+relPath, 0444) // read-only

	urlStr := "file://DEBUG@./" + relPath + "?maxsize=100"
	ec := &errorCollector{}
	opts, err := ParseURL(urlStr, ec.handler)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	lg := L().New(opts...)
	if lg != nil {
		t.Error("unexpected success")
	}

	if len(ec.errs) == 0 {
		t.Fatal("no error")
	}
	t.Logf("got errors: %v", ec.errs)
}

func TestParseURL_Invalid(t *testing.T) {
	// most errors are ErrInvalidFormat
	for _, s := range []string{
		"http://DEBUG@./path/to/file.log?maxsize=100",
		"file://DEBUG@././path/to/file.log?maxsize=100",
		"file://NOTICE@./path/to/file.log?maxsize=100",
		"file://DEBUG@.?maxsize=100",
		"file://DEBUG@stderr/path",
		"file://DEBUG@stdout/path",
		"file://DEBUG@stdout?maxsize=10",
		"file://DEBUG@12/path",
		"file://DEBUG@something",
		"file://DEBUG@stdout?level=INFO",
		"file://DEBUG@stderr?maxsize=1&maxsize=1",
		"//stdout?locdirs=fritz",
		"//stdout?maxsize=fritz",
		"//stdout?nbackups=fritz",
		"//stdout?nbackups=10&_non_param_=fritz",
	} {
		_, err := ParseURL(s, nil)
		if err != ErrInvalidFormat {
			t.Fatalf("%q: unexpected error: %v", s, err)
		}
	}

	// generic errors
	for _, s := range []string{
		"file://DEBUG@.:bsfgb/path/to/file.log?maxsize=100",
		"file://FRITZ@./path/to/file.log?maxsize=100",
		"file://DEBUG@stdout?level=FRITZ",
		"file://stderr?level=fritz",
		"//stdout?minloc=fritz",
	} {
		_, err := ParseURL(s, nil)
		if err == nil {
			t.Fatalf("%q: unexpected success", s)
		}
	}
}

// Local Variables:
// tab-width: 4
// End:
