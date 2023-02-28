package benchmarks

import (
	uuidGofrs "github.com/gofrs/uuid"
	uuidGoogle "github.com/google/uuid"
	uuidJakehl "github.com/jakehl/goid"
	uuidRekby "github.com/rekby/fastuuid"
	uuidRogpeppe "github.com/rogpeppe/fastuuid"
	uuidSatori "github.com/satori/go.uuid"
	uuidRwxrob "gitlab.com/rwxrob/uniq"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"
)

func benchThread(n int, f func() string) {
	var s string
	for i := 0; i < n; i++ {
		f()
	}
	_, _ = io.WriteString(io.Discard, s)
}

func bench(b *testing.B, f func() string) {
	b.Run("one-thread", func(b *testing.B) {
		runtime.GC()
		b.ReportAllocs()
		benchThread(b.N, f)
	})
	b.Run("multi-thread", func(b *testing.B) {
		runtime.GC()
		b.ReportAllocs()
		threadCount := runtime.GOMAXPROCS(0)
		b.StopTimer()
		var wg sync.WaitGroup
		start := make(chan bool)

		for i := 0; i < threadCount; i++ {
			wg.Add(1)

			n := b.N / threadCount

			if i == threadCount-1 {
				n += b.N % threadCount
			}

			go func(n int) {
				benchThread(b.N/10, f)
				wg.Done()
			}(n)
		}

		time.Sleep(time.Millisecond)
		b.StartTimer()
		close(start)
		wg.Wait()
	})
}

func BenchmarkRekbyUUID(b *testing.B) {
	bench(b, func() string {
		return uuidRekby.MustUUIDv4String()
	})
}

func BenchmarkGoogleUUID4(b *testing.B) {
	bench(b, func() string {
		return uuidGoogle.NewString()
	})
}

func BenchmarkSatoriUUID4(b *testing.B) {
	bench(b, func() string {
		return uuidSatori.NewV4().String()
	})
}

func BenchmarkGofrs(b *testing.B) {
	bench(b, func() string {
		return uuidGofrs.Must(uuidGofrs.NewV4()).String()
	})
}

var rogpeppeGeneratorMu sync.Mutex
var rogpeppeGenerator = uuidRogpeppe.MustNewGenerator()

func BenchmarkRogpeppeUnsecuredBecauseItCounter(b *testing.B) {
	bench(b, func() string {
		rogpeppeGeneratorMu.Lock()
		res := rogpeppeGenerator.Hex128()
		rogpeppeGeneratorMu.Unlock()
		return res
	})
}

func BenchmarkJakehl(b *testing.B) {
	bench(b, func() string {
		return uuidJakehl.NewV4UUID().String()
	})
}

func BenchmarkRwxrob(b *testing.B) {
	bench(b, func() string {
		return uuidRwxrob.UUID()
	})
}
