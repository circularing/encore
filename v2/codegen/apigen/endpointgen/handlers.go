package endpointgen

import (
	. "github.com/dave/jennifer/jen"

	"github.com/circularing/encore/pkg/option"
	"github.com/circularing/encore/v2/codegen"
	"github.com/circularing/encore/v2/codegen/internal/genutil"
	"github.com/circularing/encore/v2/parser/apis/api"
)

type handlerDesc struct {
	gu        *genutil.Helper
	ep        *api.Endpoint
	svcStruct option.Option[*codegen.VarDecl]

	req  *requestDesc
	resp *responseDesc
	desc *codegen.VarDecl
}

func (h *handlerDesc) Typed() *Statement {
	ep := h.ep
	if ep.Raw {
		return Nil()
	}

	return Func().Params(
		Id("ctx").Qual("context", "Context"),
		h.req.reqDataExpr().Add(h.req.Type()),
	).Params(h.resp.Type(), Error()).BlockFunc(func(g *Group) {
		// fnExpr is the expression for the function we want to call,
		// either just MyRPCName or svc.MyRPCName if we have a service struct.
		var fnExpr *Statement

		// If we have a service struct, initialize it first.
		if ss, ok := h.svcStruct.Get(); ok && ep.Recv.Present() {
			g.List(Id("svc"), Id("initErr")).Op(":=").Add(ss.Qual()).Dot("Get").Call()
			g.If(Id("initErr").Op("!=").Nil()).Block(
				Return(h.resp.zero(), Id("initErr")),
			)
			fnExpr = Id("svc").Dot(ep.Name)
		} else {
			fnExpr = Id(ep.Name)
		}

		g.Do(func(s *Statement) {
			if ep.Response != nil {
				s.List(Id("resp"), Err())
			} else {
				s.Err()
			}
		}).Op(":=").Add(fnExpr).CallFunc(func(g *Group) {
			g.Id("ctx")
			for _, arg := range h.req.HandlerArgs() {
				g.Add(arg)
			}
		})
		g.If(Err().Op("!=").Nil()).Block(Return(h.resp.zero(), Err()))

		if ep.Response != nil {
			g.Return(Id("resp"), Nil())
		} else {
			g.Return(h.resp.zero(), Nil())
		}
	})
}

func (h *handlerDesc) Raw() *Statement {
	ep := h.ep
	if !ep.Raw {
		return Nil()
	}

	return Func().Params(
		Id("w").Qual("net/http", "ResponseWriter"),
		Id("req").Op("*").Qual("net/http", "Request"),
	).BlockFunc(func(g *Group) {
		// fnExpr is the expression for the function we want to call,
		// either just MyRPCName or svc.MyRPCName if we have a service struct.
		var fnExpr *Statement

		// If we have a service struct, initialize it first.
		if ss, ok := h.svcStruct.Get(); ok && ep.Recv.Present() {
			g.List(Id("svc"), Id("initErr")).Op(":=").Add(ss.Qual()).Dot("Get").Call()
			g.If(Id("initErr").Op("!=").Nil()).Block(
				Qual("encore.dev/beta/errs", "HTTPErrorWithCode").Call(Id("w"), Id("initErr"), Lit(0)),
				Return(),
			)
			fnExpr = Id("svc").Dot(ep.Name)
		} else {
			fnExpr = Id(ep.Name)
		}

		g.Add(fnExpr).Call(Id("w"), Id("req"))
	})
}
