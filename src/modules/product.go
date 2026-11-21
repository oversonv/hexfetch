//go:build linux

package modules

import (
	"strings"
)

type product struct{}

func init() { Register(product{}) }

func (product) Name() string { return "product" }

func (product) Run(ctx Context) ([]Entry, error) {
	vendor := dmiValue("sys_vendor")
	name := dmiValue("product_name")
	value := strings.TrimSpace(vendor + " " + name)
	if value == "" {
		return nil, Unsupported("product information unavailable")
	}
	return Single(ctx, "Product", value), nil
}
