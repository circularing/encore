package middlewaregen

import (
	"testing"

	"github.com/circularing/encore/pkg/option"
	"github.com/circularing/encore/v2/app"
	"github.com/circularing/encore/v2/codegen"
	"github.com/circularing/encore/v2/codegen/apigen/servicestructgen"
	"github.com/circularing/encore/v2/codegen/internal/codegentest"
)

func TestCodegen(t *testing.T) {
	fn := func(gen *codegen.Generator, desc *app.Desc) {
		Gen(gen, desc.Framework.MustGet().GlobalMiddleware, option.None[*codegen.VarDecl]())
		for _, svc := range desc.Services {
			var svcStruct option.Option[*codegen.VarDecl]
			if fw, ok := svc.Framework.Get(); ok {
				if ss, ok := fw.ServiceStruct.Get(); ok {
					svcStruct = option.Some(servicestructgen.Gen(gen, svc, ss))
				}
			}
			Gen(gen, svc.Framework.MustGet().Middleware, svcStruct)
		}
	}

	codegentest.Run(t, fn)
}
