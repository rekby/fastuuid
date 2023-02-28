// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ibufio_test

import (
	"bytes"
	"errors"
	"fmt"
	. "github.com/rekby/fastuuid/internal/ibufio"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

// Reads from a reader and rot13s the result.
type rot13Reader struct {
	r io.Reader
}

func newRot13Reader(r io.Reader) *rot13Reader {
	r13 := new(rot13Reader)
	r13.r = r
	return r13
}

func (r13 *rot13Reader) Read(p []byte) (int, error) {
	n, err := r13.r.Read(p)
	for i := 0; i < n; i++ {
		c := p[i] | 0x20 // lowercase byte
		if 'a' <= c && c <= 'm' {
			p[i] += 13
		} else if 'n' <= c && c <= 'z' {
			p[i] -= 13
		}
	}
	return n, err
}

// Call ReadByte to accumulate the text of a file
func readBytes(buf *Reader) string {
	var b [1000]byte
	nb := 0
	for {
		c, err := buf.ReadByte()
		if err == io.EOF {
			break
		}
		if err == nil {
			b[nb] = c
			nb++
		} else if err != iotest.ErrTimeout {
			panic("Data: " + err.Error())
		}
	}
	return string(b[0:nb])
}

func TestReaderSimple(t *testing.T) {
	data := "hello world"
	b := NewReader(strings.NewReader(data))
	if s := readBytes(b); s != "hello world" {
		t.Errorf("simple hello world test failed: got %q", s)
	}

	b = NewReader(newRot13Reader(strings.NewReader(data)))
	if s := readBytes(b); s != "uryyb jbeyq" {
		t.Errorf("rot13 hello world test failed: got %q", s)
	}
}

type readMaker struct {
	name string
	fn   func(io.Reader) io.Reader
}

var readMakers = []readMaker{
	{"full", func(r io.Reader) io.Reader { return r }},
	{"byte", iotest.OneByteReader},
	{"half", iotest.HalfReader},
	{"data+err", iotest.DataErrReader},
	{"timeout", iotest.TimeoutReader},
}

// Call ReadString (which ends up calling everything else)
// to accumulate the text of a file.
func readLines(b *Reader) string {
	s := ""
	for {
		s1, err := b.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil && err != iotest.ErrTimeout {
			panic("GetLines: " + err.Error())
		}
		s += s1
	}
	return s
}

// Call Read to accumulate the text of a file
func reads(buf *Reader, m int) string {
	var b [1000]byte
	nb := 0
	for {
		n, err := buf.Read(b[nb : nb+m])
		nb += n
		if err == io.EOF {
			break
		}
	}
	return string(b[0:nb])
}

type bufReader struct {
	name string
	fn   func(*Reader) string
}

var bufreaders = []bufReader{
	{"1", func(b *Reader) string { return reads(b, 1) }},
	{"2", func(b *Reader) string { return reads(b, 2) }},
	{"3", func(b *Reader) string { return reads(b, 3) }},
	{"4", func(b *Reader) string { return reads(b, 4) }},
	{"5", func(b *Reader) string { return reads(b, 5) }},
	{"7", func(b *Reader) string { return reads(b, 7) }},
	{"bytes", readBytes},
	{"lines", readLines},
}

const minReadBufferSize = 16

var bufsizes = []int{
	0, minReadBufferSize, 23, 32, 46, 64, 93, 128, 1024, 4096,
}

func TestReader(t *testing.T) {
	var texts [31]string
	str := ""
	all := ""
	for i := 0; i < len(texts)-1; i++ {
		texts[i] = str + "\n"
		all += texts[i]
		str += string(rune(i%26 + 'a'))
	}
	texts[len(texts)-1] = all

	for h := 0; h < len(texts); h++ {
		text := texts[h]
		for i := 0; i < len(readMakers); i++ {
			for j := 0; j < len(bufreaders); j++ {
				for k := 0; k < len(bufsizes); k++ {
					readmaker := readMakers[i]
					bufreader := bufreaders[j]
					bufsize := bufsizes[k]
					read := readmaker.fn(strings.NewReader(text))
					buf := NewReaderSize(read, bufsize)
					s := bufreader.fn(buf)
					if s != text {
						t.Errorf("reader=%s fn=%s bufsize=%d want=%q got=%q",
							readmaker.name, bufreader.name, bufsize, text, s)
					}
				}
			}
		}
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	return 0, nil
}

func TestZeroReader(t *testing.T) {
	var z zeroReader
	r := NewReader(z)

	c := make(chan error)
	go func() {
		_, err := r.ReadByte()
		c <- err
	}()

	select {
	case err := <-c:
		if err == nil {
			t.Error("error expected")
		} else if err != io.ErrNoProgress {
			t.Error("unexpected error:", err)
		}
	case <-time.After(time.Second):
		t.Error("test timed out (endless loop in ReadByte?)")
	}
}

// A StringReader delivers its data one string segment at a time via Read.
type StringReader struct {
	data []string
	step int
}

func (r *StringReader) Read(p []byte) (n int, err error) {
	if r.step < len(r.data) {
		s := r.data[r.step]
		n = copy(p, s)
		r.step++
	} else {
		err = io.EOF
	}
	return
}

func readRuneSegments(t *testing.T, segments []string) {
	got := ""
	want := strings.Join(segments, "")
	r := NewReader(&StringReader{data: segments})
	for {
		r, _, err := r.ReadRune()
		if err != nil {
			if err != io.EOF {
				return
			}
			break
		}
		got += string(r)
	}
	if got != want {
		t.Errorf("segments=%v got=%s want=%s", segments, got, want)
	}
}

var segmentList = [][]string{
	{},
	{""},
	{"日", "本語"},
	{"\u65e5", "\u672c", "\u8a9e"},
	{"\U000065e5", "\U0000672c", "\U00008a9e"},
	{"\xe6", "\x97\xa5\xe6", "\x9c\xac\xe8\xaa\x9e"},
	{"Hello", ", ", "World", "!"},
	{"Hello", ", ", "", "World", "!"},
}

func TestReadRune(t *testing.T) {
	for _, s := range segmentList {
		readRuneSegments(t, s)
	}
}

func TestUnreadRune(t *testing.T) {
	segments := []string{"Hello, world:", "日本語"}
	r := NewReader(&StringReader{data: segments})
	got := ""
	want := strings.Join(segments, "")
	// Normal execution.
	for {
		r1, _, err := r.ReadRune()
		if err != nil {
			if err != io.EOF {
				t.Error("unexpected error on ReadRune:", err)
			}
			break
		}
		got += string(r1)
		// Put it back and read it again.
		if err = r.UnreadRune(); err != nil {
			t.Fatal("unexpected error on UnreadRune:", err)
		}
		r2, _, err := r.ReadRune()
		if err != nil {
			t.Fatal("unexpected error reading after unreading:", err)
		}
		if r1 != r2 {
			t.Fatalf("incorrect rune after unread: got %c, want %c", r1, r2)
		}
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNoUnreadRuneAfterPeek(t *testing.T) {
	br := NewReader(strings.NewReader("example"))
	br.ReadRune()
	br.Peek(1)
	if err := br.UnreadRune(); err == nil {
		t.Error("UnreadRune didn't fail after Peek")
	}
}

func TestNoUnreadByteAfterPeek(t *testing.T) {
	br := NewReader(strings.NewReader("example"))
	br.ReadByte()
	br.Peek(1)
	if err := br.UnreadByte(); err == nil {
		t.Error("UnreadByte didn't fail after Peek")
	}
}

func TestNoUnreadRuneAfterDiscard(t *testing.T) {
	br := NewReader(strings.NewReader("example"))
	br.ReadRune()
	br.Discard(1)
	if err := br.UnreadRune(); err == nil {
		t.Error("UnreadRune didn't fail after Discard")
	}
}

func TestNoUnreadByteAfterDiscard(t *testing.T) {
	br := NewReader(strings.NewReader("example"))
	br.ReadByte()
	br.Discard(1)
	if err := br.UnreadByte(); err == nil {
		t.Error("UnreadByte didn't fail after Discard")
	}
}

func TestNoUnreadRuneAfterWriteTo(t *testing.T) {
	br := NewReader(strings.NewReader("example"))
	br.WriteTo(io.Discard)
	if err := br.UnreadRune(); err == nil {
		t.Error("UnreadRune didn't fail after WriteTo")
	}
}

func TestNoUnreadByteAfterWriteTo(t *testing.T) {
	br := NewReader(strings.NewReader("example"))
	br.WriteTo(io.Discard)
	if err := br.UnreadByte(); err == nil {
		t.Error("UnreadByte didn't fail after WriteTo")
	}
}

func TestUnreadByte(t *testing.T) {
	segments := []string{"Hello, ", "world"}
	r := NewReader(&StringReader{data: segments})
	got := ""
	want := strings.Join(segments, "")
	// Normal execution.
	for {
		b1, err := r.ReadByte()
		if err != nil {
			if err != io.EOF {
				t.Error("unexpected error on ReadByte:", err)
			}
			break
		}
		got += string(b1)
		// Put it back and read it again.
		if err = r.UnreadByte(); err != nil {
			t.Fatal("unexpected error on UnreadByte:", err)
		}
		b2, err := r.ReadByte()
		if err != nil {
			t.Fatal("unexpected error reading after unreading:", err)
		}
		if b1 != b2 {
			t.Fatalf("incorrect byte after unread: got %q, want %q", b1, b2)
		}
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUnreadByteMultiple(t *testing.T) {
	segments := []string{"Hello, ", "world"}
	data := strings.Join(segments, "")
	for n := 0; n <= len(data); n++ {
		r := NewReader(&StringReader{data: segments})
		// Read n bytes.
		for i := 0; i < n; i++ {
			b, err := r.ReadByte()
			if err != nil {
				t.Fatalf("n = %d: unexpected error on ReadByte: %v", n, err)
			}
			if b != data[i] {
				t.Fatalf("n = %d: incorrect byte returned from ReadByte: got %q, want %q", n, b, data[i])
			}
		}
		// Unread one byte if there is one.
		if n > 0 {
			if err := r.UnreadByte(); err != nil {
				t.Errorf("n = %d: unexpected error on UnreadByte: %v", n, err)
			}
		}
		// Test that we cannot unread any further.
		if err := r.UnreadByte(); err == nil {
			t.Errorf("n = %d: expected error on UnreadByte", n)
		}
	}
}

func TestUnreadByteOthers(t *testing.T) {
	// A list of readers to use in conjunction with UnreadByte.
	var readers = []func(*Reader, byte) ([]byte, error){
		(*Reader).ReadBytes,
		(*Reader).ReadSlice,
		func(r *Reader, delim byte) ([]byte, error) {
			data, err := r.ReadString(delim)
			return []byte(data), err
		},
		// ReadLine doesn't fit the data/pattern easily
		// so we leave it out. It should be covered via
		// the ReadSlice test since ReadLine simply calls
		// ReadSlice, and it's that function that handles
		// the last byte.
	}

	// Try all readers with UnreadByte.
	for rno, read := range readers {
		// Some input data that is longer than the minimum reader buffer size.
		const n = 10
		var buf bytes.Buffer
		for i := 0; i < n; i++ {
			buf.WriteString("abcdefg")
		}

		r := NewReaderSize(&buf, minReadBufferSize)
		readTo := func(delim byte, want string) {
			data, err := read(r, delim)
			if err != nil {
				t.Fatalf("#%d: unexpected error reading to %c: %v", rno, delim, err)
			}
			if got := string(data); got != want {
				t.Fatalf("#%d: got %q, want %q", rno, got, want)
			}
		}

		// Read the data with occasional UnreadByte calls.
		for i := 0; i < n; i++ {
			readTo('d', "abcd")
			for j := 0; j < 3; j++ {
				if err := r.UnreadByte(); err != nil {
					t.Fatalf("#%d: unexpected error on UnreadByte: %v", rno, err)
				}
				readTo('d', "d")
			}
			readTo('g', "efg")
		}

		// All data should have been read.
		_, err := r.ReadByte()
		if err != io.EOF {
			t.Errorf("#%d: got error %v; want EOF", rno, err)
		}
	}
}

// Test that UnreadRune fails if the preceding operation was not a ReadRune.
func TestUnreadRuneError(t *testing.T) {
	buf := make([]byte, 3) // All runes in this test are 3 bytes long
	r := NewReader(&StringReader{data: []string{"日本語日本語日本語"}})
	if r.UnreadRune() == nil {
		t.Error("expected error on UnreadRune from fresh buffer")
	}
	_, _, err := r.ReadRune()
	if err != nil {
		t.Error("unexpected error on ReadRune (1):", err)
	}
	if err = r.UnreadRune(); err != nil {
		t.Error("unexpected error on UnreadRune (1):", err)
	}
	if r.UnreadRune() == nil {
		t.Error("expected error after UnreadRune (1)")
	}
	// Test error after Read.
	_, _, err = r.ReadRune() // reset state
	if err != nil {
		t.Error("unexpected error on ReadRune (2):", err)
	}
	_, err = r.Read(buf)
	if err != nil {
		t.Error("unexpected error on Read (2):", err)
	}
	if r.UnreadRune() == nil {
		t.Error("expected error after Read (2)")
	}
	// Test error after ReadByte.
	_, _, err = r.ReadRune() // reset state
	if err != nil {
		t.Error("unexpected error on ReadRune (2):", err)
	}
	for range buf {
		_, err = r.ReadByte()
		if err != nil {
			t.Error("unexpected error on ReadByte (2):", err)
		}
	}
	if r.UnreadRune() == nil {
		t.Error("expected error after ReadByte")
	}
	// Test error after UnreadByte.
	_, _, err = r.ReadRune() // reset state
	if err != nil {
		t.Error("unexpected error on ReadRune (3):", err)
	}
	_, err = r.ReadByte()
	if err != nil {
		t.Error("unexpected error on ReadByte (3):", err)
	}
	err = r.UnreadByte()
	if err != nil {
		t.Error("unexpected error on UnreadByte (3):", err)
	}
	if r.UnreadRune() == nil {
		t.Error("expected error after UnreadByte (3)")
	}
	// Test error after ReadSlice.
	_, _, err = r.ReadRune() // reset state
	if err != nil {
		t.Error("unexpected error on ReadRune (4):", err)
	}
	_, err = r.ReadSlice(0)
	if err != io.EOF {
		t.Error("unexpected error on ReadSlice (4):", err)
	}
	if r.UnreadRune() == nil {
		t.Error("expected error after ReadSlice (4)")
	}
}

func TestUnreadRuneAtEOF(t *testing.T) {
	// UnreadRune/ReadRune should error at EOF (was a bug; used to panic)
	r := NewReader(strings.NewReader("x"))
	r.ReadRune()
	r.ReadRune()
	r.UnreadRune()
	_, _, err := r.ReadRune()
	if err == nil {
		t.Error("expected error at EOF")
	} else if err != io.EOF {
		t.Error("expected EOF; got", err)
	}
}

func TestReadStringAllocs(t *testing.T) {
	r := strings.NewReader("       foo       foo        42        42        42        42        42        42        42        42       4.2       4.2       4.2       4.2\n")
	buf := NewReader(r)
	allocs := testing.AllocsPerRun(100, func() {
		r.Seek(0, io.SeekStart)
		buf.Reset(r)

		_, err := buf.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 1 {
		t.Errorf("Unexpected number of allocations, got %f, want 1", allocs)
	}
}

// Check that write errors are returned properly.

type errorWriterTest struct {
	n, m   int
	err    error
	expect error
}

func (w errorWriterTest) Write(p []byte) (int, error) {
	return len(p) * w.n / w.m, w.err
}

var errorWriterTests = []errorWriterTest{
	{0, 1, nil, io.ErrShortWrite},
	{1, 2, nil, io.ErrShortWrite},
	{1, 1, nil, nil},
	{0, 1, io.ErrClosedPipe, io.ErrClosedPipe},
	{1, 2, io.ErrClosedPipe, io.ErrClosedPipe},
	{1, 1, io.ErrClosedPipe, io.ErrClosedPipe},
}

func TestNewReaderSizeIdempotent(t *testing.T) {
	const BufSize = 1000
	b := NewReaderSize(strings.NewReader("hello world"), BufSize)
	// Does it recognize itself?
	b1 := NewReaderSize(b, BufSize)
	if b1 != b {
		t.Error("NewReaderSize did not detect underlying Reader")
	}
	// Does it wrap if existing buffer is too small?
	b2 := NewReaderSize(b, 2*BufSize)
	if b2 == b {
		t.Error("NewReaderSize did not enlarge buffer")
	}
}

type teststringwriter struct {
	write       string
	writeString string
}

func (w *teststringwriter) Write(b []byte) (int, error) {
	w.write += string(b)
	return len(b), nil
}

func (w *teststringwriter) WriteString(s string) (int, error) {
	w.writeString += s
	return len(s), nil
}

func (w *teststringwriter) check(t *testing.T, write, writeString string) {
	t.Helper()
	if w.write != write {
		t.Errorf("write: expected %q, got %q", write, w.write)
	}
	if w.writeString != writeString {
		t.Errorf("writeString: expected %q, got %q", writeString, w.writeString)
	}
}

func TestBufferFull(t *testing.T) {
	const longString = "And now, hello, world! It is the time for all good men to come to the aid of their party"
	buf := NewReaderSize(strings.NewReader(longString), minReadBufferSize)
	line, err := buf.ReadSlice('!')
	if string(line) != "And now, hello, " || err != ErrBufferFull {
		t.Errorf("first ReadSlice(,) = %q, %v", line, err)
	}
	line, err = buf.ReadSlice('!')
	if string(line) != "world!" || err != nil {
		t.Errorf("second ReadSlice(,) = %q, %v", line, err)
	}
}

func TestPeek(t *testing.T) {
	p := make([]byte, 10)
	// string is 16 (minReadBufferSize) long.
	buf := NewReaderSize(strings.NewReader("abcdefghijklmnop"), minReadBufferSize)
	if s, err := buf.Peek(1); string(s) != "a" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "a", string(s), err)
	}
	if s, err := buf.Peek(4); string(s) != "abcd" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "abcd", string(s), err)
	}
	if _, err := buf.Peek(-1); err != ErrNegativeCount {
		t.Fatalf("want ErrNegativeCount got %v", err)
	}
	if s, err := buf.Peek(32); string(s) != "abcdefghijklmnop" || err != ErrBufferFull {
		t.Fatalf("want %q, ErrBufFull got %q, err=%v", "abcdefghijklmnop", string(s), err)
	}
	if _, err := buf.Read(p[0:3]); string(p[0:3]) != "abc" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "abc", string(p[0:3]), err)
	}
	if s, err := buf.Peek(1); string(s) != "d" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "d", string(s), err)
	}
	if s, err := buf.Peek(2); string(s) != "de" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "de", string(s), err)
	}
	if _, err := buf.Read(p[0:3]); string(p[0:3]) != "def" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "def", string(p[0:3]), err)
	}
	if s, err := buf.Peek(4); string(s) != "ghij" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "ghij", string(s), err)
	}
	if _, err := buf.Read(p[0:]); string(p[0:]) != "ghijklmnop" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "ghijklmnop", string(p[0:minReadBufferSize]), err)
	}
	if s, err := buf.Peek(0); string(s) != "" || err != nil {
		t.Fatalf("want %q got %q, err=%v", "", string(s), err)
	}
	if _, err := buf.Peek(1); err != io.EOF {
		t.Fatalf("want EOF got %v", err)
	}

	// Test for issue 3022, not exposing a reader's error on a successful Peek.
	buf = NewReaderSize(dataAndEOFReader("abcd"), 32)
	if s, err := buf.Peek(2); string(s) != "ab" || err != nil {
		t.Errorf(`Peek(2) on "abcd", EOF = %q, %v; want "ab", nil`, string(s), err)
	}
	if s, err := buf.Peek(4); string(s) != "abcd" || err != nil {
		t.Errorf(`Peek(4) on "abcd", EOF = %q, %v; want "abcd", nil`, string(s), err)
	}
	if n, err := buf.Read(p[0:5]); string(p[0:n]) != "abcd" || err != nil {
		t.Fatalf("Read after peek = %q, %v; want abcd, EOF", p[0:n], err)
	}
	if n, err := buf.Read(p[0:1]); string(p[0:n]) != "" || err != io.EOF {
		t.Fatalf(`second Read after peek = %q, %v; want "", EOF`, p[0:n], err)
	}
}

