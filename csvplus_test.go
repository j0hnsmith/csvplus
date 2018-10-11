package csvplus_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/j0hnsmith/csvplus"
)

func ExampleUnmarshal() {
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
	// Output:
	// {A:a B:1 C:<nil>}
	// {A:b B:2 C:false (dereferenced)}
}

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

func ExampleUnmarshaler() {
	//	type YesNoBool bool

	// 	func (ynb *YesNoBool) UnmarshalCSV(s string) error {
	// 		if ynb == nil {
	// 			return fmt.Errorf("cannot unmarshal into nil pointer")
	// 		}
	// 		switch s {
	// 		case "yes":
	// 			*ynb = YesNoBool(true)
	// 			return nil
	// 		case "no":
	// 			*ynb = YesNoBool(false)
	// 			return nil
	// 		}
	// 			return fmt.Errorf("unable to convert %s to bool", s)
	// 		}

	type Item struct {
		Name   string
		Agreed YesNoBool // custom type that implements Unmarshaler
	}

	// The CSV data we want to unmarshal, note the custom format.
	data := []byte("name,agreed\nRob,yes\nRuss,no")

	var items []Item
	err := csvplus.Unmarshal(data, &items)
	if err != nil {
		panic(err)
	}

	for _, item := range items {
		fmt.Println(item)
	}
	// Output:
	// {Rob true}
	// {Russ false}
}

type String struct {
	Field string
}

type Int struct {
	Field int
}

type IntPtr struct {
	Field *int
}

type Bool struct {
	Field bool
}

type Float32 struct {
	Field float32
}

type Float64 struct {
	Field float64
}

type DateTimeNano struct {
	Field time.Time `csvplus:"format:time.RFC3339Nano"`
}

type DateTimeRFC struct {
	Field time.Time `csvplus:"format:time.RFC3339"`
}

type DateTimeFormat struct {
	Field time.Time `csvplus:"format:2006-01"`
}

type DateTimeNoTag struct {
	Field time.Time
}

type MyString string

func (m *MyString) UnmarshalCSV(r string) error {
	if m == nil {
		return fmt.Errorf("cannot unmarshal into nil pointer")
	}
	*m = MyString(r)
	return nil
}

type Custom struct {
	Field MyString
}

type CustomPtr struct {
	Field *MyString
}

