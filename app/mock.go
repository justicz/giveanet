package main

import (
	"crypto/rand"
	"fmt"
)

func makeFakeStripeSession() (string, error) {
		var tok [16]byte
		_, err := rand.Read(tok[:])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("mn_test_tok_%x", tok), nil
}
