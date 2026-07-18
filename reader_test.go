package ar

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	t.Parallel()

	r, err := NewReader(openFile(t, "testdata/even_file_sizes.a"))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	expectedFirstHdr := Header{
		Name:    "first_even",
		ModTime: time.Unix(1664113056, 0),
		UID:     501,
		GID:     20,
		Mode:    0o644,
		Size:    16,
	}

	firstHdr, err := r.Next()
	if err != nil {
		t.Fatalf("read first header: %v", err)
	}

	if *firstHdr != expectedFirstHdr {
		t.Fatalf("first header mismatch:\ngot:      %#v\nexpected: %#v",
			firstHdr, expectedFirstHdr)
	}

	firstContent := make([]byte, firstHdr.Size)

	_, err = r.Read(firstContent)
	if err != nil {
		t.Fatalf("read first file: %v", err)
	}

	expectedFirstContent := readFile(t, "testdata/first_even")
	if !bytes.Equal(firstContent, expectedFirstContent) {
		t.Fatalf("first content mismatch: got %q, expected %q",
			string(firstContent), string(expectedFirstContent))
	}

	expectedSecondHdr := Header{
		Name:    "second_even",
		ModTime: time.Unix(1664113074, 0),
		UID:     501,
		GID:     20,
		Mode:    0o644,
		Size:    10,
	}

	secondHdr, err := r.Next()
	if err != nil {
		t.Fatalf("read first header: %v", err)
	}

	if *secondHdr != expectedSecondHdr {
		t.Fatalf("second header mismatch:\ngot:      %#v\nexpected: %#v",
			secondHdr, expectedSecondHdr)
	}

	secondContent := make([]byte, secondHdr.Size)

	_, err = r.Read(secondContent)
	if err != nil {
		t.Fatalf("read first file: %v", err)
	}

	expectedSecondContent := readFile(t, "testdata/second_even")
	if !bytes.Equal(secondContent, expectedSecondContent) {
		t.Fatalf("second content mismatch: got %q, expected %q",
			string(secondContent), string(expectedSecondContent))
	}
}

func TestReaderSkip(t *testing.T) {
	t.Parallel()

	r, err := NewReader(openFile(t, "testdata/even_file_sizes.a"))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	_, err = r.Next()
	if err != nil {
		t.Fatalf("skip first header: %v", err)
	}

	secondHdr, err := r.Next()
	if err != nil {
		t.Fatalf("read second header: %v", err)
	}

	data := make([]byte, secondHdr.Size)

	_, err = r.Read(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	expected := readFile(t, "testdata/second_even")
	if !bytes.Equal(data, expected) {
		t.Fatalf("content mismatch: got %q, expected %q", string(data), string(expected))
	}
}

func TestReaderSkipPadded(t *testing.T) {
	t.Parallel()

	r, err := NewReader(openFile(t, "testdata/uneven_file_sizes.a"))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	_, err = r.Next()
	if err != nil {
		t.Fatalf("skip first header: %v", err)
	}

	secondHdr, err := r.Next()
	if err != nil {
		t.Fatalf("read second header: %v", err)
	}

	data := make([]byte, secondHdr.Size)

	_, err = r.Read(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	expected := readFile(t, "testdata/second_uneven")
	if !bytes.Equal(data, expected) {
		t.Fatalf("content mismatch: got %q, expected %q", string(data), string(expected))
	}
}

func TestReaderPaddingReset(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	w := NewWriter(buf)

	for _, entry := range []struct {
		name string
		data string
	}{
		{name: "odd", data: "1"},
		{name: "even", data: "22"},
		{name: "third", data: "333"},
	} {
		err := w.WriteHeader(&Header{Name: entry.name, Size: int64(len(entry.data))})
		if err != nil {
			t.Fatalf("write header %q: %v", entry.name, err)
		}

		_, err = w.Write([]byte(entry.data))
		if err != nil {
			t.Fatalf("write data %q: %v", entry.name, err)
		}
	}

	err := w.Close()
	if err != nil {
		t.Fatalf("close writer: %v", err)
	}

	r, err := NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	for _, name := range []string{"odd", "even", "third"} {
		hdr, err := r.Next()
		if err != nil {
			t.Fatalf("read header %q: %v", name, err)
		}

		if hdr.Name != name {
			t.Fatalf("got header %q, expected %q", hdr.Name, name)
		}
	}
}

func TestReadBSDNameExtension(t *testing.T) {
	t.Parallel()

	r, err := NewReader(openFile(t, "testdata/long_filename_bsd.a"))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	hdr, err := r.Next()
	if err != nil {
		t.Fatalf("read header with BSD extended filename: %v", err)
	}

	expectedName := "very_long_file_name_that_does_not_fit_into_name_field.txt"

	if hdr.Name != expectedName {
		t.Fatalf("name is %q instead of %q", hdr.Name, expectedName)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	expectedData := "content\n" //nolint
	if !bytes.Equal(data, []byte(expectedData)) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data), expectedData)
	}
}

