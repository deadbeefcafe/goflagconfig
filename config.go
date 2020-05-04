/*Package config implements config file parsing.

Usage:

Define configs using config.String(), Bool(), Int(), etc.

This declares an integer config, -configname, stored in the pointer ip, with type *int.
	import "config"
	var ip = config.Int("configname", 1234, "help message for configname")
If you like, you can bind the config to a variable using the Var() functions.
	var configvar int
	func init() {
		config.IntVar(&configvar, "configname", 1234, "help message for configname")
	}
Or you can create custom configs that satisfy the Value interface (with
pointer receivers) and couple them to config parsing by
	config.Var(&configVal, "name", "help message for configname")
For such configs, the default value is just the initial value of the variable.

After all configs are defined, call
	config.Parse()
to parse the command line into the defined configs.

Configs may then be used directly. If you're using the configs themselves,
they are all pointers; if you bind to variables, they're values.
	fmt.Println("ip has value ", *ip)
	fmt.Println("configvar has value ", configvar)

After parsing, the arguments following the configs are available as the
slice config.Args() or individually as config.Arg(i).
The arguments are indexed from 0 through config.NArg()-1.

Command line config syntax:
	-config
	-config=x
	-config x  // non-boolean configs only
One or two minus signs may be used; they are equivalent.
The last form is not permitted for boolean configs because the
meaning of the command
	cmd -x *
will change if there is a file called 0, false, etc.  You must
use the -config=false form to turn off a boolean config.

Config parsing stops just before the first non-config argument
("-" is a non-config argument) or after the terminator "--".

Integer configs accept 1234, 0664, 0x1234 and may be negative.
Boolean configs may be:
	1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False
Duration configs accept any input valid for time.ParseDuration.

The default set of command-line configs is controlled by
top-level functions.  The ConfigSet type allows one to define
independent sets of configs, such as to implement subcommands
in a command-line interface. The methods of ConfigSet are
analogous to the top-level functions for the command-line
config set.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
*/
package goflagconfig

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// -- bool Value
type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue {
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	*b = boolValue(v)
	return err
}

func (b *boolValue) Get() interface{} { return bool(*b) }

func (b *boolValue) String() string { return strconv.FormatBool(bool(*b)) }

// -- int Value
type intValue int

func newIntValue(val int, p *int) *intValue {
	*p = val
	return (*intValue)(p)
}

func (i *intValue) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = intValue(v)
	return err
}

func (i *intValue) Get() interface{} { return int(*i) }

func (i *intValue) String() string { return strconv.Itoa(int(*i)) }

// -- int64 Value
type int64Value int64

func newInt64Value(val int64, p *int64) *int64Value {
	*p = val
	return (*int64Value)(p)
}

func (i *int64Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = int64Value(v)
	return err
}

func (i *int64Value) Get() interface{} { return int64(*i) }

func (i *int64Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// -- uint Value
type uintValue uint

func newUintValue(val uint, p *uint) *uintValue {
	*p = val
	return (*uintValue)(p)
}

func (i *uintValue) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	*i = uintValue(v)
	return err
}

func (i *uintValue) Get() interface{} { return uint(*i) }

func (i *uintValue) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- uint64 Value
type uint64Value uint64

func newUint64Value(val uint64, p *uint64) *uint64Value {
	*p = val
	return (*uint64Value)(p)
}

func (i *uint64Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	*i = uint64Value(v)
	return err
}

func (i *uint64Value) Get() interface{} { return uint64(*i) }

func (i *uint64Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- string Value
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() interface{} { return string(*s) }

func (s *stringValue) String() string { return string(*s) }

// -- float64 Value
type float64Value float64

func newFloat64Value(val float64, p *float64) *float64Value {
	*p = val
	return (*float64Value)(p)
}

func (f *float64Value) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	*f = float64Value(v)
	return err
}

func (f *float64Value) Get() interface{} { return float64(*f) }

func (f *float64Value) String() string { return strconv.FormatFloat(float64(*f), 'g', -1, 64) }

// -- time.Duration Value
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error {
	v, err := time.ParseDuration(s)
	*d = durationValue(v)
	return err
}

func (d *durationValue) Get() interface{} { return time.Duration(*d) }

func (d *durationValue) String() string { return (*time.Duration)(d).String() }

// Value is the interface to the dynamic value stored in a config.
// (The default value is represented as a string.)
//
// If a Value has an IsBoolConfig() bool method returning true,
// the command-line parser makes -name equivalent to -name=true
// rather than using the next command-line argument.
//
// Set is called once, in command line order, for each config present.
// The config package may call the String method with a zero-valued receiver,
// such as a nil pointer.
type Value interface {
	String() string
	Set(string) error
	Get() interface{}
}

