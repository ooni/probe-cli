package main

import (
	"fmt"
	"reflect"
)

// TypeName returns v's package-qualified type name.
func (d *Descriptor) TypeName(v interface{}) string {
	return reflect.TypeOf(v).String()
}

// RequestTypeName calls d.TypeName(d.Request).
func (d *Descriptor) RequestTypeName() string {
	return d.TypeName(d.Request)
}

// ResponseTypeName calls d.TypeName(d.Response).
func (d *Descriptor) ResponseTypeName() string {
	return d.TypeName(d.Response)
}

// APIStructName returns the correct struct type name
// for the API we're currently processing.
func (d *Descriptor) APIStructName() string {
	return fmt.Sprintf("simple%sAPI", d.Name)
}

// FakeAPIStructName returns the correct struct type name
// for the fake for the API we're currently processing.
func (d *Descriptor) FakeAPIStructName() string {
	return fmt.Sprintf("Fake%sAPI", d.Name)
}

// WithLoginAPIStructName returns the correct struct type name
// for the WithLoginAPI we're currently processing.
func (d *Descriptor) WithLoginAPIStructName() string {
	return fmt.Sprintf("withLogin%sAPI", d.Name)
}

// CallerInterfaceName returns the correct caller interface name
// for the API we're currently processing.
func (d *Descriptor) CallerInterfaceName() string {
	return fmt.Sprintf("callerFor%sAPI", d.Name)
}

// ClonerInterfaceName returns the correct cloner interface name
// for the API we're currently processing.
func (d *Descriptor) ClonerInterfaceName() string {
	return fmt.Sprintf("clonerFor%sAPI", d.Name)
}

// WithCacheAPIStructName returns the correct struct type name for
// the cache for the API we're currently processing.
func (d *Descriptor) WithCacheAPIStructName() string {
	return fmt.Sprintf("withCache%sAPI", d.Name)
}

// CacheEntryName returns the correct struct type name for the
// cache entry for the API we're currently processing.
func (d *Descriptor) CacheEntryName() string {
	return fmt.Sprintf("cacheEntryFor%sAPI", d.Name)
}

// CacheKey returns the correct cache key for the API
// we're currently processing.
func (d *Descriptor) CacheKey() string {
	return fmt.Sprintf("%s.cache", d.Name)
}

// StructFields returns all the struct fields of in. This function
// assumes that in is a pointer to struct, and will otherwise panic.
func (d *Descriptor) StructFields(in interface{}) []*reflect.StructField {
	t := reflect.TypeOf(in)
	if t.Kind() != reflect.Ptr {
		panic("not a pointer")
	}
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		panic("not a struct")
	}
	var out []*reflect.StructField
	for idx := 0; idx < t.NumField(); idx++ {
		f := t.Field(idx)
		out = append(out, &f)
	}
	return out
}

// StructFieldsWithTag returns all the struct fields of
// in that have the specified tag.
func (d *Descriptor) StructFieldsWithTag(in interface{}, tag string) []*reflect.StructField {
	var out []*reflect.StructField
	for _, f := range d.StructFields(in) {
		if f.Tag.Get(tag) != "" {
			out = append(out, f)
		}
	}
	return out
}

// RequestOrResponseTypeKind returns the type kind of in, which should
// be a request or a response. This function assumes that in is either a
// pointer to struct or a map and will panic otherwise.
func (d *Descriptor) RequestOrResponseTypeKind(in interface{}) reflect.Kind {
	t := reflect.TypeOf(in)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		if t.Kind() != reflect.Struct {
			panic("not a struct")
		}
		return reflect.Struct
	}
	if t.Kind() != reflect.Map {
		panic("not a map")
	}
	return reflect.Map
}

// RequestTypeKind calls d.RequestOrResponseTypeKind(d.Request).
func (d *Descriptor) RequestTypeKind() reflect.Kind {
	return d.RequestOrResponseTypeKind(d.Request)
}

// ResponseTypeKind calls d.RequestOrResponseTypeKind(d.Response).
func (d *Descriptor) ResponseTypeKind() reflect.Kind {
	return d.RequestOrResponseTypeKind(d.Response)
}

// TypeNameAsStruct assumes that in is a pointer to struct and
// returns the type of the corresponding struct. The returned
// type is package qualified.
func (d *Descriptor) TypeNameAsStruct(in interface{}) string {
	t := reflect.TypeOf(in)
	if t.Kind() != reflect.Ptr {
		panic("not a pointer")
	}
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		panic("not a struct")
	}
	return t.String()
}

// RequestTypeNameAsStruct calls d.TypeNameAsStruct(d.Request)
func (d *Descriptor) RequestTypeNameAsStruct() string {
	return d.TypeNameAsStruct(d.Request)
}

// ResponseTypeNameAsStruct calls d.TypeNameAsStruct(d.Response)
func (d *Descriptor) ResponseTypeNameAsStruct() string {
	return d.TypeNameAsStruct(d.Response)
}
