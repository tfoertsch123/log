package rotate

import (
	"fmt"
	"sync"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// helper: create temporary directory for each test
func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// helper: read file contents
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// helper: check file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestNewNoRotationNoAppend(t *testing.T) {
	dir := setup(t)
	fn := filepath.Join(dir, "test.log")

	if f, err := os.Create(fn); err != nil {
		t.Fatal(err)
	} else {
		f.Write([]byte("to be overwritten"))
		f.Close()
	}

	r, err := New(WithFileName(fn))
	if err != nil {
		t.Fatal(err)
	}
	defer r.File().Close()

	// Check file created
	if !fileExists(fn) {
		t.Error("file was not created")
	}
	if r.Name() != fn {
		t.Errorf("Name() = %q, want %q", r.Name(), fn)
	}

	// Write something
	n, err := r.Write([]byte("hello"))
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

	if r.Rotations() != -1 {
		t.Errorf("number of rotations should be -1")
	}
}

func TestNewRotationWithAppend(t *testing.T) {
	dir := setup(t)
	fn := filepath.Join(dir, "test.log")

	if f, err := os.Create(fn); err != nil {
		t.Fatal(err)
	} else {
		f.Write([]byte("to be kept"))
		f.Close()
	}

	r, err := New(WithFileName(fn), WithMaxSize(10), WithInitialAppend(true))
	if err != nil {
		t.Fatal(err)
	}
	defer r.File().Close()

	// Check file created
	if !fileExists(fn) {
		t.Error("file was not created")
	}
	if r.Name() != fn {
		t.Errorf("Name() = %q, want %q", r.Name(), fn)
	}

	if r.Rotations() != 0 {
		t.Errorf("number of rotations should be 0")
	}

	// Write something
	n, err := r.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5", n)
	}

	// After rotation the main file should contain the overflow ("abc")
	// but we need to give the async rotate a moment to complete
	time.Sleep(100 * time.Millisecond)

	// Verify content
	if got := readFile(t, fn + `~`); got != "to be kepthello" {
		t.Errorf("content = %q, want %q", got, "to be kepthello")
	}

	if got := readFile(t, fn); got != "" {
		t.Errorf("rotated file should be empty, got %q", got)
	}

	if r.Rotations() != 1 {
		t.Errorf("number of rotations should be 1")
	}
}

func TestNewWithRotationAndSizeTrigger(t *testing.T) {
	dir := setup(t)
	fn := filepath.Join(dir, "test.log")
	r, err := New(
		WithFileName(fn),
		WithMaxSize(10),       // rotate after 10 bytes
		WithNBackups(2),       // keep 2 backups
	)
	if err != nil {
		t.Fatal(err)
	}
	defer r.File().Close()

	if r.Rotations() != 1 {
		t.Errorf("number of rotations should be 1")
	}

	// Write 8 bytes – no rotation yet
	_, err = r.Write([]byte("12345678"))
	if err != nil {
		t.Fatal(err)
	}

	// Write 3 more bytes – total 11, should trigger rotation
	_, err = r.Write([]byte("abc"))
	if err != nil {
		t.Fatal(err)
	}

	// After rotation the main file should contain the overflow ("abc")
	// but we need to give the async rotate a moment to complete
	time.Sleep(100 * time.Millisecond)

	mainContent := readFile(t, fn)
	if mainContent != "" {
		t.Errorf("main file contains %q, want %q", mainContent, "")
	}

	// Backup file "test.log~" should contain previous content
	backup1 := fn + "~"
	if !fileExists(backup1) {
		t.Fatal("backup file not created")
	}
	backupContent := readFile(t, backup1)
	if backupContent != "12345678abc" {
		t.Errorf("backup content = %q, want %q", backupContent, "12345678abc")
	}

	if r.Rotations() != 2 {
		t.Errorf("number of rotations should be 2")
	}

	// No second backup because maxVersions=2 (original + 1 backup)
	backup2 := fn + "~1"
	if fileExists(backup2) {
		t.Error("unexpected backup file exists")
	}
}

