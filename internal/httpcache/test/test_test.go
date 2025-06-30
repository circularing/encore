package test_test

import (
	"testing"

	"github.com/circularing/encore/internal/httpcache"
	"github.com/circularing/encore/internal/httpcache/test"
)

func TestMemoryCache(t *testing.T) {
	test.Cache(t, httpcache.NewMemoryCache())
}