type dataAndEOFReader string

func (r dataAndEOFReader) Read(p []byte) (int, error) {
	return copy(p, r), io.EOF
}

func TestPeekThenUnreadRune(t *testing.T) {
	// This sequence used to cause a crash.
	r := NewReader(strings.NewReader("x"))
	r.ReadRune()
	r.Peek(1)
	r.UnreadRune()
	r.ReadRune() // Used to panic here
}

var testOutput = []byte("0123456789abcdefghijklmnopqrstuvwxy")
var testInput = []byte("012\n345\n678\n9ab\ncde\nfgh\nijk\nlmn\nopq\nrst\nuvw\nxy")
var testInputrn = []byte("012\r\n345\r\n678\r\n9ab\r\ncde\r\nfgh\r\nijk\r\nlmn\r\nopq\r\nrst\r\nuvw\r\nxy\r\n\n\r\n")

// TestReader wraps a []byte and returns reads of a specific length.
type testReader struct {
	data   []byte
	stride int
}

func (t *testReader) Read(buf []byte) (n int, err error) {
	n = t.stride
	if n > len(t.data) {
		n = len(t.data)
	}
	if n > len(buf) {
		n = len(buf)
	}
	copy(buf, t.data)
	t.data = t.data[n:]
	if len(t.data) == 0 {
		err = io.EOF
	}
	return
}

