package servicestructgen

import (
	. "github.com/dave/jennifer/jen"

	"github.com/circularing/encore/pkg/option"
	"github.com/circularing/encore/v2/app"
	"github.com/circularing/encore/v2/codegen"
	"github.com/circularing/encore/v2/internals/schema"
	"github.com/circularing/encore/v2/parser/apis/servicestruct"
)

func Gen(gen *codegen.Generator, svc *app.Service, s *servicestruct.ServiceStruct) *codegen.VarDecl {
	initFuncName := option.Map(s.Init, func(init *schema.FuncDecl) *Statement {
		return Id(init.Name)
	}).GetOrElse(Nil())

	f := gen.File(s.Decl.File.Pkg, "svcstruct")
	decl := f.VarDecl(s.Decl.Name).Value(Op("&").Qual("encore.dev/appruntime/apisdk/service", "Decl").Types(
		Id(s.Decl.Name),
	).Values(Dict{
		Id("Service"):     Lit(svc.Name),
		Id("Name"):        Lit(s.Decl.Name),
		Id("Setup"):       initFuncName,
		Id("SetupDefLoc"): Lit(gen.TraceNodes.SvcStruct(s)),
	}))

	f.Jen.Func().Id("init").Params().Block(
		Qual("encore.dev/appruntime/apisdk/service", "Register").Call(decl.Qual()),
	)

	return decl
}
