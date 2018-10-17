# CSVPlus

CSVPlus provides marshalling/unmarshalling* of CSV data (with header rows) into go structs.

* marshalling on todo list

## Why?

`csv.NewReader().Read()` only provides records as `[]string` leaving the user to perform type conversion.

## TODO

* [ ] add support for marshalling
* [x] expose `csv.Reader` config
* [x] consider using struct tags to map CSV colums to struct fields

## Examples
Unmarshal

```go
type Item struct {
    A string
    B int

    C *bool
}

// The CSV data we want to unmarshal.
// If your data is in a *File (or other io.Reader), use UnmarshalReader().
data := []byte("first,second,third\na,1,\nb,2,f")

var items []Item
err := csvplus.Unmarshal(data, &items)
if err != nil {
    panic(err)
}

fmt.Printf("%+v\n", items[0])
fmt.Printf("{A:%s B:%d C:%t (dereferenced)}\n", items[1].A, items[1].B, *items[1].C)
```

Custom field unmarshalling
```go

// YesNoBool is an example field that implements Unmarhsaler, it's used in an example.
type YesNoBool bool

// UnmarshalCSV is an implementation of the Unmarshaller interface, converts a string record to a native
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

time.Time layout via struct tag
```go
type MyStruct struct {
	Field time.Time `csvplus:"format:time.RFC3339"`
}

data := []byte("name,dob\nRob,2000-01-01T12:00:00Z\nRuss,2000-01-01T12:00:00Z")

data := []string
err := csvplus.Unmarshal(record, s)
if err != nil {
    t.Fatal(err)
}

for _, item := range items {
    fmt.Println(item)
}
```
 