func testReadLine(t *testing.T, input []byte) {
	//for stride := 1; stride < len(input); stride++ {
	for stride := 1; stride < 2; stride++ {
		done := 0
		reader := testReader{input, stride}
		l := NewReaderSize(&reader, len(input)+1)
		for {
			line, isPrefix, err := l.ReadLine()
			if len(line) > 0 && err != nil {
				t.Errorf("ReadLine returned both data and error: %s", err)
			}
			if isPrefix {
				t.Errorf("ReadLine returned prefix")
			}
			if err != nil {
				if err != io.EOF {
					t.Fatalf("Got unknown error: %s", err)
				}
				break
			}
			if want := testOutput[done : done+len(line)]; !bytes.Equal(want, line) {
				t.Errorf("Bad line at stride %d: want: %x got: %x", stride, want, line)
			}
			done += len(line)
		}
		if done != len(testOutput) {
			t.Errorf("ReadLine didn't return everything: got: %d, want: %d (stride: %d)", done, len(testOutput), stride)
		}
	}
}

func TestReadLine(t *testing.T) {
	testReadLine(t, testInput)
	testReadLine(t, testInputrn)
}

func TestLineTooLong(t *testing.T) {
	data := make([]byte, 0)
	for i := 0; i < minReadBufferSize*5/2; i++ {
		data = append(data, '0'+byte(i%10))
	}
	buf := bytes.NewReader(data)
	l := NewReaderSize(buf, minReadBufferSize)
	line, isPrefix, err := l.ReadLine()
	if !isPrefix || !bytes.Equal(line, data[:minReadBufferSize]) || err != nil {
		t.Errorf("bad result for first line: got %q want %q %v", line, data[:minReadBufferSize], err)
	}
	data = data[len(line):]
	line, isPrefix, err = l.ReadLine()
	if !isPrefix || !bytes.Equal(line, data[:minReadBufferSize]) || err != nil {
		t.Errorf("bad result for second line: got %q want %q %v", line, data[:minReadBufferSize], err)
	}
	data = data[len(line):]
	line, isPrefix, err = l.ReadLine()
	if isPrefix || !bytes.Equal(line, data[:minReadBufferSize/2]) || err != nil {
		t.Errorf("bad result for third line: got %q want %q %v", line, data[:minReadBufferSize/2], err)
	}
	line, isPrefix, err = l.ReadLine()
	if isPrefix || err == nil {
		t.Errorf("expected no more lines: %x %s", line, err)
	}
}

