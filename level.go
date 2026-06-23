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

package log

import (
	"errors"
	"strings"
)

// Level represents a log severity level. Lower values indicate higher severity.
// The zero value of Level is PANIC. Use ToLevel or ParseLevel to obtain a
// valid Level.
type Level int8
const (
	// NOTICE is a special level that always prints, regardless of the
	// logger's threshold. It is the only level that can be lower than PANIC.
	NOTICE Level = iota - 1
	// PANIC is the highest severity. Logging at this level will also call
	// os.Exit(1).
	PANIC
	// ERROR indicates an error condition.
	ERROR
	// WARN indicates a warning condition.
	WARN
	// INFO indicates an informational message.
	INFO
	// DEBUG indicates a debug message.
	DEBUG
	// DEBG2 is a more verbose debug level.
	DEBG2
	// DEBG3 is an even more verbose debug level.
	DEBG3
	// DEBG4 is a highly verbose debug level.
	DEBG4
	// DEBG5 is the most verbose debug level.
	DEBG5
)

var levels []string = []string{
	`NOTICE`,
	`PANIC`,
	`ERROR`,
	`WARN`,
	`INFO`,
	`DEBUG`,
	`DEBG2`,
	`DEBG3`,
	`DEBG4`,
	`DEBG5`,
}

// ToLevel converts an integer to a Level, clamping it to the valid range.
// Values above DEBG5 are clamped to DEBG5; values below PANIC are clamped to
// PANIC.
func ToLevel(i int) Level {
	if i > int(DEBG5) {return DEBG5}

	// PANIC is the lowest real level. NOTICE means to always print.
	// That's why everything below the normal range becomes PANIC.
	if i < int(PANIC) {return PANIC}
	return Level(i)
}

// String returns the human-readable name of the level.
// It returns one of: "NOTICE", "PANIC", "ERROR", "WARN", "INFO",
// "DEBUG", "DEBG2", "DEBG3", "DEBG4", "DEBG5".
func (l Level) String() string {
	return levels[int8(l)+1]
}

// ErrInvalidLevel is returned by ParseLevel when the input string does not
// match any known log level.
var ErrInvalidLevel error = errors.New(`invalid log level`)

// ParseLevel parses a string into a Level.
// It accepts case-insensitive names: "notice", "panic", "error",
// "warn"/"warning",
// "info", "debug", "debug2"/"debg2", "debug3"/"debg3", "debug4"/"debg4",
// "debug5"/"debg5", or the numeric strings "0" through "8".
// Returns ErrInvalidLevel if the string does not match.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "notice":
		return NOTICE, nil
	case "panic", "0":
		return PANIC, nil
	case "error", "1":
		return ERROR, nil
	case "warn", "warning", "2":
		return WARN, nil
	case "info", "3":
		return INFO, nil
	case "debug", "4":
		return DEBUG, nil
	case "debug2", "debg2", "5":
		return DEBG2, nil
	case "debug3", "debg3", "6":
		return DEBG3, nil
	case "debug4", "debg4", "7":
		return DEBG4, nil
	case "debug5", "debg5", "8":
		return DEBG5, nil
	default:
		return PANIC, ErrInvalidLevel
	}
}

// Local Variables:
// tab-width: 4
// End:
