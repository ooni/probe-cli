package main

import (
	"fmt"
	"strings"
	"time"
)

func (d *Descriptor) genNewCache(sb *strings.Builder) {
	fmt.Fprintf(sb, "// %s implements caching for %s.\n",
		d.CacheStructName(), d.APIStructName())
	fmt.Fprintf(sb, "type %s struct {\n", d.CacheStructName())
	fmt.Fprintf(sb, "\tAPI %s // mandatory\n", d.CallerInterfaceName())
	fmt.Fprint(sb, "\tGobCodec GobCodec // optional\n")
	fmt.Fprint(sb, "\tKVStore KVStore // mandatory\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "type %s struct {\n", d.CacheEntryName())
	fmt.Fprintf(sb, "\tReq %s\n", d.RequestTypeName())
	fmt.Fprintf(sb, "\tResp %s\n", d.ResponseTypeName())
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (c *%s) Call(ctx context.Context, req %s) (%s, error) {\n",
		d.CacheStructName(), d.RequestTypeName(), d.ResponseTypeName())
	fmt.Fprint(sb, "\tresp, err := c.API.Call(ctx, req)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\tif resp, _ := c.readcache(req); resp != nil {\n")
	fmt.Fprint(sb, "\t\t\treturn resp, nil\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tif err := c.writecache(req, resp); err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn resp, nil\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (c *%s) gobCodec() GobCodec {\n", d.CacheStructName())
	fmt.Fprint(sb, "\tif c.GobCodec != nil {\n")
	fmt.Fprint(sb, "\t\treturn c.GobCodec\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn &defaultGobCodec{}\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (c *%s) getcache() ([]%s, error) {\n",
		d.CacheStructName(), d.CacheEntryName())
	fmt.Fprintf(sb, "\tdata, err := c.KVStore.Get(\"%s\")\n", d.CacheKey())
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprintf(sb, "\tvar out []%s\n", d.CacheEntryName())
	fmt.Fprint(sb, "\tif err := c.gobCodec().Decode(data, &out); err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn out, nil\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (c *%s) setcache(in []%s) error {\n",
		d.CacheStructName(), d.CacheEntryName())
	fmt.Fprint(sb, "\tdata, err := c.gobCodec().Encode(in)\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprintf(sb, "\treturn c.KVStore.Set(\"%s\", data)\n", d.CacheKey())
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (c *%s) readcache(req %s) (%s, error) {\n",
		d.CacheStructName(), d.RequestTypeName(), d.ResponseTypeName())
	fmt.Fprint(sb, "\tcache, err := c.getcache()\n")
	fmt.Fprint(sb, "\tif err != nil {\n")
	fmt.Fprint(sb, "\t\treturn nil, err\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\tfor _, cur := range cache {\n")
	fmt.Fprint(sb, "\t\tif reflect.DeepEqual(req, cur.Req) {\n")
	fmt.Fprint(sb, "\t\t\treturn cur.Resp, nil\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn nil, errCacheNotFound\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "func (c *%s) writecache(req %s, resp %s) error {\n",
		d.CacheStructName(), d.RequestTypeName(), d.ResponseTypeName())
	fmt.Fprint(sb, "\tcache, _ := c.getcache()\n")
	fmt.Fprintf(sb, "\tout := []%s{{Req: req, Resp: resp}}\n", d.CacheEntryName())
	fmt.Fprint(sb, "\tconst toomany = 64\n")
	fmt.Fprint(sb, "\tfor idx, cur := range cache {\n")
	fmt.Fprint(sb, "\t\tif reflect.DeepEqual(req, cur.Req) {\n")
	fmt.Fprint(sb, "\t\t\tcontinue // we already updated the cache\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\tif idx > toomany {\n")
	fmt.Fprint(sb, "\t\t\tbreak\n")
	fmt.Fprint(sb, "\t\t}\n")
	fmt.Fprint(sb, "\t\tout = append(out, cur)\n")
	fmt.Fprint(sb, "\t}\n")
	fmt.Fprint(sb, "\treturn c.setcache(out)\n")
	fmt.Fprint(sb, "}\n\n")

	fmt.Fprintf(sb, "var _ %s = &%s{}\n\n", d.CallerInterfaceName(),
		d.CacheStructName())
}

// GenCachingGo generates caching.go.
func GenCachingGo() {
	var sb strings.Builder
	fmt.Fprint(&sb, "// Code generated by go generate; DO NOT EDIT.\n")
	fmt.Fprintf(&sb, "// %s\n\n", time.Now())
	fmt.Fprint(&sb, "package ooapi\n\n")
	fmt.Fprint(&sb, "//go:generate go run ./internal/generator\n\n")
	fmt.Fprint(&sb, "import (\n")
	fmt.Fprint(&sb, "\t\"context\"\n")
	fmt.Fprint(&sb, "\t\"reflect\"\n")
	fmt.Fprint(&sb, "\n")
	fmt.Fprint(&sb, "\t\"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel\"\n")
	fmt.Fprint(&sb, ")\n")
	for _, desc := range Descriptors {
		if !desc.RequiresCache {
			continue
		}
		desc.genNewCache(&sb)
	}
	writefile("caching.go", &sb)
}
