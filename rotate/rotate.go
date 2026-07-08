/*
   Copyright 2026 Torsten Foertsch

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
 */

// Package rotate implements a size-based log file rotator. It writes to a file
// and automatically rotates (renames) it when the total written size exceeds
// a configurable threshold. Backups are kept with a versioning scheme:
//   - The main file:       <filename>
//   - First backup:        <filename>~
//   - Second backup:       <filename>~01
//   - Third backup:        <filename>~02
//   - etc.
//
// Usage:
//   r, err := rotate.New(
//       rotate.WithFileName("/var/log/app.log"),
//       rotate.WithMaxSize(10*1024*1024),  // 10 MB
//       rotate.WithNBackups(5),
//       rotate.WithErrorHandler(func(r *rotate.Rotate, err error) {
//           log.Printf("rotation error: %v", err)
//       }),
//   )
//   if err != nil { ... }
//   defer r.Close()
//   // write to r like any io.Writer
//
// Thread safety:
//   All exported methods are safe for concurrent use, except that File()
//   returns a raw *os.File whose lifetime is not protected after the call.
//   Use Name() instead if you need to open the file independently, or
//   acquire external synchronization if you must hold the File handle.
package rotate

import (
	"sync"
	"os"
	"io"
	"io/fs"
	"fmt"
	"errors"
	"path/filepath"
)

// ErrorHnd is a callback invoked when an automatic rotation fails. It receives
// the Rotate instance and the error that occurred. If nil, rotation errors
// are silently ignored.
type ErrorHnd func(*Rotate, error)()

// Rotate is a file writer that automatically rotates the log file when its
// size reaches a configurable maximum. It implements io.Writer and io.Closer.
type Rotate struct {
	wmu         sync.Mutex		// protects write
	rmu         sync.Mutex		// protects rotation
	rot         int64			// number of rotations so far
	                            // or -1 if not rotating
	maxSize     int64
	maxVersions uint
	errorHnd    ErrorHnd
	fh          *os.File
	cur_sz      int64
	dir         string
	fn          string
}

type descr struct {
	fn  string
	ms  int64
	rot int64
	mv  uint
	hnd ErrorHnd
	append bool
}

// Opt is a function type used to pass information to New().
type Opt func(*descr)

// WithFileName sets the path of the file to write to. Required.
func WithFileName(f string) Opt {return func(x *descr) {x.fn = f}}

// WithMaxSize sets the file size in bytes that triggers a rotation.
// When maxSize > 0, rotation is enabled. If maxSize == 0 (default),
// no rotation occurs and the file is truncated on open (unless
// WithInitialAppend(true) is used).
func WithMaxSize(f int64) Opt {return func(x *descr) {x.ms, x.rot = f, 0}}

// WithNBackups sets the number of backup copies to keep.
// The default is 1. If maxSize > 0, backups are rotated through
// the naming scheme described in the package documentation.
func WithNBackups(f uint) Opt {return func(x *descr) {x.mv = f}}

// WithErrorHandler sets a callback for rotation errors. If nil (default),
// rotation errors are ignored.
func WithErrorHandler(f ErrorHnd) Opt {return func(x *descr) {x.hnd = f}}

// WithInitialAppend controls whether the file is appended to or truncated
// when opened. If true, existing content is preserved and new data is
// appended. If false (default), the file is truncated.
func WithInitialAppend(f bool) Opt {return func(x *descr) {x.append = f}}

// New creates a Rotate writer. Options must include WithFileName.
// Other options are optional.
//
// If maxSize is zero (or not set), the file is opened (or created) once
// and never rotated. In that case, WithInitialAppend controls truncation.
//
// If maxSize > 0, rotation is enabled. The file is initially truncated
// (unless WithInitialAppend(true) is used). On the first open, a rotation
// is performed to ensure any old backup chain starts cleanly.
//
// Returns an error if the file cannot be opened or if the initial rotation
// fails (rotation enabled case).
func New(_opts ...Opt) (*Rotate, error) {
	d := &descr{mv: 1, rot: -1}
	for _, o := range _opts {o(d)}

	r := &Rotate{
		maxSize:     d.ms,
		maxVersions: d.mv,
		errorHnd:    d.hnd,
		rot:         d.rot,
	}
	r.dir, r.fn = filepath.Split(d.fn)

	if d.append || r.maxSize <= 0 {
		// no O_EXCL - we are good if the file already exists
		// no O_TRUNC - if it exists we just append (only if d.append)
		flags := os.O_CREATE|os.O_WRONLY|os.O_APPEND
		if !d.append {flags |= os.O_TRUNC}
		f, err := os.OpenFile(d.fn, flags, 0600)
		if err != nil {return nil, err}
		r.fh = f
		// lseek cannot return an error here
		r.cur_sz, _ = f.Seek(0, 2)
	} else {
		// Normally doRotateSize() should be protected by rmu. But
		// this is the first time and no other thread knows about it
		// yet. So, we don't need the lock.
		if err := r.doRotate(); err != nil {return nil, err}
	}

	return r, nil
}

