package isal

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"testing"
)

var (
	textTwain, _ = os.ReadFile("mt.txt")
	textE, _     = os.ReadFile("e.txt")
)

var suites = []struct{ name, file string }{
	// Digits is the digits of the irrational number e. Its decimal representation
	// does not repeat, but there are only 10 possible digits, so it should be
	// reasonably compressible.
	{"Digits", "e.txt"},
	{"Twain", "mt.txt"},
	// Newton is Isaac Newtons's educational text on Opticks.
	{"Newton", "Isaac.Newton-Opticks.txt"},
}

var levelTests = []struct {
	name  string
	level int
}{
	{"Level 1", 1},
	{"Level 2", 2},
	{"Level 3", 3},
}

var sizes = []struct {
	name string
	n    int
}{
	{"1e4", 1e4},
	{"1e5", 1e5},
	{"1e6", 1e6},
}

var strGettysBurgAddress = "" +
	"  Four score and seven years ago our fathers brought forth on\n" +
	"this continent, a new nation, conceived in Liberty, and dedicated\n" +
	"to the proposition that all men are created equal.\n" +
	"  Now we are engaged in a great Civil War, testing whether that\n" +
	"nation, or any nation so conceived and so dedicated, can long\n" +
	"endure.\n" +
	"  We are met on a great battle-field of that war.\n" +
	"  We have come to dedicate a portion of that field, as a final\n" +
	"resting place for those who here gave their lives that that\n" +
	"nation might live.  It is altogether fitting and proper that\n" +
	"we should do this.\n" +
	"  But, in a larger sense, we can not dedicate — we can not\n" +
	"consecrate — we can not hallow — this ground.\n" +
	"  The brave men, living and dead, who struggled here, have\n" +
	"consecrated it, far above our poor power to add or detract.\n" +
	"The world will little note, nor long remember what we say here,\n" +
	"but it can never forget what they did here.\n" +
	"  It is for us the living, rather, to be dedicated here to the\n" +
	"unfinished work which they who fought here have thus far so\n" +
	"nobly advanced.  It is rather for us to be here dedicated to\n" +
	"the great task remaining before us — that from these honored\n" +
	"dead we take increased devotion to that cause for which they\n" +
	"gave the last full measure of devotion —\n" +
	"  that we here highly resolve that these dead shall not have\n" +
	"died in vain — that this nation, under God, shall have a new\n" +
	"birth of freedom — and that government of the people, by the\n" +
	"people, for the people, shall not perish from this earth.\n" +
	"\n" +
	"Abraham Lincoln, November 19, 1863, Gettysburg, Pennsylvania\n"

func doBench(b *testing.B, f func(b *testing.B, buf []byte, level, n int)) {
	for _, suite := range suites {
		buf, err := os.ReadFile(suite.file)
		if err != nil {
			b.Fatal(err)
		}
		if len(buf) == 0 {
			b.Fatalf("test file %q has no data", suite.file)
		}
		for _, l := range levelTests {
			for _, s := range sizes {
				b.Run(suite.name+"/"+l.name+"/"+s.name, func(b *testing.B) {
					f(b, buf, l.level, s.n)
				})
			}
		}
	}
}

func runStringCompressTest(str string, t *testing.T) {
	b := new(bytes.Buffer)

	z, _ := NewWriter(b)

	z.Write([]byte(str))
	err := z.Close()
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize Writer '%v'", err)
	}

	/* validate with compress/flate */
	g := flate.NewReader(b)
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize compress/flate '%v'", err)
	}

	s := new(bytes.Buffer)
	n, err := io.Copy(s, g)
	if err != nil {
		t.Fatalf("TestInit: error failed to copy buffer to the validator '%v'", err)
	}

	if s.String() != str {
		t.Errorf("mismatch\n***expected***\n%q:%d bytes\n\n ***received***\n%q:%d", str, len(str), s, n)
	}

}

func runStringDecompressTest(str string, t *testing.T) {
	b := new(bytes.Buffer)

	/* validate with compress/gzip */
	g, _ := flate.NewWriter(b, 1)

	g.Write([]byte(str))
	err := g.Close()
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize compress/flate: '%v'", err)
	}

	z := NewReader(b)
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize Reader '%v'", err)
	}
	s := make([]byte, 10*1024)
	n := z.Read(s)

	if string(s[:n]) != str {
		t.Errorf("mismatch\n***expected***\n%q:%d bytes\n\n ***received***\n%q:%d", str, len(str), s, n)
	}

}

func TestCompressShortString(t *testing.T) {
	str := string("Hello World\n")
	runStringCompressTest(str, t)
}

func TestCompressLongString(t *testing.T) {
	str := string(strGettysBurgAddress)
	runStringCompressTest(str, t)
}

func TestDecompressShortString(t *testing.T) {
	str := string("Hello World\n")
	runStringDecompressTest(str, t)
}

func TestDecompressLongString(t *testing.T) {
	str := string(strGettysBurgAddress)
	runStringDecompressTest(str, t)
}

func Inflate(deflated []byte) []byte {
	var b bytes.Buffer
	r := flate.NewReader(bytes.NewReader(deflated))
	b.ReadFrom(r)
	r.Close()
	return b.Bytes()
}

