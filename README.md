# slogerr

[![Go Reference](https://pkg.go.dev/badge/github.com/utgwkk/slogerr.svg)](https://pkg.go.dev/github.com/utgwkk/slogerr)

`slogerr` provides `Error` and `NamedError` helpers for Go's `log/slog` package, inspired by the equivalent functions in [uber-go/zap](https://github.com/uber-go/zap).

Each helper returns a `slog.Attr` that encodes structured error information:

- **`msg`** — the error message from `err.Error()`
- **`verbose`** — the `%+v` representation, added only when it differs from `msg` (e.g. errors carrying a stack trace)
- **`causes`** — the individual errors when `err` wraps multiple errors via `errors.Join` or `fmt.Errorf("%w %w", ...)`

## Installation

```
go get github.com/utgwkk/slogerr
```

## Usage

```go
import (
    "errors"
    "log/slog"

    "github.com/utgwkk/slogerr"
)

// Basic usage — key is "error"
slog.Info("request failed", slogerr.Error(err))

// Custom key
slog.Info("request failed", slogerr.NamedError("cause", err))

// nil is a no-op: returns a zero-value slog.Attr that handlers skip
slog.Info("maybe failed", slogerr.Error(nil))

// errors.Join: individual causes are recorded under "causes"
joined := errors.Join(errors.New("db error"), errors.New("timeout"))
slog.Info("multiple errors", slogerr.Error(joined))
```

## Output format

The attribute value is a `slog.GroupValue`, so with `slog.NewJSONHandler` the output looks like:

**Simple error**
```json
{"msg":"request failed","error":{"msg":"something went wrong"}}
```

**Error with verbose representation** (e.g. from `github.com/pkg/errors`)
```json
{"msg":"request failed","error":{"msg":"something went wrong","verbose":"something went wrong\nmain.main()\n\t/app/main.go:42"}}
```

**Joined errors**
```json
{"msg":"request failed","error":{"msg":"db error\ntimeout","causes":{"0":{"msg":"db error"},"1":{"msg":"timeout"}}}}
```

> **Note:** `slog`'s type system has no native array kind; causes are represented as a group with numeric string keys (`"0"`, `"1"`, ...) rather than a JSON array.

## API

```go
// Error is shorthand for NamedError("error", err).
func Error(err error) slog.Attr

// NamedError constructs a slog.Attr that stores error information under key.
// Returns a zero-value Attr (no-op) when err is nil.
func NamedError(key string, err error) slog.Attr
```

## Acknowledgments

This library is inspired by the `Error` and `NamedError` functions in
[uber-go/zap](https://github.com/uber-go/zap) (MIT License, Copyright (c) 2016–2024 Uber Technologies, Inc.).