func TestUnmarshalRecord(t *testing.T) { // nolint: gocyclo
	t.Run("string pointer fails", func(t *testing.T) {
		a := "not a pointer to a struct"
		record := []string{"1"}
		err := csvplus.UnmarshalRecord(record, &a)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("struct fails", func(t *testing.T) {
		a := Int{Field: 1}
		record := []string{"1"}
		err := csvplus.UnmarshalRecord(record, a)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("struct fields length mismatch", func(t *testing.T) {
		a := Int{Field: 1}
		record := []string{"1", "2"}
		err := csvplus.UnmarshalRecord(record, &a)
		if err == nil {
			t.Error("expected error")
		}
		if !strings.HasPrefix(err.Error(), "field number mismatch") {
			t.Error("wrong error, expected: 'field number mismatch...'")
		}
	})

	t.Run("empty record", func(t *testing.T) {
		record := []string{""}
		s := new(Int)
		err := csvplus.UnmarshalRecord(record, s)
		if err != nil {
			t.Fatal(err)
		}
		if s.Field != 0 {
			t.Error("expected 0 (empty value)")
		}
	})

	t.Run("csvplus.Unmarshaler", func(t *testing.T) {
		t.Run("string field", func(t *testing.T) {
			record := []string{"foo"}
			s := new(Custom)
			err := csvplus.UnmarshalRecord(record, s)
			if err != nil {
				t.Fatal(err)
			}
			if s.Field != "foo" {
				t.Error("expected foo")
			}
		})
		t.Run("*string field", func(t *testing.T) {
			record := []string{"foo"}
			s := new(CustomPtr)
			err := csvplus.UnmarshalRecord(record, s)
			if err != nil {
				t.Fatal(err)
			}
			if *s.Field != "foo" {
				t.Error("expected foo")
			}
		})
	})

	t.Run("int", func(t *testing.T) {
		record := []string{"1"}
		s := new(Int)
		err := csvplus.UnmarshalRecord(record, s)
		if err != nil {
			t.Fatal(err)
		}
		if s.Field != 1 {
			t.Error("expected 1")
		}
	})

	t.Run("string", func(t *testing.T) {
		record := []string{"foo"}
		s := new(String)
		err := csvplus.UnmarshalRecord(record, s)
		if err != nil {
			t.Fatal(err)
		}
		if s.Field != "foo" {
			t.Error("expected foo")
		}
	})

	t.Run("int error", func(t *testing.T) {
		record := []string{"foo"}
		s := new(Int)
		err := csvplus.UnmarshalRecord(record, s)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "unable to convert foo to int in field Field"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error, expected: '%s'", expectedPrefix)
		}
	})

	t.Run("int ptr", func(t *testing.T) {
		// this test essentially covers pointers to any type that's supported as a value
		record := []string{"1"}
		s := new(IntPtr)
		err := csvplus.UnmarshalRecord(record, s)
		if err != nil {
			t.Fatal(err)
		}
		if *s.Field != 1 {
			t.Error("expected 1")
		}
	})

	t.Run("float32", func(t *testing.T) {
		record := []string{"1.0"}
		s := new(Float32)
		err := csvplus.UnmarshalRecord(record, s)
		if err != nil {
			t.Fatal(err)
		}
		if s.Field != float32(1.0) {
			t.Error("expected 1.0")
		}
	})

	t.Run("float32 error", func(t *testing.T) {
		record := []string{"foo"}
		s := new(Float32)
		err := csvplus.UnmarshalRecord(record, s)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "unable to convert foo to float in field Field"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error, expected: '%s'", expectedPrefix)
		}
	})

	t.Run("float64", func(t *testing.T) {
		record := []string{"1.0"}
		s := new(Float64)
		err := csvplus.UnmarshalRecord(record, s)
		if err != nil {
			t.Fatal(err)
		}
		if s.Field != float64(1.0) {
			t.Error("expected 1.0")
		}
	})

	t.Run("float64 error", func(t *testing.T) {
		record := []string{"foo"}
		s := new(Float64)
		err := csvplus.UnmarshalRecord(record, s)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "unable to convert foo to float in field Field"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error, expected: '%s'", expectedPrefix)
		}
	})

	t.Run("bool", func(t *testing.T) {
		var tests = []struct {
			Name     string
			Expected bool
		}{
			{
				"true",
				true,
			},
			{
				"1",
				true,
			},
			{
				"f",
				false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.Name, func(t *testing.T) {
				record := []string{tt.Name}
				s := new(Bool)
				err := csvplus.UnmarshalRecord(record, s)
				if err != nil {
					t.Fatal(err)
				}
				if s.Field != tt.Expected {
					t.Errorf("expected %v", tt.Expected)
				}
			})
		}
	})

	t.Run("bool error", func(t *testing.T) {
		record := []string{"foo"}
		s := new(Bool)
		err := csvplus.UnmarshalRecord(record, s)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "unable to convert foo to bool in field Field"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error, expected: '%s'", expectedPrefix)
		}
	})

	t.Run("time.Time", func(t *testing.T) {
		t.Run("RFC3339Nano", func(t *testing.T) {
			dt := time.Now().UTC()
			dts := dt.Format(time.RFC3339Nano)
			record := []string{dts}
			s := new(DateTimeNano)
			err := csvplus.UnmarshalRecord(record, s)
			if err != nil {
				t.Fatal(err)
			}
			if s.Field != dt {
				t.Errorf("expected %v, got %v", dt, s.Field)
			}
		})

		t.Run("RFC3339", func(t *testing.T) {
			dt := time.Now().UTC()
			dts := dt.Format(time.RFC3339)
			record := []string{dts}
			s := new(DateTimeRFC)
			err := csvplus.UnmarshalRecord(record, s)
			if err != nil {
				t.Fatal(err)
			}
			dt1, _ := time.Parse(time.RFC3339, dts)
			if s.Field != dt1 {
				t.Errorf("expected %v, got %v", dt1, s.Field)
			}
		})

		t.Run("custom format", func(t *testing.T) {
			dt := time.Now().UTC()
			format := "2006-01"
			dts := dt.Format(format)
			record := []string{dts}
			s := new(DateTimeFormat)
			err := csvplus.UnmarshalRecord(record, s)
			if err != nil {
				t.Fatal(err)
			}
			dt1, _ := time.Parse(format, dts)
			if s.Field != dt1 {
				t.Errorf("expected %v, got %v", dt1, s.Field)
			}
		})

		t.Run("no struct tag", func(t *testing.T) {
			record := []string{"2018-10"}
			s := new(DateTimeNoTag)
			err := csvplus.UnmarshalRecord(record, s)
			if err == nil {
				t.Error("expected error because time.Time field without a layout in a struct tag")
			}
		})

		t.Run("invalid format", func(t *testing.T) {
			dt := time.Now().UTC()
			dts := dt.Format("invalid format")
			record := []string{dts}
			s := new(DateTimeRFC)
			err := csvplus.UnmarshalRecord(record, s)
			if err == nil {
				t.Fatal("expected error")
			}
			expectedPrefix := "invalid layout format for field Field"
			if !strings.HasPrefix(err.Error(), expectedPrefix) {
				t.Errorf("wrong error prefix, expected: '%s', got: %s", expectedPrefix, err.Error())
			}
		})
	})
}

func TestUnmarshal(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		type Item struct {
			A string
			B int
		}
		data := []byte("first,second\na,1\nb,2")
		var items []Item
		err := csvplus.Unmarshal(data, &items)
		if err != nil {
			t.Fatal(err)
		}
		if items[0].A != "a" {
			t.Errorf("expected 'a', got: %s", items[0].A)
		}
		if items[0].B != 1 {
			t.Errorf("expected 1, got: %d", items[0].B)
		}
		if items[1].A != "b" {
			t.Errorf("expected 'b', got: %s", items[1].A)
		}
		if items[1].B != 2 {
			t.Errorf("expected 2, got: %d", items[1].B)
		}
	})

	t.Run("slice as value instead of pointer", func(t *testing.T) {
		type Item struct {
			A string
			B int
		}
		data := []byte("first,second\na,1\nb,2")
		var items []Item
		err := csvplus.Unmarshal(data, items)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "non pointer"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error prefix, expected: '%s', got: %s", expectedPrefix, err.Error())
		}
	})
}

func TestUnmarshalReader(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		type Item struct {
			A string
			B int
		}
		data := []byte("first,second\na,1\nb,2")
		buf := bytes.NewBuffer(data)
		var items []Item
		err := csvplus.UnmarshalReader(buf, &items)
		if err != nil {
			t.Fatal(err)
		}
		if items[0].A != "a" {
			t.Errorf("expected 'a', got: %s", items[0].A)
		}
		if items[0].B != 1 {
			t.Errorf("expected 1, got: %d", items[0].B)
		}
		if items[1].A != "b" {
			t.Errorf("expected 'b', got: %s", items[1].A)
		}
		if items[1].B != 2 {
			t.Errorf("expected 2, got: %d", items[1].B)
		}
	})
}