func TestReadAfterLines(t *testing.T) {
	line1 := "this is line1"
	restData := "this is line2\nthis is line 3\n"
	inbuf := bytes.NewReader([]byte(line1 + "\n" + restData))
	outbuf := new(strings.Builder)
	maxLineLength := len(line1) + len(restData)/2
	l := NewReaderSize(inbuf, maxLineLength)
	line, isPrefix, err := l.ReadLine()
	if isPrefix || err != nil || string(line) != line1 {
		t.Errorf("bad result for first line: isPrefix=%v err=%v line=%q", isPrefix, err, string(line))
	}
	n, err := io.Copy(outbuf, l)
	if int(n) != len(restData) || err != nil {
		t.Errorf("bad result for Read: n=%d err=%v", n, err)
	}
	if outbuf.String() != restData {
		t.Errorf("bad result for Read: got %q; expected %q", outbuf.String(), restData)
	}
}

func TestReadEmptyBuffer(t *testing.T) {
	l := NewReaderSize(new(bytes.Buffer), minReadBufferSize)
	line, isPrefix, err := l.ReadLine()
	if err != io.EOF {
		t.Errorf("expected EOF from ReadLine, got '%s' %t %s", line, isPrefix, err)
	}
}

func TestLinesAfterRead(t *testing.T) {
	l := NewReaderSize(bytes.NewReader([]byte("foo")), minReadBufferSize)
	_, err := io.ReadAll(l)
	if err != nil {
		t.Error(err)
		return
	}

	line, isPrefix, err := l.ReadLine()
	if err != io.EOF {
		t.Errorf("expected EOF from ReadLine, got '%s' %t %s", line, isPrefix, err)
	}
}