// Close closes the underlying file. After Close, no more writes or rotations
// are possible. It is safe to call multiple times.
func (r *Rotate) Close() error {
	r.rmu.Lock()
	defer r.rmu.Unlock()
	r.wmu.Lock()
	defer r.wmu.Unlock()
	if r.fh != nil {
		if err := r.fh.Close(); err != nil {return err}
		r.fh = nil
	}
	return nil
}

// doRotate performs the actual file rotation. Must be called with rmu held.
func (r *Rotate) doRotate() error {
	target := filepath.Join(r.dir, r.fn)
	fn := filepath.Join(r.dir, `.`+r.fn)
	f, err := os.OpenFile(
		fn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND, 0600,
	)
	if err != nil {return err}

	defer func(fh *os.File, x string) {
		if err != nil {
			fh.Close()
			os.Remove(fn)
		}
	}(f, fn)

	digits10 := func(n uint) int {
		if n == 0 {return 1}
		d := 0
		for n > 0 {
			d++
			n /= 10
		}
		return d
	}

	width := digits10(r.maxVersions-1)
	suff := func(i uint) string {
		if i == 0 {return ""}
		if i == 1 {return "~"}
		return fmt.Sprintf(`~%0*d`, width, i-1)
	}

	pairs := func(yield func(string, string) bool) {
		for i := r.maxVersions; i > 0; i-- {
			if !yield(
				filepath.Join(r.dir, r.fn+suff(i-1)),
				filepath.Join(r.dir, r.fn+suff(i)),
			) {
				return			// handle "break" or similar in for range loop
			}
		}
		yield(fn, target)
	}

	for from, to := range pairs {
		err = os.Rename(from, to)
		if to != target && errors.Is(err, fs.ErrNotExist) {err = nil}
		if err != nil {return err}
	}
	r.rot++

	old_fh := r.fh

	// fh+cur_sz are used in Write(). So, we need that lock.
	r.wmu.Lock()
	defer r.wmu.Unlock()

	r.fh = f
	r.cur_sz = 0
	if old_fh != nil {old_fh.Close()}

	return nil
}

// Rotate triggers an asynchronous rotation. If a rotation is already in
// progress, this call is a no‑op (the mutex TryLock fails). The rotation
// happens in a separate goroutine. Any error is passed to the ErrorHandler.
//
// This is called automatically by Write when the size threshold is reached,
// but can also be called manually to force a rotation.
func (r *Rotate) Rotate() {
	if r.rmu.TryLock() {
		go func() {
			err := r.doRotate()
			r.rmu.Unlock()

			// errorHnd should not be called while we are holding the lock
			if err != nil {if r.errorHnd != nil {r.errorHnd(r, err)}}
		}()
	}
}

// Name returns the name of the main log file.
func (r *Rotate) Name() string {
	return filepath.Join(r.dir, r.fn)
}

// MaxSize returns the MaxSize parameter
func (r *Rotate) MaxSize() int64 {
	return r.maxSize
}

// NBackups returns the NBackups parameter
func (r *Rotate) NBackups() uint {
	return r.maxVersions
}

// File returns the current underlying *os.File. The returned file handle
// may be closed or replaced by concurrent calls to Close or rotation.
// Use Name() to open your own file handle if you need independent access.
func (r *Rotate) File() *os.File {
	r.wmu.Lock()
	defer r.wmu.Unlock()
	return r.fh
}

// Rotations returns the number of successful rotations performed so far.
func (r *Rotate) Rotations() int64 {
	r.rmu.Lock()
	defer r.rmu.Unlock()
	return r.rot
}

// Write writes len(p) bytes to the file. It returns the number of bytes
// written and any error that occurred. Short writes are handled
// automatically by retrying.
//
// If the cumulative written size exceeds maxSize after this write,
// an asynchronous rotation is triggered.
func (r *Rotate) Write(p []byte) (int, error) {
	need_to_rotate := false
	r.wmu.Lock()
	defer func() {
		r.wmu.Unlock()
		if need_to_rotate {r.Rotate()}
	}()

	f := r.fh

	var err error
	var written, n int = 0, 0
    for len(p) > 0 {
        n, err = f.Write(p)
		written += n
        if err != nil && err != io.ErrShortWrite {
			break
        }
        p = p[n:] // advance the slice
    }
	r.cur_sz += int64(written)
	need_to_rotate = r.maxSize > 0 && r.cur_sz >= r.maxSize

	return written, err
}

// Local Variables:
// tab-width: 4
// End:
