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

func findAttr(attrs []slog.Attr, key string) (slog.Attr, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a, true
		}
	}
	return slog.Attr{}, false
}

func TestError_joinedErrors(t *testing.T) {
	err := errors.Join(errors.New("a"), errors.New("b"))
	attr := Error(err)

	group := attr.Value.Resolve()
	attrs := group.Group()

	causesAttr, ok := findAttr(attrs, "causes")
	if !ok {
		t.Fatal("expected causes attr, not found")
	}
	causes := causesAttr.Value.Group()
	if len(causes) != 2 {
		t.Fatalf("len(causes) = %d, want 2", len(causes))
	}
	if causes[0].Key != "0" || causes[0].Value.Resolve().Group()[0].Value.String() != "a" {
		t.Errorf("causes[0] = %v, want 0.msg=a", causes[0])
	}
	if causes[1].Key != "1" || causes[1].Value.Resolve().Group()[0].Value.String() != "b" {
		t.Errorf("causes[1] = %v, want 1.msg=b", causes[1])
	}
}

func TestError_joinedErrors_withNil(t *testing.T) {
	err := errors.Join(errors.New("a"), nil, errors.New("b"))
	attr := Error(err)

	group := attr.Value.Resolve()
	causesAttr, ok := findAttr(group.Group(), "causes")
	if !ok {
		t.Fatal("expected causes attr, not found")
	}
	causes := causesAttr.Value.Group()
	if len(causes) != 2 {
		t.Fatalf("len(causes) = %d, want 2 (nil should be skipped)", len(causes))
	}
}

// multiErr is an error whose Unwrap returns the slice as-is, including any nils.
type multiErr struct {
	msg  string
	errs []error
}

func (e *multiErr) Error() string    { return e.msg }
func (e *multiErr) Unwrap() []error  { return e.errs }

// TestError_multiUnwrapperWithNilCause tests that nil entries returned by
// Unwrap() []error are skipped (lines 65-67 in slogerrattr.go).
// errors.Join filters nils before storing, so a custom type is needed here.
func TestError_multiUnwrapperWithNilCause(t *testing.T) {
	err := &multiErr{
		msg:  "multi",
		errs: []error{errors.New("a"), nil, errors.New("b")},
	}
	attr := Error(err)

	group := attr.Value.Resolve()
	causesAttr, ok := findAttr(group.Group(), "causes")
	if !ok {
		t.Fatal("expected causes attr, not found")
	}
	causes := causesAttr.Value.Group()
	if len(causes) != 2 {
		t.Fatalf("len(causes) = %d, want 2 (nil cause should be skipped)", len(causes))
	}
	if causes[0].Key != "0" || causes[0].Value.Resolve().Group()[0].Value.String() != "a" {
		t.Errorf("causes[0] = %v, want 0.msg=a", causes[0])
	}
	if causes[1].Key != "2" || causes[1].Value.Resolve().Group()[0].Value.String() != "b" {
		t.Errorf("causes[1] = %v, want 2.msg=b", causes[1])
	}
}

// TestError_multiUnwrapperAllNilCauses tests that when all causes are nil,
// no "causes" attr is added to the output.
func TestError_multiUnwrapperAllNilCauses(t *testing.T) {
	err := &multiErr{
		msg:  "all nil causes",
		errs: []error{nil, nil},
	}
	attr := Error(err)

	group := attr.Value.Resolve()
	_, ok := findAttr(group.Group(), "causes")
	if ok {
		t.Fatal("expected no causes attr when all causes are nil")
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
