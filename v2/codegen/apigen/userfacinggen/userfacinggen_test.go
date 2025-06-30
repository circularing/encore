package userfacinggen

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
		svc := desc.Services[0]
		var svcStruct option.Option[*codegen.VarDecl]
		if fw, ok := svc.Framework.Get(); ok {
			if ss, ok := fw.ServiceStruct.Get(); ok {
				decl := servicestructgen.Gen(gen, svc, ss)
				svcStruct = option.Some(decl)
			}
		}
		Gen(gen, svc, svcStruct)
	}
	codegentest.Run(t, fn)
}
