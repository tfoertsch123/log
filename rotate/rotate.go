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

type ErrorHnd func(*Rotate, error)()
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

func WithFileName(f string) Opt {return func(x *descr) {x.fn = f}}
func WithMaxSize(f int64) Opt {return func(x *descr) {x.ms, x.rot = f, 0}}
func WithNBackups(f uint) Opt {return func(x *descr) {x.mv = f}}
func WithErrorHandler(f ErrorHnd) Opt {return func(x *descr) {x.hnd = f}}
func WithInitialAppend(f bool) Opt {return func(x *descr) {x.append = f}}

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

func (r *Rotate) Close() error {
	if r.fh != nil {
		if err := r.fh.Close(); err != nil {return err}
		r.fh = nil
	}
	return nil
}

// This is supposed to be protected by rmu.
func (r *Rotate) doRotate() error {
	fn := filepath.Join(r.dir, `.`+r.fn)
	f, err := os.OpenFile(
		fn, os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_APPEND, 0600,
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
		yield(fn, filepath.Join(r.dir, r.fn))
	}

	for from, to := range pairs {
		err = os.Rename(from, to)
		if to != r.fn && errors.Is(err, fs.ErrNotExist) {err = nil}
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

func (r *Rotate) Rotate() {
	if r.rmu.TryLock() {
		go func() {
			err := r.doRotate()
			r.rmu.Unlock()

			// errorHnd should not be called while we are holding the lock
			if err != nil {r.errorHnd(r, err)}
		}()
	}
}

func (r *Rotate) Name() string {return filepath.Join(r.dir, r.fn)}
func (r *Rotate) File() *os.File {
	r.wmu.Lock()
	defer r.wmu.Unlock()
	return r.fh
}
func (r *Rotate) Rotations() int64 {
	r.rmu.Lock()
	defer r.rmu.Unlock()
	return r.rot
}

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
