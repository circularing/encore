package app

import (
	"github.com/circularing/encore/v2/app/apiframework"
	"github.com/circularing/encore/v2/internals/parsectx"
)

func (d *Desc) validateAuthHandlers(pc *parsectx.Context, fw *apiframework.AppDesc) {
	handler, found := fw.AuthHandler.Get()
	if !found {
		return
	}

	// Validate the auth data can be marshalled
	// (the same validation we run on request/response types)
	if authData, found := handler.AuthData.Get(); found {
		d.validateType(pc, handler.Decl.AST.Type.Results.List[1].Type, authData.ToType())
	}
}
