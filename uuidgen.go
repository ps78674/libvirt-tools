package main

import (
	"crypto/rand"
	"fmt"
)

func randomUUID() string {
	b := make([]byte, 20)
	rand.Read(b)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[5:7], b[8:10], b[11:13], b[14:20])
}

func main() {
	fmt.Println(randomUUID())
}
