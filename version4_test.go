package fastuuid

import (
	"errors"
	"github.com/rekby/fastuuid/internal"
	"github.com/stretchr/testify/require"
	"io"
	"regexp"
	"testing"
)

var uuidRegExp = regexp.MustCompile(`^[0-9a-f]{8}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{4}\b-[0-9a-f]{12}$`)

type errorReader struct{}

func (p errorReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestMustUUIDv4(t *testing.T) {
	v1 := MustUUIDv4()
	v2 := MustUUIDv4()

	require.NotEqual(t, v1, v2)
	require.Panics(t, func() {
		internal.SetRandomSource(errorReader{})
		defer internal.SetRandomSource(nil)

		MustUUIDv4()
	})
}

func TestMustUUIDv4String(t *testing.T) {
	v1 := MustUUIDv4String()
	v2 := MustUUIDv4String()

	require.Regexp(t, uuidRegExp, v1)
	require.Regexp(t, uuidRegExp, v2)
	require.NotEqual(t, v1, v2)

	require.Panics(t, func() {
		internal.SetRandomSource(errorReader{})
		defer internal.SetRandomSource(nil)

		MustUUIDv4String()
	})
}

func TestMustUUIDv4StringByte(t *testing.T) {
	v1 := make([]byte, 36)
	v2 := make([]byte, 36)
	MustUUIDv4StringBytes(v1)
	MustUUIDv4StringBytes(v2)

	require.Regexp(t, uuidRegExp, string(v1))
	require.Regexp(t, uuidRegExp, string(v2))
	require.NotEqual(t, v1, v2)

	require.Panics(t, func() {
		internal.SetRandomSource(errorReader{})
		defer internal.SetRandomSource(nil)

		short := make([]byte, 15)
		MustUUIDv4StringBytes(short)
	})

	require.Panics(t, func() {
		internal.SetRandomSource(errorReader{})
		defer internal.SetRandomSource(nil)

		MustUUIDv4StringBytes(v1)
	})
}

func BenchmarkMustUUIDv4(b *testing.B) {
	b.ReportAllocs()

	var s [16]byte
	for i := 0; i < b.N; i++ {
		s = MustUUIDv4()
	}
	_, _ = io.Discard.Write(s[:])
}

func BenchmarkMustUUIDv4String(b *testing.B) {
	b.ReportAllocs()

	var s string
	for i := 0; i < b.N; i++ {
		s = MustUUIDv4String()
	}
	_, _ = io.WriteString(io.Discard, s)
}

func BenchmarkMustUUIDv4StringByte(b *testing.B) {
	b.ReportAllocs()

	buf := make([]byte, 36)
	for i := 0; i < b.N; i++ {
		MustUUIDv4StringBytes(buf)
	}
	_, _ = io.Discard.Write(buf)
}
