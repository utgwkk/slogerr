package slogerr

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
// Unwrap() []error are skipped (lines 65-67 in slogerr.go).
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

// logValueErr is an error that also implements slog.LogValuer.
type logValueErr struct {
	msg    string
	fields []slog.Attr
}

func (e *logValueErr) Error() string { return e.msg }
func (e *logValueErr) LogValue() slog.Value {
	return slog.GroupValue(e.fields...)
}

func TestError_logValuer(t *testing.T) {
	// When error implements slog.LogValuer, its LogValue() result is used directly
	// as the attribute value — no wrapping with msg/verbose/causes.
	err := &logValueErr{
		msg: "something failed",
		fields: []slog.Attr{
			slog.String("request_id", "abc-123"),
			slog.Int("code", 42),
		},
	}
	attr := Error(err)

	resolved := attr.Value.Resolve()
	if resolved.Kind() != slog.KindGroup {
		t.Fatalf("value kind = %v, want Group", resolved.Kind())
	}
	attrs := resolved.Group()

	if len(attrs) != 2 {
		t.Fatalf("len(attrs) = %d, want 2; attrs = %v", len(attrs), attrs)
	}

	ridAttr, ok := findAttr(attrs, "request_id")
	if !ok {
		t.Fatal("expected request_id attr, not found")
	}
	if ridAttr.Value.String() != "abc-123" {
		t.Errorf("request_id = %q, want %q", ridAttr.Value.String(), "abc-123")
	}

	codeAttr, ok := findAttr(attrs, "code")
	if !ok {
		t.Fatal("expected code attr, not found")
	}
	if codeAttr.Value.Int64() != 42 {
		t.Errorf("code = %d, want 42", codeAttr.Value.Int64())
	}

	// msg sub-key from errValue should NOT be present.
	if _, ok := findAttr(attrs, "msg"); ok {
		t.Error("unexpected msg attr: LogValuer errors should not be wrapped")
	}
}

func TestError_logValuer_emptyGroup(t *testing.T) {
	// LogValuer returning an empty group produces an empty group value.
	err := &logValueErr{msg: "plain error", fields: nil}
	attr := Error(err)
	resolved := attr.Value.Resolve()
	if resolved.Kind() != slog.KindGroup {
		t.Fatalf("value kind = %v, want Group", resolved.Kind())
	}
	if len(resolved.Group()) != 0 {
		t.Errorf("expected empty group, got %v", resolved.Group())
	}
}

// formatterLogValueErr implements error, fmt.Formatter, and slog.LogValuer.
type formatterLogValueErr struct {
	msg     string
	verbose string
	fields  []slog.Attr
}

func (e *formatterLogValueErr) Error() string { return e.msg }

func (e *formatterLogValueErr) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('+') {
		fmt.Fprint(f, e.verbose)
		return
	}
	fmt.Fprint(f, e.msg)
}

func (e *formatterLogValueErr) LogValue() slog.Value {
	return slog.GroupValue(e.fields...)
}

func TestError_logValuer_withVerbose(t *testing.T) {
	// slog.LogValuer takes priority: even if the error also implements fmt.Formatter,
	// the output is solely what LogValue() returns — no msg/verbose wrapping.
	err := &formatterLogValueErr{
		msg:     "rich error",
		verbose: "rich error\n  stack detail",
		fields:  []slog.Attr{slog.String("trace_id", "xyz")},
	}
	attr := Error(err)
	resolved := attr.Value.Resolve()
	attrs := resolved.Group()

	traceAttr, ok := findAttr(attrs, "trace_id")
	if !ok {
		t.Fatal("expected trace_id attr from LogValuer, not found")
	}
	if traceAttr.Value.String() != "xyz" {
		t.Errorf("trace_id = %q, want %q", traceAttr.Value.String(), "xyz")
	}

	if _, ok := findAttr(attrs, "msg"); ok {
		t.Error("unexpected msg attr: LogValuer errors should not be wrapped")
	}
	if _, ok := findAttr(attrs, "verbose"); ok {
		t.Error("unexpected verbose attr: LogValuer errors should not be wrapped")
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