// Getter is an interface that allows the contents of a Value to be retrieved.
// It wraps the Value interface, rather than being part of it, because it
// appeared after Go 1 and its compatibility rules. All Value types provided
// by this package satisfy the Getter interface.
/*
type Getter interface {
	Value
	Get() interface{}
}
*/

// A ConfigSet represents a set of defined configs. The zero value of a ConfigSet
// has no name and has ContinueOnError error handling.
type ConfigSet struct {
	filename string
	parsed   bool
	actual   map[string]*Config
	formal   map[string]*Config
}

// A Config represents the state of a config.
type Config struct {
	Name     string // name as it appears on command line
	Usage    string // help message
	Value    Value  // value as set
	DefValue string // default value (as text); for usage message
}

// sortConfigs returns the configs as a slice in lexicographical sorted order.
func sortConfigs(configs map[string]*Config) []*Config {
	list := make(sort.StringSlice, len(configs))
	i := 0
	for _, f := range configs {
		list[i] = f.Name
		i++
	}
	list.Sort()
	result := make([]*Config, len(list))
	for i, name := range list {
		result[i] = configs[name]
	}
	return result
}

// VisitAll visits the configs in lexicographical order, calling fn for each.
// It visits all configs, even those not set.
func (f *ConfigSet) VisitAll(fn func(*Config)) {
	for _, config := range sortConfigs(f.formal) {
		fn(config)
	}
}

// VisitAll visits the command-line configs in lexicographical order, calling
// fn for each. It visits all configs, even those not set.
func VisitAll(fn func(*Config)) {
	Configuration.VisitAll(fn)
}

// Visit visits the configs in lexicographical order, calling fn for each.
// It visits only those configs that have been set.
func (f *ConfigSet) Visit(fn func(*Config)) {
	for _, config := range sortConfigs(f.actual) {
		fn(config)
	}
}

// Visit visits the command-line configs in lexicographical order, calling fn
// for each. It visits only those configs that have been set.
func Visit(fn func(*Config)) {
	Configuration.Visit(fn)
}

// Lookup returns the Config structure of the named config, returning nil if none exists.
func (f *ConfigSet) Lookup(name string) *Config {
	return f.formal[name]
}

// Lookup returns the Config structure of the named command-line config,
// returning nil if none exists.
func Lookup(name string) *Config {
	return Configuration.formal[name]
}

// Set sets the value of the named config.
func (f *ConfigSet) Set(name, value string) error {
	config, ok := f.formal[name]
	if !ok {
		f.String(name, value, "")
		fmt.Printf("Added config (string) %s = %s\n", name, value)
		return nil
		//return fmt.Errorf("no such config %v", name)
	}
	err := config.Value.Set(value)
	if err != nil {
		return err
	}
	if f.actual == nil {
		f.actual = make(map[string]*Config)
	}
	f.actual[name] = config
	return nil
}

// Set sets the value of the named command-line config.
func Set(name, value string) error {
	return Configuration.Set(name, value)
}

// NConfig returns the number of configs that have been set.
func (f *ConfigSet) NConfig() int { return len(f.actual) }

// NConfig returns the number of command-line configs that have been set.
func NConfig() int { return len(Configuration.actual) }

// BoolVar defines a bool config with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the config.
func (f *ConfigSet) BoolVar(p *bool, name string, value bool, usage string) {
	f.Var(newBoolValue(value, p), name, usage)
}

// BoolVar defines a bool config with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the config.
func BoolVar(p *bool, name string, value bool, usage string) {
	Configuration.Var(newBoolValue(value, p), name, usage)
}

// Bool defines a bool config with specified name, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the config.
func (f *ConfigSet) Bool(name string, value bool, usage string) *bool {
	p := new(bool)
	f.BoolVar(p, name, value, usage)
	return p
}

// Bool defines a bool config with specified name, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the config.
func Bool(name string, value bool, usage string) *bool {
	return Configuration.Bool(name, value, usage)
}

// IntVar defines an int config with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the config.
func (f *ConfigSet) IntVar(p *int, name string, value int, usage string) {
	f.Var(newIntValue(value, p), name, usage)
}

// IntVar defines an int config with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the config.
func IntVar(p *int, name string, value int, usage string) {
	Configuration.Var(newIntValue(value, p), name, usage)
}

