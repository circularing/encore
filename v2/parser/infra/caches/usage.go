package caches

import (
	"github.com/circularing/encore/v2/parser/resource/usage"
)

type KeyspaceUsage struct {
	usage.Base

	Keyspace *Keyspace
}

func ResolveKeyspaceUsage(data usage.ResolveData, keyspace *Keyspace) usage.Usage {
	return &KeyspaceUsage{
		Base: usage.Base{
			File: data.Expr.DeclaredIn(),
			Bind: data.Expr.ResourceBind(),
			Expr: data.Expr,
		},
		Keyspace: keyspace,
	}
}
