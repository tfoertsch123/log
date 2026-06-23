# log
A logging package for Go.

## Installation

```bash
go get -u github.com/tfoertsch123/log
```

## What is it?

Classical logging with log levels and messages. No structured logging.

The following log levels are defined:

- `NOTICE` - printed always
- `PANIC` - print + `exit(1)`
- `ERROR`
- `WARN`
- `INFO`
- `DEBUG`
- `DEBG2`
- `DEBG3`
- `DEBG4`
- `DEBG5`

The log target is an `io.Writer`, by default `os.Stderr`.

A logger can have a topic or subsystem.

Log messages look like so:

```
TIMESTAMP LEVEL [TOPIC] (CODE LOCATION) MESSAGE
```

Topic and code location are optional.

Examples:
```
2026-10-05 09:02:05.987654 INFO message without a topic
2026-10-05 09:02:05.987654 DEBUG (log/exmpl01_log_test.go:21) with code location
2026-10-05 09:02:05.987654 DEBUG [TPC] (log/exmpl01_log_test.go:27) with topic and location
```

The timestamp can be configured. Default precision is microseconds.

The code location is printed for a certain log level and above. The file
name is printed with a specific number of directory components.
This can also be configured.

The package allows you to construct a tree of loggers. Any configuration
change with the exception of the topic propagates to all of its kids
recursively.

That allows you for instance to set up static loggers with different topics
and later modify the log level for all of them.

```go
var base *log.Logger = log.NewR(log.WithOutput(...))
var tpc1 *log.Logger = base.New(log.WithTopic("TOPIC1"))
var tpc2 *log.Logger = base.New(log.WithTopic("TOPIC2"))

func subsys1() {
    tpc1.Info(`message`)
}

func subsys2() {
    tpc2.Info(`message`)
}

func main() {
    base.SetLevel(log.DEBUG)
    ...
}
```

Alternatively, you can rely on the default logger:

```go
func subsys1() {
    log.NewC(log.WithTopic(`TOPIC1`))
    defer log.Close()

    log.Info(`message`)
}

func subsys2() {
    log.NewC(log.WithTopic(`TOPIC2`))
    defer log.Close()

    log.Info(`message`)
}

func main() {
    log.SetLevel(log.DEBUG)
    log.SetOutput(...)
}
```
