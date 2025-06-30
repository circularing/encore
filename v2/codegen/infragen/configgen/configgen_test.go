package configgen_test

import (
	"testing"

	"github.com/circularing/encore/v2/app"
	"github.com/circularing/encore/v2/codegen"
	"github.com/circularing/encore/v2/codegen/infragen"
	"github.com/circularing/encore/v2/codegen/internal/codegentest"
)

func TestCodegen(t *testing.T) {
	fn := func(gen *codegen.Generator, desc *app.Desc) {
		infragen.Process(gen, desc)
	}

	codegentest.Run(t, fn)
}