// Int defines an int config with specified name, default value, and usage string.
// The return value is the address of an int variable that stores the value of the config.
func (f *ConfigSet) Int(name string, value int, usage string) *int {
	p := new(int)
	f.IntVar(p, name, value, usage)
	return p
}

// Int defines an int config with specified name, default value, and usage string.
// The return value is the address of an int variable that stores the value of the config.
func Int(name string, value int, usage string) *int {
	return Configuration.Int(name, value, usage)
}

// Int64Var defines an int64 config with specified name, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the config.
func (f *ConfigSet) Int64Var(p *int64, name string, value int64, usage string) {
	f.Var(newInt64Value(value, p), name, usage)
}

// Int64Var defines an int64 config with specified name, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the config.
func Int64Var(p *int64, name string, value int64, usage string) {
	Configuration.Var(newInt64Value(value, p), name, usage)
}

// Int64 defines an int64 config with specified name, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the config.
func (f *ConfigSet) Int64(name string, value int64, usage string) *int64 {
	p := new(int64)
	f.Int64Var(p, name, value, usage)
	return p
}

// Int64 defines an int64 config with specified name, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the config.
func Int64(name string, value int64, usage string) *int64 {
	return Configuration.Int64(name, value, usage)
}

// UintVar defines a uint config with specified name, default value, and usage string.
// The argument p points to a uint variable in which to store the value of the config.
func (f *ConfigSet) UintVar(p *uint, name string, value uint, usage string) {
	f.Var(newUintValue(value, p), name, usage)
}

// UintVar defines a uint config with specified name, default value, and usage string.
// The argument p points to a uint  variable in which to store the value of the config.
func UintVar(p *uint, name string, value uint, usage string) {
	Configuration.Var(newUintValue(value, p), name, usage)
}

// Uint defines a uint config with specified name, default value, and usage string.
// The return value is the address of a uint  variable that stores the value of the config.
func (f *ConfigSet) Uint(name string, value uint, usage string) *uint {
	p := new(uint)
	f.UintVar(p, name, value, usage)
	return p
}

// Uint defines a uint config with specified name, default value, and usage string.
// The return value is the address of a uint  variable that stores the value of the config.
func Uint(name string, value uint, usage string) *uint {
	return Configuration.Uint(name, value, usage)
}

// Uint64Var defines a uint64 config with specified name, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the config.
func (f *ConfigSet) Uint64Var(p *uint64, name string, value uint64, usage string) {
	f.Var(newUint64Value(value, p), name, usage)
}

// Uint64Var defines a uint64 config with specified name, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the config.
func Uint64Var(p *uint64, name string, value uint64, usage string) {
	Configuration.Var(newUint64Value(value, p), name, usage)
}

// Uint64 defines a uint64 config with specified name, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the config.
func (f *ConfigSet) Uint64(name string, value uint64, usage string) *uint64 {
	p := new(uint64)
	f.Uint64Var(p, name, value, usage)
	return p
}

// Uint64 defines a uint64 config with specified name, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the config.
func Uint64(name string, value uint64, usage string) *uint64 {
	return Configuration.Uint64(name, value, usage)
}

// StringVar defines a string config with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the config.
func (f *ConfigSet) StringVar(p *string, name string, value string, usage string) {
	f.Var(newStringValue(value, p), name, usage)
}

// StringVar defines a string config with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the config.
func StringVar(p *string, name string, value string, usage string) {
	Configuration.Var(newStringValue(value, p), name, usage)
}

// String defines a string config with specified name, default value, and usage string.
// The return value is the address of a string variable that stores the value of the config.
func (f *ConfigSet) String(name string, value string, usage string) *string {
	p := new(string)
	f.StringVar(p, name, value, usage)
	return p
}

// String defines a string config with specified name, default value, and usage string.
// The return value is the address of a string variable that stores the value of the config.
func String(name string, value string, usage string) *string {
	return Configuration.String(name, value, usage)
}

// Float64Var defines a float64 config with specified name, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the config.
func (f *ConfigSet) Float64Var(p *float64, name string, value float64, usage string) {
	f.Var(newFloat64Value(value, p), name, usage)
}

// Float64Var defines a float64 config with specified name, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the config.
func Float64Var(p *float64, name string, value float64, usage string) {
	Configuration.Var(newFloat64Value(value, p), name, usage)
}

// Float64 defines a float64 config with specified name, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the config.
func (f *ConfigSet) Float64(name string, value float64, usage string) *float64 {
	p := new(float64)
	f.Float64Var(p, name, value, usage)
	return p
}

