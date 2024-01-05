package isal

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime"
	"testing"
)

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

const (
	inputSize = 10_000_000
)

func TestCompressCopy(t *testing.T) {
	b := bytes.Repeat([]byte("A"), inputSize)
	in := bytes.NewBuffer(b)
	out := bytes.NewBuffer(make([]byte, inputSize))
	z, _ := NewWriter(out)
	n, err := io.Copy(z, in)
	if err != nil {
		t.Fatalf("Testfail: %v len(b):%d n:%d", err, len(b), n)
	}
}

func TestCompressVerify(t *testing.T) {
	b := bytes.Repeat([]byte("A"), inputSize)
	cbuf := new(bytes.Buffer) // output for compression
	dbuf := new(bytes.Buffer) // output for decompression

	z, err := NewWriter(cbuf)
	if err != nil {
		t.Fatal("Testfail:", err)
	}

	n, err := z.Write(b)
	if err != nil {
		t.Fatal("Testfail:", err)
	}

	if n != len(b) {
		t.Fatalf("Testfail: short write len(b):%d n:%d\n", len(b), n)
	}

	z.Close()

	g, err := gzip.NewReader(cbuf)
	if err != nil {
		t.Fatal("Testfail:", err)
	}

	nw, err := io.Copy(dbuf, g)
	if nw != int64(len(b)) {
		t.Fatalf("Testfail: length mismatch len(b):%d n:%d\n", len(b), n)
	}

	if err != nil {
		t.Fatal("Testfail:", err)
	}

	if !bytes.Equal(b, dbuf.Bytes()) {
		t.Fatal("Testfail: mismatch between compress in and compress out", n)
	}

	g.Close()
	fmt.Printf("Finished compress verify test\n")
}

func TestDeflateInflate(t *testing.T) {

	var b bytes.Buffer
	w, _ := gzip.NewWriterLevel(&b, 5)
	w.Write(textE)
	w.Close()

	var b1 bytes.Buffer
	z, _ := NewWriterLevel(&b1, 2)
	z.Write(textTwain)
	z.Close()

	buf1 := make([]byte, 1024*1024)
	r, _ := NewReader(bytes.NewReader(b.Bytes()))
	n5, _ := r.Read(buf1)

	fmt.Printf("%d n5\n", n5)
	r.Close()

	n4, _ := os.ReadFile("e.txt")

	fmt.Printf("%d Digits %d CompressedDigits, %d CompressedTwain %d returnd digit \n", len(n4), b.Len(), b1.Len(), n5)
	if !(bytes.Equal(n4, buf1[:n5])) {
		t.Errorf("files not same\n")
	}

	buf2 := make([]byte, 8*1024*1024)
	q, _ := NewReader(bytes.NewReader(b1.Bytes()))
	n2, _ := q.Read(buf2)

	q.Close()

	n3, _ := os.ReadFile("mt.txt")

	fmt.Printf("%d Digits %d CompressedDigits, %d Twain %d CompressedTwain %d returnd digit %d returned twian\n", len(n4), b.Len(), len(n3), b1.Len(), n5, n2)
	if !(bytes.Equal(n3, buf2[:n2])) {
		t.Errorf("files not same\n")
	}

}

func TestRawReadSilesia(t *testing.T) {
	fileName := "Isaac.Newton-Opticks.txt"
	inBuf := new(bytes.Buffer)
	zoutBuf := new(bytes.Buffer)
	nseg := 0

	fin, err := os.OpenFile(fileName, os.O_RDONLY, 0755)
	repeatReader := newRepeatReader(fin, 500)
	if err != nil {
		log.Fatalln(err)
	}
	// copy into a buffer
	nfr, err := io.Copy(inBuf, repeatReader)

	if err != nil {
		t.Fatal("could read input buffer:", err)
	}

	fin.Close()

	outBytesBuf := make([]byte, nfr)
	outBytesSegBuf := make([]byte, 210*1024*1024)
	testBytesBuf := make([]byte, nfr)
	copy(testBytesBuf, inBuf.Bytes())

	gw := gzip.NewWriter(zoutBuf)
	if err != nil {
		t.Fatal(err)
	}

	// compress input buffer
	nw, err := io.Copy(gw, inBuf)

	if err != nil {
		t.Fatal(err)
	}

	gw.Close()

	t.Log("compressed_size", nw, "input:", nfr, "output:", zoutBuf.Len())

	// decompress compressed buffer
	//zr, err := gzip.NewReader(zoutBuf)
	zr, err := NewReader(zoutBuf)

	if err != nil {
		t.Fatal("could read input buffer:", err)
	}

	var totalRead int

	for {
		nr, err := zr.Read(outBytesSegBuf)
		nseg++
		nc := copy(outBytesBuf[totalRead:], outBytesSegBuf[:nr])
		totalRead += nc
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}
	t.Log("original:", nfr, "total read", totalRead, "segments", nseg)

	if int64(totalRead) != nfr {
		t.Fatalf("output mismatch file size = %d != Read() = %d\n", nfr, totalRead)
	}

	if !bytes.Equal(outBytesBuf, testBytesBuf) {
		t.Fatalf("output mismatch file decompressed and original\n")
	}

	zr.Close()
	fmt.Printf("finished read file test\n")
}