func TestReadLineNonNilLineOrError(t *testing.T) {
	r := NewReader(strings.NewReader("line 1\n"))
	for i := 0; i < 2; i++ {
		l, _, err := r.ReadLine()
		if l != nil && err != nil {
			t.Fatalf("on line %d/2; ReadLine=%#v, %v; want non-nil line or Error, but not both",
				i+1, l, err)
		}
	}
}

type readLineResult struct {
	line     []byte
	isPrefix bool
	err      error
}

var readLineNewlinesTests = []struct {
	input  string
	expect []readLineResult
}{
	{"012345678901234\r\n012345678901234\r\n", []readLineResult{
		{[]byte("012345678901234"), true, nil},
		{nil, false, nil},
		{[]byte("012345678901234"), true, nil},
		{nil, false, nil},
		{nil, false, io.EOF},
	}},
	{"0123456789012345\r012345678901234\r", []readLineResult{
		{[]byte("0123456789012345"), true, nil},
		{[]byte("\r012345678901234"), true, nil},
		{[]byte("\r"), false, nil},
		{nil, false, io.EOF},
	}},
}

func TestReadLineNewlines(t *testing.T) {
	for _, e := range readLineNewlinesTests {
		testReadLineNewlines(t, e.input, e.expect)
	}
}

func testReadLineNewlines(t *testing.T, input string, expect []readLineResult) {
	b := NewReaderSize(strings.NewReader(input), minReadBufferSize)
	for i, e := range expect {
		line, isPrefix, err := b.ReadLine()
		if !bytes.Equal(line, e.line) {
			t.Errorf("%q call %d, line == %q, want %q", input, i, line, e.line)
			return
		}
		if isPrefix != e.isPrefix {
			t.Errorf("%q call %d, isPrefix == %v, want %v", input, i, isPrefix, e.isPrefix)
			return
		}
		if err != e.err {
			t.Errorf("%q call %d, err == %v, want %v", input, i, err, e.err)
			return
		}
	}
}

