package iouring

import (
	"os"
	"testing"
)

func BenchmarkReadFile(b *testing.B) {
	f, _ := os.Open("iouring.go")
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 100)
		for pb.Next() {
			ReadFile(f, buf, 0)
		}
	})
}

func BenchmarkReadAt(b *testing.B) {
	f, _ := os.Open("iouring.go")
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 100)
		for pb.Next() {
			f.ReadAt(buf, 0)
		}
	})
}