// Float64 defines a float64 config with specified name, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the config.
func Float64(name string, value float64, usage string) *float64 {
	return Configuration.Float64(name, value, usage)
}

// DurationVar defines a time.Duration config with specified name, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the config.
// The config accepts a value acceptable to time.ParseDuration.
func (f *ConfigSet) DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	f.Var(newDurationValue(value, p), name, usage)
}

// DurationVar defines a time.Duration config with specified name, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the config.
// The config accepts a value acceptable to time.ParseDuration.
func DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	Configuration.Var(newDurationValue(value, p), name, usage)
}

// Duration defines a time.Duration config with specified name, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the config.
// The config accepts a value acceptable to time.ParseDuration.
func (f *ConfigSet) Duration(name string, value time.Duration, usage string) *time.Duration {
	p := new(time.Duration)
	f.DurationVar(p, name, value, usage)
	return p
}

// Duration defines a time.Duration config with specified name, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the config.
// The config accepts a value acceptable to time.ParseDuration.
func Duration(name string, value time.Duration, usage string) *time.Duration {
	return Configuration.Duration(name, value, usage)
}

// Var defines a config with the specified name and usage string. The type and
// value of the config are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a config that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
func (f *ConfigSet) Var(value Value, name string, usage string) {
	// Remember the default value as a string; it won't change.
	config := &Config{name, usage, value, value.String()}
	_, alreadythere := f.formal[name]
	if alreadythere {
		var msg string
		if f.filename == "" {
			msg = fmt.Sprintf("config redefined: %s", name)
		} else {
			msg = fmt.Sprintf("%s config redefined: %s", f.filename, name)
		}
		fmt.Println(msg)
		panic(msg) // Happens only if configs are declared with identical names
	}
	if f.formal == nil {
		f.formal = make(map[string]*Config)
	}
	f.formal[name] = config
}

// Var defines a config with the specified name and usage string. The type and
// value of the config are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a config that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
func Var(value Value, name string, usage string) {
	Configuration.Var(value, name, usage)
}

// Configuration is the default set of command-line configs, parsed from os.Args.
// The top-level functions such as BoolVar, Arg, and so on are wrappers for the
// methods of Configuration.
var Configuration = NewConfigSet("")

func init() {
	//Configuration.filename = ""
}

// NewConfigSet returns a new, empty config set with the specified name and
// error handling property.
func NewConfigSet(filename string) *ConfigSet {
	f := &ConfigSet{
		filename: filename,
	}
	//f.Usage = f.defaultUsage
	return f
}

// Init sets the name and error handling property for a config set.
// By default, the zero ConfigSet uses an empty name and the
// ContinueOnError error handling policy.
func (f *ConfigSet) Init(filename string) {
	f.filename = filename
}

// Save writes the configuration to the filename configured in the
// NewConfigSet function
func (f *ConfigSet) Save() {
	if f.filename == "" {
		fmt.Printf("No filename to save.\n")
		return
	}
	fmt.Printf("Writing config to %s\n", f.filename)
	out, err := os.Create(f.filename)
	if err != nil {
		return
	}
	defer out.Close()

	visitor := func(f *Config) {
		fmt.Fprintf(out, "%s=%s # %s\n", f.Name, f.Value.String(), f.Usage)
	}
	VisitAll(visitor)
	fmt.Printf("Done.\n")
}

// Print will dump all the current configuration settings
func (f *ConfigSet) Print() {
	visitor := func(f *Config) {
		fmt.Printf("%-20s = %s # %s\n", f.Name, f.Value.String(), f.Usage)
	}
	VisitAll(visitor)
}

func (f *ConfigSet) Load() {
	if f.filename == "" {
		fmt.Printf("No file to load.\n")
		return
	}
	fmt.Printf("Loading config from %s\n", f.filename)
	in, err := os.Open(f.filename)
	if err != nil {
		return
	}
	defer in.Close()

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		//fmt.Printf("LINE: [%s]\n", line)
		ci := strings.Index(line, "#")
		if ci > -1 {
			line = line[:ci]
		}
		kv := strings.Split(line, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			err = f.Set(key, val)
			if err != nil {
				fmt.Printf("f.Set returned err=%v\n", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return
	}
}

func SetFile(filename string) {
	Configuration.filename = filename
}

func Save() {
	Configuration.Save()
}

func Print() {
	Configuration.Print()
}

func Load() {
	Configuration.Load()
}