func TestDeflateInflate(t *testing.T) {

	var b bytes.Buffer
	w, _ := NewWriterLevel(&b, 2)
	w.Write(textE)
	w.Close()

	var b1 bytes.Buffer
	z, _ := NewWriterLevel(&b1, 2)
	z.Write(textTwain)
	z.Close()

	buf := make([]byte, 1024*1024)

	y := NewReader(bytes.NewReader(b1.Bytes()))
	n2 := y.Read(buf)

	buf1 := make([]byte, 1024*1024)
	r := NewReader(bytes.NewReader(b.Bytes()))
	n5 := r.Read(buf1)

	n3, _ := os.ReadFile("mt.txt")
	n4, _ := os.ReadFile("e.txt")

	fmt.Printf("%d Digits %d CompressedDigits, %d Twain %d CompressedTwain\n", len(n4), b.Len(), len(n3), b1.Len())
	if !(bytes.Equal(n3, buf[:n2])) {
		t.Errorf("files not same\n")
	}
	if !(bytes.Equal(n4, buf1[:n5])) {
		t.Errorf("files not same\n")
	}

	var b3, b4 bytes.Buffer
	q, _ := NewWriterLevel(&b3, 2)
	q.Write(textE)
	q.Close()

	s, _ := flate.NewWriter(&b4, 1)
	s.Write(textTwain)
	s.Close()

	buf3 := make([]byte, 1024*1024)

	n2 = NewReader(bytes.NewReader(b4.Bytes())).Read(buf3)

	buf4 := Inflate(b3.Bytes())

	if !(bytes.Equal(n3, buf3[:n2])) {
		t.Errorf("files not same\n")
	}
	if !(bytes.Equal(n4, buf4)) {
		t.Errorf("files not same\n")
	}

}

func testDeflate(t *testing.T, r *rand.Rand, src []byte) {
	orgSrc := src
	out := bytes.Buffer{}
	zout, _ := NewWriterLevel(&out, 2)
	_, _ = zout.Write(src)

	got := bytes.Buffer{}
	zin := flate.NewReader(bytes.NewReader(out.Bytes()))
	_, _ = io.Copy(&got, zin)
	if !bytes.Equal(got.Bytes(), orgSrc) {
		t.Fatal("fail")
	}
}

func TestDeflateRandom(t *testing.T) {
	for iter := 0; iter < 20; iter++ {
		i := iter
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			r := rand.New(rand.NewSource(int64(i)))
			n := r.Intn(16 << 20)
			data := make([]byte, n)
			_, _ = r.Read(data)
			testDeflate(t, r, data)
		})
	}
}

func BenchmarkDecodeISAL(b *testing.B) {
	doBench(b, func(b *testing.B, buf0 []byte, level, n int) {
		b.ReportAllocs()
		b.StopTimer()

		compressed := new(bytes.Buffer)
		w, err := flate.NewWriter(compressed, level)
		if err != nil {
			b.Fatal(err)
		}
		w.Write(buf0)

		b.SetBytes(int64(len(buf0)))
		w.Close()

		buf1 := compressed.Bytes()
		buf0, compressed, w = nil, nil, nil
		buf4 := make([]byte, 1024*1024)
		runtime.GC()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			_ = NewReader(bytes.NewReader(buf1)).Read(buf4)

		}
	})
}

func BenchmarkDecodeNative(b *testing.B) {
	doBench(b, func(b *testing.B, buf0 []byte, level, n int) {
		b.ReportAllocs()
		b.StopTimer()

		compressed := new(bytes.Buffer)
		w, err := flate.NewWriter(compressed, level)
		if err != nil {
			b.Fatal(err)
		}
		w.Write(buf0)
		w.Close()
		b.SetBytes(int64(len(buf0)))

		buf1 := compressed.Bytes()
		buf0, compressed, w = nil, nil, nil
		runtime.GC()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			io.Copy(io.Discard, flate.NewReader(bytes.NewReader(buf1)))
		}
	})
}

func BenchmarkEncodeISAL(b *testing.B) {
	doBench(b, func(b *testing.B, buf0 []byte, level, n int) {
		b.StopTimer()
		b.SetBytes(int64(n))

		buf1 := make([]byte, n)
		for i := 0; i < n; i += len(buf0) {
			if len(buf0) > n-i {
				buf0 = buf0[:n-i]
			}
			copy(buf1[i:], buf0)
		}
		buf0 = nil
		w, err := NewWriterLevel(io.Discard, level)
		if err != nil {
			b.Fatal(err)
		}
		runtime.GC()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			//        w.Reset(io.Discard)
			w.Write(buf1)
			w.Close()
		}
	})
}

func BenchmarkEncodeNative(b *testing.B) {
	doBench(b, func(b *testing.B, buf0 []byte, level, n int) {
		b.StopTimer()
		b.SetBytes(int64(n))

		buf1 := make([]byte, n)
		for i := 0; i < n; i += len(buf0) {
			if len(buf0) > n-i {
				buf0 = buf0[:n-i]
			}
			copy(buf1[i:], buf0)
		}
		buf0 = nil
		w, err := flate.NewWriter(io.Discard, level)
		if err != nil {
			b.Fatal(err)
		}
		runtime.GC()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			w.Reset(io.Discard)
			w.Write(buf1)
			w.Close()
		}
	})
}