func createTestInput(n int) []byte {
	input := make([]byte, n)
	for i := range input {
		// 101 and 251 are arbitrary prime numbers.
		// The idea is to create an input sequence
		// which doesn't repeat too frequently.
		input[i] = byte(i % 251)
		if i%101 == 0 {
			input[i] ^= byte(i / 101)
		}
	}
	return input
}

func TestReaderWriteTo(t *testing.T) {
	input := createTestInput(8192)
	r := NewReader(onlyReader{bytes.NewReader(input)})
	w := new(bytes.Buffer)
	if n, err := r.WriteTo(w); err != nil || n != int64(len(input)) {
		t.Fatalf("r.WriteTo(w) = %d, %v, want %d, nil", n, err, len(input))
	}

	for i, val := range w.Bytes() {
		if val != input[i] {
			t.Errorf("after write: out[%d] = %#x, want %#x", i, val, input[i])
		}
	}
}

type errorWriterToTest struct {
	rn, wn     int
	rerr, werr error
	expected   error
}

func (r errorWriterToTest) Read(p []byte) (int, error) {
	return len(p) * r.rn, r.rerr
}

func (w errorWriterToTest) Write(p []byte) (int, error) {
	return len(p) * w.wn, w.werr
}

var errorWriterToTests = []errorWriterToTest{
	{1, 0, nil, io.ErrClosedPipe, io.ErrClosedPipe},
	{0, 1, io.ErrClosedPipe, nil, io.ErrClosedPipe},
	{0, 0, io.ErrUnexpectedEOF, io.ErrClosedPipe, io.ErrClosedPipe},
	{0, 1, io.EOF, nil, nil},
}

func TestReaderWriteToErrors(t *testing.T) {
	for i, rw := range errorWriterToTests {
		r := NewReader(rw)
		if _, err := r.WriteTo(rw); err != rw.expected {
			t.Errorf("r.WriteTo(errorWriterToTests[%d]) = _, %v, want _,%v", i, err, rw.expected)
		}
	}
}

type errorReaderFromTest struct {
	rn, wn     int
	rerr, werr error
	expected   error
}

func (r errorReaderFromTest) Read(p []byte) (int, error) {
	return len(p) * r.rn, r.rerr
}

func (w errorReaderFromTest) Write(p []byte) (int, error) {
	return len(p) * w.wn, w.werr
}

var errorReaderFromTests = []errorReaderFromTest{
	{0, 1, io.EOF, nil, nil},
	{1, 1, io.EOF, nil, nil},
	{0, 1, io.ErrClosedPipe, nil, io.ErrClosedPipe},
	{0, 0, io.ErrClosedPipe, io.ErrShortWrite, io.ErrClosedPipe},
	{1, 0, nil, io.ErrShortWrite, io.ErrShortWrite},
}

// A writeCountingDiscard is like io.Discard and counts the number of times
// Write is called on it.
type writeCountingDiscard int

func (w *writeCountingDiscard) Write(p []byte) (int, error) {
	*w++
	return len(p), nil
}

type negativeReader int

func (r *negativeReader) Read([]byte) (int, error) { return -1, nil }

