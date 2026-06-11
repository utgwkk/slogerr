package slogerrattr

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
// Errors that produce a different %+v output than Error() also get a "verbose" sub-key.
func NamedError(key string, err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
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
	msg := e.err.Error()
	attrs := []slog.Attr{slog.String("msg", msg)}
	if verbose := fmt.Sprintf("%+v", e.err); verbose != msg {
		attrs = append(attrs, slog.String("verbose", verbose))
	}
	if mu, ok := e.err.(multiUnwrapper); ok {
		errs := mu.Unwrap()
		causeAttrs := make([]slog.Attr, 0, len(errs))
		for i, cause := range errs {
			if cause == nil {
				continue
			}
			causeAttrs = append(causeAttrs, slog.Any(strconv.Itoa(i), &errValue{err: cause}))
		}
		if len(causeAttrs) > 0 {
			attrs = append(attrs, slog.Attr{Key: "causes", Value: slog.GroupValue(causeAttrs...)})
		}
	}
	return slog.GroupValue(attrs...)
}