func runStringCompressTest(str string, t *testing.T) {
	b := new(bytes.Buffer)

	z, _ := NewWriterLevel(b, 1)

	z.Write([]byte(str))
	err := z.Close()
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize Writer '%v'", err)
	}

	// validate with compress/flate
	g, _ := gzip.NewReader(b)
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize compress/flate '%v'", err)
	}

	s := new(bytes.Buffer)
	n, err := io.Copy(s, g)

	if err != nil {
		t.Errorf("TestFail: error failed to do io.Copy '%v'", err)
	}

	if s.String() != str {
		t.Errorf("mismatch\n***expected***\n%q:%d bytes\n\n ***received***\n%q:%d", str, len(str), s, n)
	}

	fmt.Printf("Finish StringCompress Test\n")

}

func runStringDecompressTest(str string, t *testing.T) {
	b := new(bytes.Buffer)

	// validate with compress/gzip
	g, _ := gzip.NewWriterLevel(b, 1)

	g.Write([]byte(str))
	err := g.Close()
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize compress/flate: '%v'", err)
	}

	z, _ := NewReader(b)
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize Reader '%v'", err)
	}
	s := make([]byte, 10*1024)
	n, _ := z.Read(s)

	if string(s[:n]) != str {
		t.Errorf("mismatch\n***expected***\n%q:%d bytes\n\n ***received***\n%q:%d", str, len(str), s, n)
	}

	fmt.Printf("Finish StringDecompress Test\n")

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

type RepeatReader struct {
	reader      io.Reader
	original    io.Reader
	repeatCount int
	current     int
}

// NewRepeatReader creates a new RepeatReader.
func newRepeatReader(reader io.Reader, repeatCount int) *RepeatReader {
	return &RepeatReader{
		reader:      reader,
		original:    reader,
		repeatCount: repeatCount,
		current:     0,
	}
}

// Read implements the io.Reader interface for RepeatReader.
func (r *RepeatReader) Read(p []byte) (int, error) {
	if r.current >= r.repeatCount {
		return 0, io.EOF // End of all repeats
	}

	n, err := r.reader.Read(p)
	if err != nil {
		if err == io.EOF && r.current < r.repeatCount-1 {
			r.current++
			r.reader = r.original // Reset the reader to the start for the next repeat
			return r.Read(p)      // Continue reading on next repeat
		}
		return n, err
	}

	return n, nil
}

