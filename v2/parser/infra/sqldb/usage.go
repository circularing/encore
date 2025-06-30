package sqldb

import (
	"github.com/circularing/encore/v2/parser/resource/usage"
)

type DatabaseUsage struct {
	usage.Base
}

func ResolveDatabaseUsage(data usage.ResolveData, db *Database) usage.Usage {
	return &DatabaseUsage{
		Base: usage.Base{
			File: data.Expr.DeclaredIn(),
			Bind: data.Expr.ResourceBind(),
			Expr: data.Expr,
		},
	}
}
