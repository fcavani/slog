# Slog

[![Build Status](https://travis-ci.org/fcavani/log.svg?branch=master)](https://travis-ci.org/fcavani/slog) [![GoDoc](https://godoc.org/github.com/fcavani/log?status.svg)](https://godoc.org/github.com/fcavani/slog)
[![Go Report Card](https://goreportcard.com/badge/github.com/fcavani/slog)](https://goreportcard.com/report/github.com/fcavani/slog)

Slog (slow log) is a attempt to make a logger with more features than the standard go logger
but with similar performance. The logger feature levels, tags, flexibility to
implement differents formatters and committers. It´s simple and easy to
use. Import the package, use the free functions and you will have a logger
to the console. If you want to log to a file change the writer.

## Performance

You can see bellow a simple comparison of some loggers.
First is the baseline logger, the go logger, with is simple but fast.
The others loggers with more functionalities have a slower performance,
as expected, but slog without debug information (Di) have a good set of features
with a performance better than Logrus.

| Benchmark name | N | Time |
|--------------------|-------|----------|
|BenchmarkPureGolog-4|300000|3980 ns/op|
|BenchmarkLogrus-4|200000|7912 ns/op|
|BenchmarkSlogNullFile-4|200000|9052 ns/op|
|BenchmarkSlogJSONNullFile-4|200000|8926 ns/op|
|BenchmarkSlogNullFileNoDi-4|200000|5662 ns/op|
|BenchmarkSlogJSONNullFileNoDi-4|200000|5716 ns/op|

Some optimizations will be needed before slog can be used like a
high-performance logger. I need to get deeper into go and learn
to do some optimizations to achieve it, mainly for the debug information.

## Bottlenecks

Slog have 2 main bottlenecks:

- Io bottleneck: this occurs when committer send the data to disk or to some db.
- Log message assemble: the log message is assembled in a byte slice and uses the append
function that make things slow. Message formatting is a problem too, mainly the
date and time formatting.
- Debug information (line number and file name) is a trouble. I need to come with
a solution that make it more adequate for production. Debug information in
a test environment is ok.

For the io bottleneck there's no safe solution besides buy a fast hardware. The
in memory approach may be good for some tasks but its not safe if something
wrong happen.

## TODO

- Need to check all code for allocations and minimize that.
- A more flexible way to deal with date and time.

## Conclusion

Slog is a logger with more features and have a good performance but I need
to make some optimizations to make it more fast.