func TestReadBSDNameExtensionDisabled(t *testing.T) {
	t.Parallel()

	r, err := NewReader(openFile(t, "testdata/long_filename_bsd.a"))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	r.DisableBSDExtensions = true

	hdr, err := r.Next()
	if err != nil {
		t.Fatalf("read header with Gnu extended filename: %v", err)
	}

	expectedName := "#1/60"

	if hdr.Name != expectedName {
		t.Fatalf("name is %q instead of %q", hdr.Name, expectedName)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	expectedData := "very_long_file_name_that_does_not_fit_into_name_field.txt\x00\x00\x00content\n"
	if !bytes.Equal(data, []byte(expectedData)) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data), expectedData)
	}
}

func TestReadGNUNameTablePadding(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}

	err := writeGlobalHeader(buf)
	if err != nil {
		t.Fatalf("write global header: %v", err)
	}

	// The GNU name table has an odd size, so it is followed by a padding byte.
	// Since the reader consumes the name table internally, parseHeader must also
	// consume this padding byte before recursively reading the next header.
	err = writeHeader(buf, &Header{Name: gnuExtendedFormatNameTable, Size: 3})
	if err != nil {
		t.Fatalf("write GNU name table header: %v", err)
	}

	_, err = buf.WriteString("x/\n")
	if err != nil {
		t.Fatalf("write GNU name table: %v", err)
	}

	err = buf.WriteByte(padding)
	if err != nil {
		t.Fatalf("write GNU name table padding: %v", err)
	}

	err = writeHeader(buf, &Header{Name: "/0", Size: 4})
	if err != nil {
		t.Fatalf("write file header: %v", err)
	}

	_, err = buf.WriteString("data")
	if err != nil {
		t.Fatalf("write file content: %v", err)
	}

	r, err := NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	hdr, err := r.Next()
	if err != nil {
		t.Fatalf("read header after GNU name table: %v", err)
	}

	if hdr.Name != "x" {
		t.Fatalf("name is %q instead of %q", hdr.Name, "x")
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	if !bytes.Equal(data, []byte("data")) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data), "data")
	}
}

func TestReadGNUNameExtension(t *testing.T) {
	t.Parallel()

	r, err := NewReader(openFile(t, "testdata/long_filenames_gnu.a"))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	hdr, err := r.Next()
	if err != nil {
		t.Fatalf("read header with GNU extended filename: %v", err)
	}

	expectedName := "very_long_file_name_that_does_not_fit_into_name_field.txt"

	if hdr.Name != expectedName {
		t.Fatalf("name is %q instead of %q", hdr.Name, expectedName)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	expectedData := "content\n"
	if !bytes.Equal(data, []byte(expectedData)) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data), expectedData)
	}

	hdr2, err := r.Next()
	if err != nil {
		t.Fatalf("read header with BSD extended filename: %v", err)
	}

	expectedName2 := "another_very_long_file_name_that_does_not_fit_into_name_field.txt"

	if hdr2.Name != expectedName2 {
		t.Fatalf("name is %q instead of %q", hdr2.Name, expectedName2)
	}

	data2, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	expectedData2 := "data\n"
	if !bytes.Equal(data2, []byte(expectedData2)) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data2), expectedData2)
	}
}

func TestReadGNUNameExtensionDisabled(t *testing.T) {
	t.Parallel()

	r, err := NewReader(openFile(t, "testdata/long_filenames_gnu.a"))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}

	r.DisableGnuExtensions = true

	hdr, err := r.Next()
	if err != nil {
		t.Fatalf("read header with GNU extended filename: %v", err)
	}

	expectedName := "//"

	if hdr.Name != expectedName {
		t.Fatalf("name is %q instead of %q", hdr.Name, expectedName)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	expectedData := ("very_long_file_name_that_does_not_fit_into_name_field.txt/\n" +
		"another_very_long_file_name_that_does_not_fit_into_name_field.txt/\n")

	if !bytes.Equal(data, []byte(expectedData)) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data), expectedData)
	}

	hdr2, err := r.Next()
	if err != nil {
		t.Fatalf("read header with BSD extended filename: %v", err)
	}

	expectedName2 := "/0"

	if hdr2.Name != expectedName2 {
		t.Fatalf("name is %q instead of %q", hdr2.Name, expectedName2)
	}

	data2, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	expectedData2 := "content\n"
	if !bytes.Equal(data2, []byte(expectedData2)) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data2), expectedData2)
	}

	hdr3, err := r.Next()
	if err != nil {
		t.Fatalf("read header with BSD extended filename: %v", err)
	}

	expectedName3 := "/59"

	if hdr3.Name != expectedName3 {
		t.Fatalf("name is %q instead of %q", hdr3.Name, expectedName3)
	}

	data3, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}

	expectedData3 := "data\n"
	if !bytes.Equal(data3, []byte(expectedData3)) {
		t.Fatalf("data mismatch: got %q instead of %q", string(data3), expectedData3)
	}
}

func TestInvalidGlobalHeader(t *testing.T) {
	t.Parallel()

	_, err := NewReader(bytes.NewReader([]byte("non-ar file content")))
	if !errors.Is(err, ErrInvalidGlobalHeader) {
		t.Fatalf("reading non-ar file did not result in %q", ErrInvalidGlobalHeader)
	}
}

func openFile(tb testing.TB, filename string) *os.File {
	tb.Helper()

	f, err := os.Open(filename)
	if err != nil {
		tb.Fatalf("open: %v", err)
	}

	return f
}

func readFile(tb testing.TB, filename string) []byte {
	tb.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		tb.Fatalf("read file: %v", err)
	}

	return data
}
