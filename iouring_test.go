package iouring

import "testing"
import "os"
import "io/ioutil"

func TestReadFile(t *testing.T) {
	buf := make([]byte, 20)
	f, _ := os.Open("test/helloworld.txt")
	read, err := ReadFile(f, buf, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := "Hello World!"
	got := string(buf[:read])
	if want != got {
		t.Fatalf("want: %q, got %q", want, got)
	}
}

func TestWriteFile(t *testing.T) {
	want := "Hello World!"
	file := "test/.helloworld.txt"
	f, _ := os.Create(file)
	written, err := WriteFile(f, []byte(want), 0)
	if err != nil || written != int64(len(want)) {
		t.Fatal(err)
	}
	got, _ := ioutil.ReadFile(file)
	if want != string(got) {
		t.Fatalf("want: %q, got %q", want, got)
	}
}

func TestAppendFile(t *testing.T) {
	str := "Hello World!"
	file := "test/.helloworld.append.txt"
	f, _ := os.Create(file)
	written, err := AppendFile(f, []byte(str))
	if err != nil || written != int64(len(str)) {
		t.Fatal(err)
	}
	written, err = AppendFile(f, []byte(str))
	if err != nil || written != int64(len(str)) {
		t.Fatal(err)
	}
	f.Close()
	got, _ := ioutil.ReadFile(file)
	want := str + str
	if want != string(got) {
		t.Fatalf("want: %q, got %q", want, got)
	}
}
