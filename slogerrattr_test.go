package slogerrattr

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"
)

// formatterErr is an error that produces a richer %+v output than %v.
type formatterErr struct {
	msg     string
	verbose string
}

func (e *formatterErr) Error() string { return e.msg }

func (e *formatterErr) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('+') {
		fmt.Fprint(f, e.verbose)
		return
	}
	fmt.Fprint(f, e.msg)
}

func TestError_nil(t *testing.T) {
	got := Error(nil)
	if !got.Equal(slog.Attr{}) {
		t.Errorf("Error(nil) = %v, want zero Attr", got)
	}
}

func TestError_simple(t *testing.T) {
	err := errors.New("something went wrong")
	attr := Error(err)

	if attr.Key != "error" {
		t.Errorf("key = %q, want %q", attr.Key, "error")
	}

	group := attr.Value.Resolve()
	if group.Kind() != slog.KindGroup {
		t.Fatalf("value kind = %v, want Group", group.Kind())
	}
	attrs := group.Group()
	if len(attrs) != 1 {
		t.Fatalf("len(attrs) = %d, want 1", len(attrs))
	}
	if attrs[0].Key != "msg" || attrs[0].Value.String() != "something went wrong" {
		t.Errorf("attrs[0] = %v, want msg=something went wrong", attrs[0])
	}
}

func TestError_verbose(t *testing.T) {
	err := &formatterErr{msg: "oops", verbose: "oops\n  stack trace here"}
	attr := Error(err)

	group := attr.Value.Resolve()
	attrs := group.Group()
	if len(attrs) != 2 {
		t.Fatalf("len(attrs) = %d, want 2", len(attrs))
	}
	if attrs[0].Key != "msg" || attrs[0].Value.String() != "oops" {
		t.Errorf("attrs[0] = %v, want msg=oops", attrs[0])
	}
	if attrs[1].Key != "verbose" || attrs[1].Value.String() != "oops\n  stack trace here" {
		t.Errorf("attrs[1] = %v, want verbose=oops\\n  stack trace here", attrs[1])
	}
}

func TestNamedError_nil(t *testing.T) {
	got := NamedError("err", nil)
	if !got.Equal(slog.Attr{}) {
		t.Errorf("NamedError(\"err\", nil) = %v, want zero Attr", got)
	}
}

func TestNamedError_customKey(t *testing.T) {
	err := errors.New("custom key error")
	attr := NamedError("myErr", err)

	if attr.Key != "myErr" {
		t.Errorf("key = %q, want %q", attr.Key, "myErr")
	}

	group := attr.Value.Resolve()
	attrs := group.Group()
	if len(attrs) != 1 || attrs[0].Value.String() != "custom key error" {
		t.Errorf("unexpected attrs: %v", attrs)
	}
}
