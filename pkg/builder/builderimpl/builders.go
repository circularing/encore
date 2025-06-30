package builderimpl

import (
	"encore.dev/appruntime/exported/experiments"
	"github.com/circularing/encore/pkg/appfile"
	"github.com/circularing/encore/pkg/builder"
	"github.com/circularing/encore/v2/tsbuilder"
	"github.com/circularing/encore/v2/v2builder"
)

func Resolve(lang appfile.Lang, expSet *experiments.Set) builder.Impl {
	if lang == appfile.LangTS || experiments.TypeScript.Enabled(expSet) {
		return tsbuilder.New()
	}
	return v2builder.BuilderImpl{}
}
