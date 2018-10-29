# csvplus

[![GoDoc](https://godoc.org/github.com/j0hnsmith/csvplus?status.svg)](https://godoc.org/github.com/j0hnsmith/csvplus)

csvplus provides marshalling/unmarshalling of CSV data (with and without header rows) into slices of structs.

## Why?

`csv.NewReader().Read()` only provides records as `[]string` leaving the user to perform type conversion. Also more convenient to go to/from a slice, don't have .

## Examples
Unmarshal

```go
type Item struct {
    First  string     `csvplus:"first"`
    Second int        `csvplus:"second"`
    Third  *bool      `csvplus:"third"`
    Forth  *time.Time `csvplus:"forth" csvplusFormat:"2006-01"`
}

// The CSV data we want to unmarshal.
// If your data is in a *File (or other io.Reader), use UnmarshalReader().
data := []byte("first,second,third,forth\na,1,,2000-01\nb,2,f,")

var items []Item
err := csvplus.Unmarshal(data, &items)
if err != nil {
    panic(err)
}

fmt.Printf("%+v\n", items[0])
fmt.Printf("{First:%s Second:%d Third:%t (dereferenced) Forth:%s}\n", items[1].First, items[1].Second, *items[1].Third, items[1].Forth)
// Output:
// {First:a Second:1 Third:<nil> Forth:2000-01-01 00:00:00 +0000 UTC}
// {First:b Second:2 Third:false (dereferenced) Forth:<nil>}
```

Custom field unmarshalling
```go

// YesNoBool is an example field that implements Unmarshaler, it's used in an example.
type YesNoBool bool

// UnmarshalCSV is an implementation of the Unmarshaler interface, converts a string record to a native
// value for this type.
func (ynb *YesNoBool) UnmarshalCSV(s string) error {
    if ynb == nil {
        return fmt.Errorf("cannot unmarshal into nil pointer")
    }
    switch s {
    case "yes":
        *ynb = YesNoBool(true)
        return nil
    case "no":
        *ynb = YesNoBool(false)
        return nil
    }
    return fmt.Errorf("unable to convert %s to bool", s)
}

type Item struct {
    Name      string     `csvplus:"name"`
    Seen      *YesNoBool `csvplus:"seen"`   // custom type that implements Unmarshaler
    Agreed    YesNoBool  `csvplus:"agreed"` // custom type that implements Unmarshaler
    Timestamp *time.Time `csvplus:"when" csvplusFormat:"2006-01"`
}

// The CSV data we want to unmarshal, note the custom format.
data := []byte("name,seen,agreed,when\nRob,yes,yes,1999-11\nRuss,,no,")

var items []Item
err := csvplus.Unmarshal(data, &items)
if err != nil {
    panic(err)
}

for _, item := range items {
    fmt.Println(item)
}
```

Marshal

```go
type Item struct {
    First  string     `csvplus:"first"`
    Second int        `csvplus:"second"`
    Third  *bool      `csvplus:"third"`
    Fourth *time.Time `csvplus:"fourth" csvplusFormat:"2006-01"`
}

tm, _ := time.Parse("2006-01", "2000-01")
f := false
items := []Item{
    {"a", 1, nil, &tm},
    {"b", 2, &f, nil},
}
data, err := csvplus.Marshal(&items)
if err != nil {
    panic(err)
}

fmt.Println(string(data))
// Output:
// first,second,third,fourth
// a,1,,2000-01
// b,2,false,
```

## Ideas for improvement
* `csvplusNilVal` tag for custom nil values (eg '-', 'n/a')
* `csvplusTrueVal` & `csvplusFalseVal` (eg 'yes' and 'no' without custom types that implement `Marshaler`/`Unmarshaler` interfaces)
* `csvplusFormat` to also handle floats via string formatting (eg `%.3e`)

PRs welcome.

# Docs

https://godoc.org/github.com/j0hnsmith/csvplus