func TestLong(t *testing.T) {
	runtime.GC()
	fileName := "Isaac.Newton-Opticks.txt"
	inBuf := new(bytes.Buffer)
	outBuf := new(bytes.Buffer)
	//outBuf := bytes.NewBuffer(make([]byte,0,128*2048*1024))
	zoutBuf := new(bytes.Buffer)

	fmt.Printf(" Mark Test\n")
	fin, err := os.OpenFile(fileName, os.O_RDONLY, 0755)
	repeatReader := newRepeatReader(fin, 500)
	if err != nil {
		log.Fatalln(err)
	}
	// copy into a buffer
	nfr, err := io.Copy(inBuf, repeatReader)

	if err != nil {
		t.Fatal("could read input buffer:", err)
	}

	fin.Close()

	zw, err := gzip.NewWriterLevel(zoutBuf, 6)
	if err != nil {
		t.Fatal(err)
	}

	// compress input buffer
	nw, err := io.Copy(zw, inBuf)

	if err != nil {
		t.Fatal(err)
	}

	zw.Close()

	t.Log("compressed_size", nw, "input:", nfr, "output:", zoutBuf.Len())

	// decompress compressed buffer
	gr, _ := NewReader(zoutBuf)

	if err != nil {
		t.Fatal("could read input buffer:", err)
	}

	nr, err := io.Copy(outBuf, gr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("original:", nfr, "nr", nr, "output:", outBuf.Len())

	if int64(outBuf.Len()) != nr {
		t.Fatalf("output mismatch nr = %d != outBuf.Len() = %d\n", nr, outBuf.Len())
	}

	if int64(outBuf.Len()) != nfr {
		t.Fatalf("output mismatch file size = %d != outBuf.Len() = %d\n", nfr, outBuf.Len())
	}

	gr.Close()

	fmt.Printf("Finish Mark test\n")
}

func testDeflate(t *testing.T, r *rand.Rand, l int, src []byte) {
	orgSrc := src
	out := bytes.Buffer{}
	zout, _ := gzip.NewWriterLevel(&out, l)
	_, _ = zout.Write(src)
	zout.Close()

	got := bytes.Buffer{}
	zin, _ := NewReader(&out)
	_, _ = io.Copy(&got, zin)
	if !bytes.Equal(got.Bytes(), orgSrc) {
		t.Fatal("fail")
	}
	zin.Close()
}


func TestDeflateRandom(t *testing.T) {
	for iter := 0; iter < 25; iter++ {
		for l := 1; l < 7; l++ {
			i, level := iter, l // Create a local copy of iter and l
			t.Run(fmt.Sprintf("%d/%d", i, level), func(t *testing.T) {
				t.Parallel()
				r := rand.New(rand.NewSource(int64(i)))
				n := r.Intn(16 << 20)
				data := make([]byte, n)
				_, _ = r.Read(data)
				testDeflate(t, r, level, data) // Use the local copy
			})
		}
	}
}

func testInflate(t *testing.T, r *rand.Rand, l int, src []byte) {
	orgSrc := src
	out := bytes.Buffer{}
	zout, _ := NewWriterLevel(&out, l)
	_, _ = zout.Write(src)
	zout.Close()

	got := bytes.Buffer{}
	zin, _ := gzip.NewReader(&out)
	_, _ = io.Copy(&got, zin)
	if !bytes.Equal(got.Bytes(), orgSrc) {
		t.Fatal("fail")
	}
	zin.Close()
}

func TestInflateRandom(t *testing.T) {
	for iter := 0; iter < 25; iter++ {
		for l := 1; l < 3; l++ {
			i, level := iter, l
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				t.Parallel()
				r := rand.New(rand.NewSource(int64(i)))
				n := r.Intn(16 << 20)
				data := make([]byte, n)
				_, _ = r.Read(data)
				testInflate(t, r, level, data)
			})
		}
	}
}

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

func BenchmarkDecodeISAL(b *testing.B) {
	runtime.GC()
	doBench(b, func(b *testing.B, buf0 []byte, level, n int) {
		b.ReportAllocs()
		b.StopTimer()

		compressed := new(bytes.Buffer)
		w, err := NewWriterLevel(compressed, level)
		if err != nil {
			b.Fatal(err)
		}
		w.Write(buf0)

		b.SetBytes(int64(len(buf0)))
		w.Close()

		buf1 := compressed.Bytes()
		buf0, compressed, w = nil, nil, nil
		buf4 := make([]byte, 2*1024*1024)
		runtime.GC()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			br, _ := NewReader(bytes.NewReader(buf1))
			//			io.Copy(ioutil.Discard,br)
			_, _ = br.Read(buf4)

		}
	})
}

func BenchmarkDecodeNative(b *testing.B) {
	doBench(b, func(b *testing.B, buf0 []byte, level, n int) {
		b.ReportAllocs()
		b.StopTimer()

		compressed := new(bytes.Buffer)
		w, err := gzip.NewWriterLevel(compressed, level)
		if err != nil {
			b.Fatal(err)
		}
		w.Write(buf0)
		w.Close()
		b.SetBytes(int64(len(buf0)))

		buf1 := compressed.Bytes()
		buf0, compressed, w = nil, nil, nil
		//buf4 := make([]byte, 2*1024*1024)

		runtime.GC()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			br, _ := gzip.NewReader(bytes.NewReader(buf1))
			io.Copy(ioutil.Discard, br)
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
		w, err := gzip.NewWriterLevel(io.Discard, level)
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

func ioCopyStringDecompressTest(str string, t *testing.T) {
	b := new(bytes.Buffer)

	// validate with compress/gzip
	g, _ := gzip.NewWriterLevel(b, 1)

	g.Write([]byte(str))
	err := g.Close()
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize compress/flate: '%v'", err)
	}

	z, _ := NewReader(b)
	if err != nil {
		t.Fatalf("TestInit: error failed to initialize Reader '%v'", err)
	}
	s := new(bytes.Buffer)

	n, err := io.Copy(s, z)
	if err != nil {
		t.Fatalf("Decompression failed: '%v'", err)
	}

	if s.String() != str {
		t.Errorf("mismatch\n***expected***\n%q:%d bytes\n\n ***received***\n%q:%d", str, len(str), s, n)
	}

	fmt.Printf("Finish StringDecompress Test\n")

}

func TestIoCopyDecompressShortString(t *testing.T) {
	str := string("Hello World\n")
	ioCopyStringDecompressTest(str, t)
}