func TestMultipleRotationsAndNaming(t *testing.T) {
	dir := setup(t)
	fn := filepath.Join(dir, "test.log")
	r, err := New(
		WithFileName(fn),
		WithMaxSize(4),
		WithNBackups(3),  // keep 3 backups: ~, ~01, ~02
	)
	if err != nil {
		t.Fatal(err)
	}
	defer r.File().Close()

	// Write data to trigger three rotations
	data := []string{"AAAA", "BBBB", "CCCC", "DDDD"}
	for _, s := range data {
		_, err = r.Write([]byte(s))
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// After all writes, naming should be:
	// Main: empty (freshly rotated)
	// ~: "DDDD" (second last)
	// ~01: "CCCC"
	// ~02: "BBBB" (oldest, but might be overwritten due to maxVersions=3)
	if got := readFile(t, fn); got != "" {
		t.Errorf("main file = %q, want %q", got, "")
	}
	if got := readFile(t, fn+"~"); got != "DDDD" {
		t.Errorf("backup ~ = %q, want %q", got, "DDDD")
	}
	if got := readFile(t, fn+"~1"); got != "CCCC" {
		t.Errorf("backup ~01 = %q, want %q", got, "CCCC")
	}
	if got := readFile(t, fn+"~2"); got != "BBBB" {
		t.Errorf("backup ~02 = %q, want %q", got, "BBBB")
	}
	// No third backup because maxVersions=3 (original + 2 backups)
	if fileExists(fn + "~3") {
		t.Error("unexpected backup ~3")
	}

	// 5 rotations so far:
	// 1 - initial creation of the file
	// 2 - after AAAA
	// 3 - after BBBB
	// 4 - after CCCC
	// 5 - after DDDD
	if r.Rotations() != 5 {
		t.Errorf("number of rotations should be 5")
	}
}

func TestManualRotate(t *testing.T) {
	dir := setup(t)
	fn := filepath.Join(dir, "test.log")
	r, err := New(
		WithFileName(fn),
		WithMaxSize(1000),
		WithNBackups(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer r.File().Close()

	// Write some data
	_, err = r.Write([]byte("initial"))
	if err != nil {
		t.Fatal(err)
	}

	// Manual rotation
	r.Rotate()
	time.Sleep(100 * time.Millisecond)

	// After rotation, main file should be empty (truncated)
	mainContent := readFile(t, fn)
	if mainContent != "" {
		t.Errorf("main file after rotate = %q, want empty", mainContent)
	}

	// Backup should contain "initial"
	backup := fn + "~"
	if fileExists(backup) {
		if got := readFile(t, backup); got != "initial" {
			t.Errorf("backup = %q, want %q", got, "initial")
		}
	} else {
		t.Fatal("backup file not created after manual rotate")
	}
}

func TestErrorHandlingBadPath(t *testing.T) {
	// Try to create a file in a non-existing directory
	badDir := filepath.Join(os.TempDir(), "nonexistent_dir_xyz")
	_, err := New(WithFileName(filepath.Join(badDir, "test.log")))
	if err == nil {
		t.Fatal("expected error for bad directory, got nil")
	}
}

func TestErrorHandlingNoPermission(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test as root")
	}
	dir := setup(t)
	// Create a read-only file
	roFile := filepath.Join(dir, "readonly.log")
	f, err := os.Create(roFile)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	os.Chmod(roFile, 0444) // read-only

	// Try to open with Write access – should fail
	_, err = New(WithFileName(roFile))
	if err == nil {
		t.Fatal("expected permission error, got nil")
	}
}

func TestErrorHandlingWithRotateNoPermission(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test as root")
	}
	dir := setup(t)
	// Create a read-only file
	roFile := filepath.Join(dir, "readonly.log")
	f, err := os.Create(roFile)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	os.Chmod(roFile, 0444) // read-only

	hndCalled := false

	// The difference to the previous case (TestErrorHandlingNoPermission)
	// is that due to MaxSize we get a Rotate object that actually rotates.
	// So, as long as the directory is writable renaming the old file
	// should succeed.
	var r *Rotate
	r, err = New(
		WithFileName(roFile),
		WithMaxSize(10),
		WithErrorHandler(func (r_ *Rotate, err_ error) {
			hndCalled = true
			if r_ != r {
				t.Errorf("error handler called but the object is not the same")
			}
			if err_ == nil {
				t.Errorf("expecting permission denied but got nil")
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Now let's make the directory read-only
	os.Chmod(dir, 0555) // read-only
	defer os.Chmod(dir, 0755)

	r.Rotate()
	time.Sleep(100 * time.Millisecond)
	if !hndCalled {
		t.Error("error handler not called")
	}
}

func TestErrorRenameFails(t *testing.T) {
 	dir := setup(t)
	nm := filepath.Join(dir, "readonly.log")

	hndCalled := false

	var r *Rotate
	r, err := New(
		WithFileName(nm),
		WithNBackups(1),
		WithMaxSize(10),
		WithErrorHandler(func (r_ *Rotate, err_ error) {
			hndCalled = true
			if r_ != r {
				t.Errorf("error handler called but the object is not the same")
			}
			if err_ == nil {
				t.Errorf("expecting EEXIST but got nil")
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// This will cause EEXIST in doRotate()
	err = os.Mkdir(nm + `~`, 0500)
	if err != nil {
		t.Fatal(err)
	}

	r.Rotate()
	time.Sleep(100 * time.Millisecond)
	if !hndCalled {
		t.Error("error handler not called")
	}
}

func TestFirstRotateFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test as root")
	}
	dir := setup(t)
	// Create file by the same name as our temp file (note the dot)
	file := filepath.Join(dir, "readonly.log")
	tmpfile := filepath.Join(dir, ".readonly.log")
	f, err := os.Create(tmpfile)
	f.Close()
	
	if err != nil {
		t.Fatal(err)
	}

	_, err = New(
		WithFileName(file),
		WithMaxSize(10),
	)
	if err == nil {
		t.Error("Expecting EEXIST but got success")
	}
}

func TestErrorFullFile(t *testing.T) {
	// Check if /dev/full exists; skip otherwise
	if _, err := os.Stat("/dev/full"); os.IsNotExist(err) {
		t.Skip("/dev/full not available on this system")
	}

	dir := setup(t)

	// Create a symlink to /dev/full in a temporary directory
	fullFile := filepath.Join(dir, "readonly.log")
	if err := os.Symlink("/dev/full", fullFile); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Try to open with Write access – should fail
	r, err := New(WithFileName(fullFile))
	if err != nil {
		t.Fatal(err)
	}
	defer r.File().Close()

	// Write something
	_, err = r.Write([]byte("hello"))
	if err == nil {
		t.Fatal("expected bad file descriptor (read-only), got nil")
	}
}

func TestConcurrentWritesWithRotation(t *testing.T) {
	dir := setup(t)
	fn := filepath.Join(dir, "concurrent.log")
	r, err := New(
		WithFileName(fn),
		WithMaxSize(1000),		// we will write 3900 bytes
		WithNBackups(3),		// that should generate main, ~, ~1 and ~2
	)
	if err != nil {
		t.Fatal(err)
	}
	defer r.File().Close()

	// Launch several goroutines writing concurrently
	var wg sync.WaitGroup
	c := make(chan int, 0)
	done := make(chan struct{}, 0)
	total := 0
	go func() {
		for n := range c {total+=n}
		close(done)
	}()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			written := 0
			for j := 0; j < 20; j++ {
				msg := fmt.Sprintf("goroutine %d line %d\n", id, j)
				written += len(msg)
				_, err := r.Write([]byte(msg))
				if err != nil {
					t.Logf("write error: %v", err)
				}
				time.Sleep(time.Millisecond) // small delay to increase race chances
			}
			c <- written
		}(i)
	}
	wg.Wait()
	close(c)
	<-done

	if !fileExists(fn) {
		t.Error("main file missing after concurrent writes")
	}

	main := readFile(t, fn)
	first := readFile(t, fn + "~")
	second := readFile(t, fn + "~1")
	third := readFile(t, fn + "~2")
	if len(main)+len(first)+len(second)+len(third) != total {
		t.Errorf("content length: exp %d got %d",
			total,
			len(main)+len(first)+len(second)+len(third))
	}
}

// Local Variables:
// tab-width: 4
// End:
