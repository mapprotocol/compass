package keystore

import (
	"fmt"
	"syscall"

	terminal "golang.org/x/term"
)

// GetPassword prompt user to enter password for encrypted keystore
func GetPassword(msg string) []byte {
	for {
		fmt.Println(msg)
		fmt.Print("> ")
		password, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Printf("invalid input: %s\n", err)
		} else {
			fmt.Printf("\n")
			return password
		}
	}
}
