package main

import (
	"crypto/rand"
	"fmt"
)

func randomMAC() string {
	b := make([]byte, 3)
	rand.Read(b)

	// s := []string{"52", "54", "00"}

	// for _, e := range strings.Fields(fmt.Sprintf("% 02x\n", b)) {
	// 	s = append(s, e)
	// }

	// return strings.Join(s, ":")

	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", b[0], b[1], b[2])
}

func main() {
	fmt.Println(randomMAC())
}
