package slogerr

import (
	"fmt"
	"log/slog"
	"strconv"
)

// Error is shorthand for NamedError("error", err).
func Error(err error) slog.Attr {
	return NamedError("error", err)
}

// NamedError constructs a slog.Attr that stores error information under key.
// Returns a zero-value Attr (no-op) when err is nil.
// If err implements slog.LogValuer, it is passed directly to slog.Any so that
// its LogValue() output is used as-is.
// Otherwise, errors get a "msg" sub-key and, when applicable, "verbose" and "causes".
func NamedError(key string, err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	if _, ok := err.(slog.LogValuer); ok {
		return slog.Any(key, err)
	}
	return slog.Any(key, &errValue{err: err})
}

type errValue struct {
	err error
}

type multiUnwrapper interface {
	Unwrap() []error
}

func (e *errValue) LogValue() slog.Value {
	return errorValue(e.err)
}

// ErrorValue converts err into a slog.Value that carries its message and, when
// applicable, "verbose" and "causes" sub-attributes. It returns the same value
// that NamedError logs for a non-LogValuer error.
func ErrorValue(err error) slog.Value {
	return errorValue(err)
}

func errorValue(err error) slog.Value {
	msg := err.Error()

	_, hasFormatter := err.(fmt.Formatter)
	mu, hasMulti := err.(multiUnwrapper)

	n := 1
	if hasFormatter {
		n++
	}
	if hasMulti {
		n++
	}

	attrs := make([]slog.Attr, 0, n)
	attrs = append(attrs, slog.String("msg", msg))

	// %+v only differs from %v when the error implements fmt.Formatter, so skip
	// the allocation-heavy formatting otherwise.
	if hasFormatter {
		if verbose := fmt.Sprintf("%+v", err); verbose != msg {
			attrs = append(attrs, slog.String("verbose", verbose))
		}
	}

	if hasMulti {
		errs := mu.Unwrap()
		causeAttrs := make([]slog.Attr, 0, len(errs))
		for i, cause := range errs {
			if cause == nil {
				continue
			}
			causeAttrs = append(causeAttrs, slog.Attr{Key: strconv.Itoa(i), Value: errorValue(cause)})
		}
		if len(causeAttrs) > 0 {
			attrs = append(attrs, slog.Attr{Key: "causes", Value: slog.GroupValue(causeAttrs...)})
		}
	}

	return slog.GroupValue(attrs...)
}
