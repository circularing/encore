package servicestructgen

import (
	"testing"

	"github.com/circularing/encore/v2/app"
	"github.com/circularing/encore/v2/codegen"
	"github.com/circularing/encore/v2/codegen/internal/codegentest"
)

func TestCodegen(t *testing.T) {
	fn := func(gen *codegen.Generator, desc *app.Desc) {
		svc := desc.Services[0]
		Gen(gen, svc, svc.Framework.MustGet().ServiceStruct.MustGet())
	}

	codegentest.Run(t, fn)
}
