package log

import (
	"os"
	"io"
	"errors"
	"net/url"
	"strings"
	"strconv"

	"github.com/alecthomas/units"

	"github.com/tfoertsch123/log/rotate"
)

var ErrInvalidFormat error = errors.New(`invalid format`)

// URL-like Format:
//   file://LEVEL@/path/to/file?key=value&key=value...

func ParseURL(s string, hnd func(error)) ([]Opt, error) {
	// helper
	e := func(err error) ([]Opt, error) {
		return nil, err
	}
	
	u, err := url.Parse(s)
	if err != nil {return e(err)}

	var rotopt []rotate.Opt
	var opt []Opt
	var file *os.File
	var level *Level

	if len(u.Scheme) > 0 && u.Scheme != "file" {
		// for now we only accept file://... or no schema
		return e(ErrInvalidFormat)
	}

	if u.User != nil {
		usr := u.User.String()
		if len(usr) > 0 {
			// accept a log level in User: //INFO@/...
			if lvl, err := ParseLevel(usr); lvl >= PANIC && err == nil {
				level = &lvl
			} else if err != nil {
				return e(err)
			} else {
				return e(ErrInvalidFormat)
			}
		}
	}

	switch strings.ToLower(u.Host) {
	case ".":
		// a single dot to indicates a relative path:
		// //INFO@./path/to/file.log
		if strings.HasPrefix(u.Path, "/./") {
			// //INFO@././path/to/file.log is invalid
			// better: //INFO@./path/to/file.log
			return e(ErrInvalidFormat)
		}
		
		// skip the initial slash
		if strings.HasPrefix(u.Path, "/") {
			rotopt = append(rotopt, rotate.WithFileName(u.Path[1:]))
		} else {
			return e(ErrInvalidFormat)
		}

	case "":
		if strings.HasPrefix(u.Path, "/./") {
			// //INFO@/./path/to/file.log is a relative file
			rotopt = append(rotopt, rotate.WithFileName(u.Path[3:]))
		} else {
			// //INFO@/path/to/file.log is an absolute file
			rotopt = append(rotopt, rotate.WithFileName(u.Path[0:]))
		}

	case "stdout":
		// //INFO@stdout
		// Path must be empty in this case
		if len(u.Path) > 0 {
			return e(ErrInvalidFormat)
		}
		
		file = os.Stdout

	case "stderr":
		// //INFO@stderr
		// Path must be empty in this case
		if len(u.Path) > 0 {
			return e(ErrInvalidFormat)
		}
		
		file = os.Stderr

	default:
		fd, err := strconv.ParseUint(u.Host, 10, 64)
		if err != nil {
			return e(ErrInvalidFormat)
		}

		// pass in an open file descriptor
		// //INFO@19
		// Path must be empty in this case
		if len(u.Path) > 0 {
			return e(ErrInvalidFormat)
		}
		file = os.NewFile(uintptr(fd), "/proc/self/fd/"+u.Host)
		// quick check if it is usable
		if _, err = file.Stat(); err != nil {
			return e(err)
		}
	}

	param := u.Query()

	known_params := []string{
		// logger params
		"level",
		"timeformat",
		"multiline",
		"minloc",
		"locdirs",
		"topic",

		// rotate params
		"maxsize",
		"nbackups",
		"append",
	}

	for _, p := range known_params {
		if s, ok := param[p]; ok {
			delete(param, p)
			if len(s) != 1 {
				return e(ErrInvalidFormat)
			}

			switch p {
			case "level":
				l, err := ParseLevel(s[0])
				if l < PANIC || err != nil {
					return e(err)
				}
				if level != nil && *level != l {
					return e(ErrInvalidFormat)
				}
				level = &l

			case "timeformat":
				opt = append(opt, WithTimeFmt(s[0]))
			
			case "multiline":
				switch s[0] {
				case "on", "1", "true", "yes":
					opt = append(opt, WithMultiLine(true))
				default:
					opt = append(opt, WithMultiLine(false))
				}

			case "minloc":
				l, err := ParseLevel(s[0])
				if l < PANIC || err != nil {
					return e(err)
				}
				opt = append(opt, WithMinLocation(l))

			case "locdirs":
				n, err := strconv.ParseInt(s[0], 10, 64)
				if err != nil {
					return e(ErrInvalidFormat)
				}
				opt = append(opt, WithLocDirectories(int(n)))

			case "topic":
				opt = append(opt, WithTopic(s[0]))

				// rotate params
			case "maxsize":
				if x, err := strconv.ParseUint(s[0], 10, 64); err == nil {
					rotopt = append(rotopt, rotate.WithMaxSize(int64(x)))
				} else if x, err := units.ParseStrictBytes(s[0]); err == nil {
					rotopt = append(rotopt, rotate.WithMaxSize(int64(x)))
				} else {
					return e(ErrInvalidFormat)
				}

			case "nbackups":
				if x, err := strconv.ParseUint(s[0], 10, 64); err == nil {
					rotopt = append(rotopt, rotate.WithNBackups(uint(x)))
				} else {
					return e(ErrInvalidFormat)
				}

			case "append":
				switch s[0] {
				case "on", "1", "true", "yes":
					rotopt = append(rotopt, rotate.WithInitialAppend(true))
				default:
					rotopt = append(rotopt, rotate.WithInitialAppend(false))
				}
			}
		}
	}

	if len(param) > 0 {
		// invalid keywords in Query
		return e(ErrInvalidFormat)
	}
	
	if file != nil {
		if len(rotopt) > 0 {
			// rotate options are invalid for a simple *os.File output
			return e(ErrInvalidFormat)
		}
		opt = append(opt, WithOutput(file))
	} else {
		opt = append(opt, WithOutputFactory(
			func() (io.Writer, error) {
				return rotate.New(rotopt...)
			},
			hnd,
		))
	}

	if level != nil {
		opt = append(opt, WithLevel(*level))
	}

	return opt, nil
}

// Local Variables:
// tab-width: 4
// End:
