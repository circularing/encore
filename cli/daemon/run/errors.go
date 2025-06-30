package run

import (
	"errors"

	"github.com/circularing/encore/pkg/errlist"
	"github.com/circularing/encore/v2/internals/perr"
)

func AsErrorList(err error) *errlist.List {
	if errList := errlist.Convert(err); errList != nil {
		return errList
	}

	list := &perr.ListAsErr{}
	if errors.As(err, &list) {
		return &errlist.List{List: list.ErrorList()}
	}
	return nil
}
