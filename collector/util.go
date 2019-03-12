package collector

import "math/rand"

const idSpace = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandomStr generates a random alphanumeric mixed case string
func RandomStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = idSpace[rand.Intn(len(idSpace))]
	}
	return string(b)
}
