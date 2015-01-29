# jsonenums

JSONenums is a tool to automate the creation of methods that satisfy the
`fmt.Stringer`, `json.Marshaler` and `json.Unmarshaler` interfaces.
Given the name of a (signed or unsigned) integer type T that has constants
defined, stringer will create a new self-contained Go source file implementing

```
    func (t T) String() string
    func (t T) MarshalJSON() ([]byte, error)
    func (t *T) UnmarshalJSON([]byte) error
```

The file is created in the same package and directory as the package that
defines T. It has helpful defaults designed for use with go generate.

JSONenums is a simple implementation of a concept and the code might not be the
most performant or beautiful to read.

For example, given this snippet,

```
    package painkiller

    type Pill int

    const (
        Placebo Pill = iota
        Aspirin
        Ibuprofen
        Paracetamol
        Acetaminophen = Paracetamol
    )
```

running this command

```
    jsonenums -type=Pill
```

in the same directory will create the file `pill_jsonenums.go`, in package
`painkiller`, containing a definition of

```
    func (r Pill) String() string
    func (r Pill) MarshalJSON() ([]byte, error)
    func (r *Pill) UnmarshalJSON([]byte) error
```

That method will translate the value of a Pill constant to the string
representation of the respective constant name, so that the call
`fmt.Print(painkiller.Aspirin) will print the string "Aspirin".

Typically this process would be run using go generate, like this:

```
    //go:generate stringer -type=Pill
```

If multiple constants have the same value, the lexically first matching name
will be used (in the example, Acetaminophen will print as "Paracetamol").

With no arguments, it processes the package in the current directory. Otherwise,
the arguments must name a single directory holding a Go package or a set of Go
source files that represent a single Go package.

The `-type` flag accepts a comma-separated list of types so a single run can
generate methods for multiple types. The default output file is t_string.go,
where t is the lower-cased name of the first type listed. THe suffix can be
overridden with the `-suffix` flag.
