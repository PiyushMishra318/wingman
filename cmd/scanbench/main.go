package main

import (
	"fmt"
	"time"

	"github.com/piyushmishra318/wingman/internal/discover"
)

func main() {
	start := time.Now()
	r := discover.All(true)
	elapsed := time.Since(start)
	auto := 0
	for _, p := range r.Packages {
		if p.CanAutoUpdate() {
			auto++
		}
	}
	fmt.Printf("scan %v — %d packages, %d shortcuts, %d auto\n", elapsed.Round(time.Millisecond), len(r.Packages), len(r.Shortcuts), auto)
}
