package main

import (
	"crypto/sha256"
	"math/bits"

	"go.uber.org/zap"
)

var (
	log, _ = zap.NewDevelopment()
	sugar  = log.Sugar()
)

func bitsDifferentByte(b1, b2 byte) int {
	return bits.OnesCount8(b1 ^ b2)
}

func countBitsDifferent(h1, h2 *[sha256.Size]byte) int {
	total := 0
	for idx := range h1 {
		diff := bitsDifferentByte(h1[idx], h2[idx])
		sugar.Infof("Comparing '%08b' '%08b' have bit difference of %d", h1[idx], h2[idx], diff)
		total += diff
	}
	return total
}

func main() {
	h1 := sha256.Sum256([]byte("first"))
	h2 := sha256.Sum256([]byte("second"))

	sugar.Infof("sha256(%6s) = %x", "first", h1)
	sugar.Infof("sha256(%6s) = %x", "second", h2)
	sugar.Info("Bits different ", countBitsDifferent(&h1, &h2))
}
