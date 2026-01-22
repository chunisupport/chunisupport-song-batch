package service

import (
	"fmt"
	"math/rand/v2"
)

// GenerateDisplayID は16文字の16進数の表示IDを生成します。
func GenerateDisplayID() string {
	return fmt.Sprintf("%016x", rand.Uint64())
}