func TestNegativeRead(t *testing.T) {
	// should panic with a description pointing at the reader, not at itself.
	// (should NOT panic with slice index error, for example.)
	b := NewReader(new(negativeReader))
	defer func() {
		switch err := recover().(type) {
		case nil:
			t.Fatal("read did not panic")
		case error:
			if !strings.Contains(err.Error(), "reader returned negative count from Read") {
				t.Fatalf("wrong panic: %v", err)
			}
		default:
			t.Fatalf("unexpected panic value: %T(%v)", err, err)
		}
	}()
	b.Read(make([]byte, 100))
}

var errFake = errors.New("fake error")

type errorThenGoodReader struct {
	didErr bool
	nread  int
}

func (r *errorThenGoodReader) Read(p []byte) (int, error) {
	r.nread++
	if !r.didErr {
		r.didErr = true
		return 0, errFake
	}
	return len(p), nil
}

func TestReaderClearError(t *testing.T) {
	r := &errorThenGoodReader{}
	b := NewReader(r)
	buf := make([]byte, 1)
	if _, err := b.Read(nil); err != nil {
		t.Fatalf("1st nil Read = %v; want nil", err)
	}
	if _, err := b.Read(buf); err != errFake {
		t.Fatalf("1st Read = %v; want errFake", err)
	}
	if _, err := b.Read(nil); err != nil {
		t.Fatalf("2nd nil Read = %v; want nil", err)
	}
	if _, err := b.Read(buf); err != nil {
		t.Fatalf("3rd Read with buffer = %v; want nil", err)
	}
	if r.nread != 2 {
		t.Errorf("num reads = %d; want 2", r.nread)
	}
}

type emptyThenNonEmptyReader struct {
	r io.Reader
	n int
}

func (r *emptyThenNonEmptyReader) Read(p []byte) (int, error) {
	if r.n <= 0 {
		return r.r.Read(p)
	}
	r.n--
	return 0, nil
}

type readFromWriter struct {
	buf           []byte
	writeBytes    int
	readFromBytes int
}

func (w *readFromWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	w.writeBytes += len(p)
	return len(p), nil
}

func (w *readFromWriter) ReadFrom(r io.Reader) (int64, error) {
	b, err := io.ReadAll(r)
	w.buf = append(w.buf, b...)
	w.readFromBytes += len(b)
	return int64(len(b)), err
}

func TestReadZero(t *testing.T) {
	for _, size := range []int{100, 2} {
		t.Run(fmt.Sprintf("bufsize=%d", size), func(t *testing.T) {
			r := io.MultiReader(strings.NewReader("abc"), &emptyThenNonEmptyReader{r: strings.NewReader("def"), n: 1})
			br := NewReaderSize(r, size)
			want := func(s string, wantErr error) {
				p := make([]byte, 50)
				n, err := br.Read(p)
				if err != wantErr || n != len(s) || string(p[:n]) != s {
					t.Fatalf("read(%d) = %q, %v, want %q, %v", len(p), string(p[:n]), err, s, wantErr)
				}
				t.Logf("read(%d) = %q, %v", len(p), string(p[:n]), err)
			}
			want("abc", nil)
			want("", nil)
			want("def", nil)
			want("", io.EOF)
		})
	}
}

func TestReaderReset(t *testing.T) {
	r := NewReader(strings.NewReader("foo foo"))
	buf := make([]byte, 3)
	r.Read(buf)
	if string(buf) != "foo" {
		t.Errorf("buf = %q; want foo", buf)
	}

	r.Reset(strings.NewReader("bar bar"))
	all, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(all) != "bar bar" {
		t.Errorf("ReadAll = %q; want bar bar", all)
	}

	*r = Reader{} // zero out the Reader
	r.Reset(strings.NewReader("bar bar"))
	all, err = io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(all) != "bar bar" {
		t.Errorf("ReadAll = %q; want bar bar", all)
	}
}

