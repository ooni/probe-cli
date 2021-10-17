// Package getoptx contains getopt extensions.
package getoptx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/iancoleman/strcase"
	"github.com/pborman/getopt/v2"
)

var (
	// errExpectedPtr indicates we expected a pointer.
	errExpectedPtr = errors.New("expected a ptr")

	// errExpectedStructPtr indicates we expected a struct ptr.
	errExpectedStructPtr = errors.New("expected a struct ptr")

	// errNoDocs indicates there's missing documentation.
	errNoDocs = errors.New("missing documentation")
)

// Program describes the program and its options.
type Program interface {
	// AfterParsingChecks implements the required set of
	// checks after successful options parsing.
	AfterParsingChecks(getopt *Parser)

	// DescribeOption returns the description of an option.
	DescribeOption(option string) string

	// PositionalArguments returns the string to print for positional
	// arguments as part of the usage. If this string is empty, it means
	// the command does not take any positional argment.
	PositionalArguments() string

	// ProgramName returns the program name.
	ProgramName() string

	// ShortDescription returns a short description of the program.
	ShortDescription() string

	// ShortOptionName returns the corresponding short option name.
	ShortOptionName(option string) rune
}

// Parser contains a CLI parser.
type Parser struct {
	prog Program
	set  *getopt.Set
}

// NewParser creates a new parser. This function panics on failure.
func NewParser(prog Program) *Parser {
	structValue, err := castToStructPtr(prog)
	if err != nil {
		panic(err)
	}
	structType := structValue.Type()
	set := getopt.New()
	set.SetProgram(prog.ProgramName())
	for idx := 0; idx < structValue.NumField(); idx++ {
		fieldValue := structValue.Field(idx)
		fieldType := structType.Field(idx)
		if !fieldValue.CanAddr() {
			continue
		}
		longOptionName := strcase.ToKebab(fieldType.Name)
		shortOptionName := prog.ShortOptionName(longOptionName)
		docs := prog.DescribeOption(longOptionName)
		if docs == "" {
			panic(fmt.Errorf("%w for %s", errNoDocs, longOptionName))
		}
		fieldValuep := fieldValue.Addr()
		set.FlagLong(fieldValuep.Interface(), longOptionName, shortOptionName, docs)
	}
	return &Parser{set: set, prog: prog}
}

// castToStructPtr converts opts to a struct pointer, if possible.
func castToStructPtr(opts Program) (reflect.Value, error) {
	v := reflect.ValueOf(opts)
	if v.Kind() != reflect.Ptr {
		return reflect.Value{}, errExpectedPtr
	}
	vp := v.Elem()
	if vp.Kind() != reflect.Struct {
		return reflect.Value{}, errExpectedStructPtr
	}
	return vp, nil
}

// Parse parses the CLI options from os.Args.
func (p *Parser) Parse(args []string) {
	err := p.set.Getopt(args, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", p.prog.ProgramName(), err)
		p.PrintShortUsage(os.Stderr)
		os.Exit(1)
	}
	p.prog.AfterParsingChecks(p)
}

// PositionalArgs returns the positional arguments.
func (p *Parser) PositionalArgs() []string {
	return p.set.Args()
}

// PrintShortUsage shows a terse help message showing usage.
func (p *Parser) PrintShortUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s [options]", p.prog.ProgramName())
	if v := p.prog.PositionalArguments(); v != "" {
		fmt.Fprintf(w, " %s", v)
	}
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Try \"%s --help\" for more help.\n", p.prog.ProgramName())
}

// PrintLongUsage shows the complete help screen for the program.
func (p *Parser) PrintLongUsage(w io.Writer) {
	fmt.Fprintf(w, "%s.\n\n", p.prog.ShortDescription())
	fmt.Fprintf(w, "Usage:\n\n  %s [options]", p.prog.ProgramName())
	if v := p.prog.PositionalArguments(); v != "" {
		fmt.Fprintf(w, " %s", v)
	}
	fmt.Fprint(w, "\n\nOptions:\n\n")
	p.set.VisitAll(func(o getopt.Option) {
		if v := o.ShortName(); v != "" {
			fmt.Fprintf(w, "  -%s, --%s", v, o.LongName())
		} else {
			fmt.Fprintf(w, "  --%s", o.LongName())
		}
		if !o.IsFlag() {
			fmt.Fprintf(w, " VALUE")
		}
		fmt.Fprint(w, "\n")
		fmt.Fprintf(w, "\t%s.\n\n", p.prog.DescribeOption(o.LongName()))
		if !o.IsFlag() {
			if v := o.Value(); v.String() != "" {
				fmt.Fprintf(w, "\tDefault value: %s.\n\n", v)
			}
		}
	})
}
