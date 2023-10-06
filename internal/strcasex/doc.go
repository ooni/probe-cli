// Package strcasex converts strings to various cases.
//
// This package forks https://github.com/iancoleman/strcase at v0.2.0.
//
// See the conversion table below:
//
//	| Function                        | Result             |
//	|---------------------------------|--------------------|
//	| ToSnake(s)                      | any_kind_of_string |
//	| ToScreamingSnake(s)             | ANY_KIND_OF_STRING |
//	| ToKebab(s)                      | any-kind-of-string |
//	| ToScreamingKebab(s)             | ANY-KIND-OF-STRING |
//	| ToDelimited(s, '.')             | any.kind.of.string |
//	| ToScreamingDelimited(s, '.')    | ANY.KIND.OF.STRING |
//	| ToCamel(s)                      | AnyKindOfString    |
//	| ToLowerCamel(s)                 | anyKindOfString    |
package strcasex