func TestReaderDiscard(t *testing.T) {
	tests := []struct {
		name     string
		r        io.Reader
		bufSize  int // 0 means 16
		peekSize int

		n int // input to Discard

		want    int   // from Discard
		wantErr error // from Discard

		wantBuffered int
	}{
		{
			name:         "normal case",
			r:            strings.NewReader("abcdefghijklmnopqrstuvwxyz"),
			peekSize:     16,
			n:            6,
			want:         6,
			wantBuffered: 10,
		},
		{
			name:         "discard causing read",
			r:            strings.NewReader("abcdefghijklmnopqrstuvwxyz"),
			n:            6,
			want:         6,
			wantBuffered: 10,
		},
		{
			name:         "discard all without peek",
			r:            strings.NewReader("abcdefghijklmnopqrstuvwxyz"),
			n:            26,
			want:         26,
			wantBuffered: 0,
		},
		{
			name:         "discard more than end",
			r:            strings.NewReader("abcdefghijklmnopqrstuvwxyz"),
			n:            27,
			want:         26,
			wantErr:      io.EOF,
			wantBuffered: 0,
		},
		// Any error from filling shouldn't show up until we
		// get past the valid bytes. Here we return 5 valid bytes at the same time
		// as an error, but test that we don't see the error from Discard.
		{
			name: "fill error, discard less",
			r: newScriptedReader(func(p []byte) (n int, err error) {
				if len(p) < 5 {
					panic("unexpected small read")
				}
				return 5, errors.New("5-then-error")
			}),
			n:            4,
			want:         4,
			wantErr:      nil,
			wantBuffered: 1,
		},
		{
			name: "fill error, discard equal",
			r: newScriptedReader(func(p []byte) (n int, err error) {
				if len(p) < 5 {
					panic("unexpected small read")
				}
				return 5, errors.New("5-then-error")
			}),
			n:            5,
			want:         5,
			wantErr:      nil,
			wantBuffered: 0,
		},
		{
			name: "fill error, discard more",
			r: newScriptedReader(func(p []byte) (n int, err error) {
				if len(p) < 5 {
					panic("unexpected small read")
				}
				return 5, errors.New("5-then-error")
			}),
			n:            6,
			want:         5,
			wantErr:      errors.New("5-then-error"),
			wantBuffered: 0,
		},
		// Discard of 0 shouldn't cause a read:
		{
			name:         "discard zero",
			r:            newScriptedReader(), // will panic on Read
			n:            0,
			want:         0,
			wantErr:      nil,
			wantBuffered: 0,
		},
		{
			name:         "discard negative",
			r:            newScriptedReader(), // will panic on Read
			n:            -1,
			want:         0,
			wantErr:      ErrNegativeCount,
			wantBuffered: 0,
		},
	}
	for _, tt := range tests {
		br := NewReaderSize(tt.r, tt.bufSize)
		if tt.peekSize > 0 {
			peekBuf, err := br.Peek(tt.peekSize)
			if err != nil {
				t.Errorf("%s: Peek(%d): %v", tt.name, tt.peekSize, err)
				continue
			}
			if len(peekBuf) != tt.peekSize {
				t.Errorf("%s: len(Peek(%d)) = %v; want %v", tt.name, tt.peekSize, len(peekBuf), tt.peekSize)
				continue
			}
		}
		discarded, err := br.Discard(tt.n)
		if ge, we := fmt.Sprint(err), fmt.Sprint(tt.wantErr); discarded != tt.want || ge != we {
			t.Errorf("%s: Discard(%d) = (%v, %v); want (%v, %v)", tt.name, tt.n, discarded, ge, tt.want, we)
			continue
		}
		if bn := br.Buffered(); bn != tt.wantBuffered {
			t.Errorf("%s: after Discard, Buffered = %d; want %d", tt.name, bn, tt.wantBuffered)
		}
	}

}

func TestReaderSize(t *testing.T) {
	if got, want := NewReader(nil).Size(), DefaultBufSize; got != want {
		t.Errorf("NewReader's Reader.Size = %d; want %d", got, want)
	}
	if got, want := NewReaderSize(nil, 1234).Size(), 1234; got != want {
		t.Errorf("NewReaderSize's Reader.Size = %d; want %d", got, want)
	}
}

// An onlyReader only implements io.Reader, no matter what other methods the underlying implementation may have.
type onlyReader struct {
	io.Reader
}

// An onlyWriter only implements io.Writer, no matter what other methods the underlying implementation may have.
type onlyWriter struct {
	io.Writer
}

// A scriptedReader is an io.Reader that executes its steps sequentially.
type scriptedReader []func(p []byte) (n int, err error)

func (sr *scriptedReader) Read(p []byte) (n int, err error) {
	if len(*sr) == 0 {
		panic("too many Read calls on scripted Reader. No steps remain.")
	}
	step := (*sr)[0]
	*sr = (*sr)[1:]
	return step(p)
}

func newScriptedReader(steps ...func(p []byte) (n int, err error)) io.Reader {
	sr := scriptedReader(steps)
	return &sr
}

// eofReader returns the number of bytes read and io.EOF for the read that consumes the last of the content.
type eofReader struct {
	buf []byte
}

func (r *eofReader) Read(p []byte) (int, error) {
	read := copy(p, r.buf)
	r.buf = r.buf[read:]

	switch read {
	case 0, len(r.buf):
		// As allowed in the documentation, this will return io.EOF
		// in the same call that consumes the last of the data.
		// https://godoc.org/io#Reader
		return read, io.EOF
	}

	return read, nil
}

func TestPartialReadEOF(t *testing.T) {
	src := make([]byte, 10)
	eofR := &eofReader{buf: src}
	r := NewReader(eofR)

	// Start by reading 5 of the 10 available bytes.
	dest := make([]byte, 5)
	read, err := r.Read(dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := len(dest); read != n {
		t.Fatalf("read %d bytes; wanted %d bytes", read, n)
	}

	// The Reader should have buffered all the content from the io.Reader.
	if n := len(eofR.buf); n != 0 {
		t.Fatalf("got %d bytes left in bufio.Reader source; want 0 bytes", n)
	}
	// To prove the point, check that there are still 5 bytes available to read.
	if n := r.Buffered(); n != 5 {
		t.Fatalf("got %d bytes buffered in bufio.Reader; want 5 bytes", n)
	}

	// This is the second read of 0 bytes.
	read, err = r.Read([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if read != 0 {
		t.Fatalf("read %d bytes; want 0 bytes", read)
	}
}

type writerWithReadFromError struct{}

func (w writerWithReadFromError) ReadFrom(r io.Reader) (int64, error) {
	return 0, errors.New("writerWithReadFromError error")
}

func (w writerWithReadFromError) Write(b []byte) (n int, err error) {
	return 10, nil
}
