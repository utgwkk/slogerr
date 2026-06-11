package slogerrattr

import (
	"errors"
	"log/slog"
	"testing"
)

// sink prevents the compiler from optimizing away benchmark results.
var sink slog.Attr

func BenchmarkError_nil(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = Error(nil)
	}
}

func BenchmarkError_simple(b *testing.B) {
	err := errors.New("something went wrong")
	b.ReportAllocs()
	for b.Loop() {
		sink = Error(err)
	}
}

// BenchmarkError_simple_resolve measures the cost of creating the attr and
// resolving the LogValue (i.e. calling LogValue() which formats the error).
func BenchmarkError_simple_resolve(b *testing.B) {
	err := errors.New("something went wrong")
	b.ReportAllocs()
	for b.Loop() {
		sink = Error(err)
		sink.Value.Resolve()
	}
}

// BenchmarkError_verbose_resolve measures the extra cost of the verbose path
// when the error implements fmt.Formatter with a richer %+v output.
func BenchmarkError_verbose_resolve(b *testing.B) {
	err := &formatterErr{msg: "oops", verbose: "oops\n  stack trace here"}
	b.ReportAllocs()
	for b.Loop() {
		sink = Error(err)
		sink.Value.Resolve()
	}
}

// BenchmarkError_joinedErrors_resolve measures the cost for errors.Join with
// two causes.
func BenchmarkError_joinedErrors_resolve(b *testing.B) {
	err := errors.Join(errors.New("db error"), errors.New("timeout"))
	b.ReportAllocs()
	for b.Loop() {
		sink = Error(err)
		sink.Value.Resolve()
	}
}

// BenchmarkError_joinedErrors10_resolve measures how the cost scales with ten
// causes.
func BenchmarkError_joinedErrors10_resolve(b *testing.B) {
	errs := make([]error, 10)
	for i := range errs {
		errs[i] = errors.New("error")
	}
	err := errors.Join(errs...)
	b.ReportAllocs()
	for b.Loop() {
		sink = Error(err)
		sink.Value.Resolve()
	}
}
