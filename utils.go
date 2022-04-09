package rabbit

import "crypto/rand"

var charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func genRandomString(n int) string {
	randBytes := make([]byte, n)
	_, _ = rand.Read(randBytes)
	for i, b := range randBytes {
		randBytes[i] = charset[b%byte(len(charset))]
	}

	return string(randBytes)
}
