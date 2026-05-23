package discover

import (
	"fmt"
	"testing"

	"github.com/piyushmishra318/wingman/internal/model"
)

func BenchmarkMerge_large(b *testing.B) {
	winget := make([]model.Package, 200)
	choco := make([]model.Package, 100)
	npm := make([]model.Package, 50)
	for i := range winget {
		winget[i] = model.Package{
			Name: fmt.Sprintf("App-%d", i), ID: fmt.Sprintf("id-%d", i),
			Source: model.SourceWinget, Status: model.StatusUpgrade,
		}
	}
	for i := range choco {
		choco[i] = model.Package{
			Name: fmt.Sprintf("dup-%d", i%50), ID: fmt.Sprintf("choco-%d", i),
			Source: model.SourceChoco, Status: model.StatusUpgrade,
		}
	}
	for i := range npm {
		npm[i] = model.Package{
			Name: fmt.Sprintf("npm-%d", i), ID: fmt.Sprintf("npm-%d", i),
			Source: model.SourceNPM, Status: model.StatusUpgrade,
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Merge(winget, choco, npm)
	}
}
