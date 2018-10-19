package csvplus_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/j0hnsmith/csvplus"
)

func ExampleUnmarshal() {
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
}

var benchItems interface{}

func BenchmarkUnmarshal(b *testing.B) {

	type Item struct {
		First  string `csvplus:"first"`
		Second int    `csvplus:"second"`
		Third  *bool  `csvplus:"third"`
	}

	// The CSV data we want to unmarshal.
	// If your data is in a *File (or other io.Reader), use UnmarshalReader().
	data := []byte("first,second,third\na,1,\nb,2,f")

	var items []Item

	for n := 0; n < b.N; n++ {
		items = nil
		// always record the result of Fib to prevent
		// the compiler eliminating the function call.
		err := csvplus.Unmarshal(data, &items)
		if err != nil {
			panic(err)
		}
	}
	// always store the result to a package level variable
	// so the compiler cannot eliminate the Benchmark itself.
	benchItems = items
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

func (ynb *YesNoBool) MarshalCSV() ([]byte, error) {
	if ynb == nil || !*ynb {
		return []byte("no"), nil
	}
	return []byte("yes"), nil
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

	fmt.Printf("{%s %t (dereferenced) %t %s}\n", items[0].Name, *items[0].Seen, items[0].Agreed, items[0].Timestamp)
	fmt.Printf("{%s %+v %t %+v}\n", items[1].Name, items[1].Seen, items[1].Agreed, items[1].Timestamp)
	// Output:
	// {Rob true (dereferenced) true 1999-11-01 00:00:00 +0000 UTC}
	// {Russ <nil> false <nil>}
}

func TestUnmarshal(t *testing.T) { // nolint: gocyclo
	t.Run("general", func(t *testing.T) {
		t.Run("slice as value instead of pointer", func(t *testing.T) {
			type Item struct {
				First  string
				Second int
			}
			data := []byte("First,Second\na,1\nb,2")
			var items []Item
			err := csvplus.Unmarshal(data, items)
			expectedPrefix := "non pointer"
			if !strings.HasPrefix(err.Error(), expectedPrefix) {
				t.Errorf("wrong error prefix, expected: '%s', got: %s", expectedPrefix, err.Error())
			}
		})

		t.Run("string pointer fails", func(t *testing.T) {
			a := "not a pointer to a slice"
			data := []byte("First,Second\na,1\nb,2")
			err := csvplus.Unmarshal(data, &a)
			if err == nil {
				t.Error("expected error")
			}
		})

		t.Run("struct field not in csv", func(t *testing.T) {
			type Item struct {
				First  string
				Second int
				Third  *bool
			}
			data := []byte("First,Second\na,1\nb,2")
			var items []Item
			err := csvplus.Unmarshal(data, &items)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Errorf("expected len(items) to be 2, got: %d", len(items))
			}
			if items[0].Third != nil {
				t.Errorf("expected nil, got: %v", items[0].Third)
			}
			if items[1].Third != nil {
				t.Errorf("expected nil, got: %v", items[0].Third)
			}
		})
	})

	t.Run("field types", func(t *testing.T) {

		t.Run("uint8", func(t *testing.T) {
			type Item struct {
				First uint8 `csvplus:"first"`
			}
			data := []byte("first\n7")
			var items []Item
			err := csvplus.Unmarshal(data, &items)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 1 {
				t.Errorf("expected len of %d, got: %d", 1, len(items))
			}
			if items[0].First != 7 {
				t.Errorf("expected '7', got: %d", items[0].First)
			}
		})

		t.Run("uint error", func(t *testing.T) {
			type Item struct {
				First uint
			}
			data := []byte("First\n-7")

			var items []Item
			err := csvplus.Unmarshal(data, &items)
			expectedPrefix := "unable to convert -7 to uint in field First"
			if !strings.HasPrefix(err.Error(), expectedPrefix) {
				t.Errorf("wrong error, expected: '%s', got: %s", expectedPrefix, err.Error())
			}
		})

		t.Run("int error", func(t *testing.T) {
			type Item struct {
				First int
			}
			data := []byte("First\nfoo")

			var items []Item
			err := csvplus.Unmarshal(data, &items)
			expectedPrefix := "unable to convert foo to int in field First"
			if !strings.HasPrefix(err.Error(), expectedPrefix) {
				t.Errorf("wrong error, expected: '%s'", expectedPrefix)
			}
		})

		t.Run("float32 error", func(t *testing.T) {
			type Item struct {
				First float32
			}
			data := []byte("First\nfoo")

			var items []Item
			err := csvplus.Unmarshal(data, &items)
			expectedPrefix := "unable to convert foo to float in field First"
			if !strings.HasPrefix(err.Error(), expectedPrefix) {
				t.Errorf("wrong error, expected: '%s', got: %s", expectedPrefix, err.Error())
			}
		})

		t.Run("bool", func(t *testing.T) {
			var tests = []struct {
				Name     string
				Data     []byte
				Expected bool
			}{
				{
					"true",
					[]byte("First\ntrue"),
					true,
				},
				{
					"1",
					[]byte("First\n1"),
					true,
				},
				{
					"t",
					[]byte("First\nt"),
					true,
				},
			}

			for _, tt := range tests {
				t.Run(tt.Name, func(t *testing.T) {
					type Item struct {
						First bool
					}
					var items []Item
					err := csvplus.Unmarshal(tt.Data, &items)
					if err != nil {
						t.Fatal(err)
					}
					if items[0].First != tt.Expected {
						t.Errorf("expected %v", tt.Expected)
					}
				})
			}
		})

		t.Run("bool error", func(t *testing.T) {
			type Item struct {
				First bool
			}
			data := []byte("First\nfoo")

			var items []Item
			err := csvplus.Unmarshal(data, &items)
			expectedPrefix := "unable to convert foo to bool in field First"
			if !strings.HasPrefix(err.Error(), expectedPrefix) {
				t.Errorf("wrong error, expected: '%s', got: %s", expectedPrefix, err.Error())
			}
		})

		t.Run("time.Time", func(t *testing.T) {
			t.Run("RFC3339Nano", func(t *testing.T) {
				type Item struct {
					First time.Time `csvplusFormat:"time.RFC3339Nano"`
				}

				dt := time.Now().UTC()
				dts := dt.Format(time.RFC3339Nano)
				data := []byte(fmt.Sprintf("First\n%s", dts))
				var items []Item
				err := csvplus.Unmarshal(data, &items)
				if err != nil {
					t.Fatal(err)
				}
				if items[0].First != dt {
					t.Errorf("expected %v, got %v", dt, items[0].First)
				}
			})

			t.Run("RFC3339", func(t *testing.T) {
				type Item struct {
					First time.Time `csvplusFormat:"time.RFC3339"`
				}

				dt := time.Now().UTC()
				dts := dt.Format(time.RFC3339)
				data := []byte(fmt.Sprintf("First\n%s", dts))
				var items []Item
				err := csvplus.Unmarshal(data, &items)
				if err != nil {
					t.Fatal(err)
				}
				dt1, _ := time.Parse(time.RFC3339, dts)
				if items[0].First != dt1 {
					t.Errorf("expected %v, got %v", dt1, items[0].First)
				}
			})

			t.Run("RFC3339 is default", func(t *testing.T) {
				type Item struct {
					First time.Time
				}

				dt := time.Now().UTC()
				dts := dt.Format(time.RFC3339)
				data := []byte(fmt.Sprintf("First\n%s", dts))
				var items []Item
				err := csvplus.Unmarshal(data, &items)
				if err != nil {
					t.Fatal(err)
				}
				dt1, _ := time.Parse(time.RFC3339, dts)
				if items[0].First != dt1 {
					t.Errorf("expected %v, got %v", dt1, items[0].First)
				}
			})

			t.Run("custom format", func(t *testing.T) {
				type Item struct {
					First time.Time `csvplusFormat:"2006-01"`
				}

				dt := time.Now().UTC()
				format := "2006-01"
				dts := dt.Format(format)
				data := []byte(fmt.Sprintf("First\n%s", dts))
				var items []Item
				err := csvplus.Unmarshal(data, &items)
				if err != nil {
					t.Fatal(err)
				}
				dt1, _ := time.Parse(format, dts)
				if items[0].First != dt1 {
					t.Errorf("expected %v, got %v", dt1, items[0].First)
				}
			})

			t.Run("invalid format", func(t *testing.T) {
				type Item struct {
					First time.Time `csvplusFormat:"invalid format"`
				}

				dt := time.Now().UTC()
				format := "2006-01"
				dts := dt.Format(format)
				data := []byte(fmt.Sprintf("First\n%s", dts))
				var items []Item
				err := csvplus.Unmarshal(data, &items)
				expectedPrefix := "invalid layout format for field First"
				if !strings.HasPrefix(err.Error(), expectedPrefix) {
					t.Errorf("wrong error prefix, expected: '%s', got: %s", expectedPrefix, err.Error())
				}
			})
		})
	})

	t.Run("header row", func(t *testing.T) {
		t.Run("works, all simple go types with col tags", func(t *testing.T) {
			type Item struct {
				First  string  `csvplus:"first"`
				Second int     `csvplus:"second"`
				Third  *bool   `csvplus:"third"`
				Forth  float64 `csvplus:"forth"`
			}
			data := []byte("first,second,third,forth\na,1,,0.2\nb,2,f,1")
			var items []Item
			err := csvplus.Unmarshal(data, &items)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Errorf("expected len of %d, got: %d", 2, len(items))
			}
			if items[0].First != "a" {
				t.Errorf("expected 'a', got: %s", items[0].First)
			}
			if items[0].Second != 1 {
				t.Errorf("expected 1, got: %d", items[0].Second)
			}
			if items[0].Third != nil {
				t.Errorf("expected pointer field to be nil, got: %v", items[0].Third)
			}
			if items[0].Forth != 0.2 {
				t.Errorf("expected 0.2, got: %.2f", items[0].Forth)
			}
			if items[1].First != "b" {
				t.Errorf("expected 'b', got: %s", items[1].First)
			}
			if items[1].Second != 2 {
				t.Errorf("expected 2, got: %d", items[1].Second)
			}
			if *items[1].Third {
				t.Errorf("expected false, got: %v", *items[1].Third)
			}
			if items[1].Forth != 1 {
				t.Errorf("expected 0.2, got: %.2f", items[1].Forth)
			}
		})

		t.Run("works without tags (col name same as struct field name)", func(t *testing.T) {
			type Item struct {
				First  string
				Second int
			}
			data := []byte("First,Second\na,1\nb,2")
			var items []Item
			err := csvplus.Unmarshal(data, &items)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Errorf("expected len of %d, got: %d", 2, len(items))
			}
			if items[0].First != "a" {
				t.Errorf("expected 'a', got: %s", items[0].First)
			}
			if items[0].Second != 1 {
				t.Errorf("expected 1, got: %d", items[0].Second)
			}
			if items[1].First != "b" {
				t.Errorf("expected 'b', got: %s", items[1].First)
			}
			if items[1].Second != 2 {
				t.Errorf("expected 2, got: %d", items[1].Second)
			}
		})

		t.Run("lowercased field names in data match", func(t *testing.T) {
			type Item struct {
				First  string
				Second int
			}
			data := []byte("first,second\na,1\nb,2")
			var items []Item
			err := csvplus.Unmarshal(data, &items)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Errorf("expected len of %d, got: %d", 2, len(items))
			}
			if items[0].First != "a" {
				t.Errorf("expected 'a', got: %s", items[0].First)
			}
			if items[0].Second != 1 {
				t.Errorf("expected 1, got: %d", items[0].Second)
			}
			if items[1].First != "b" {
				t.Errorf("expected 'b', got: %s", items[1].First)
			}
			if items[1].Second != 2 {
				t.Errorf("expected 2, got: %d", items[1].Second)
			}
		})

		t.Run("skipped field -", func(t *testing.T) {
			type Item struct {
				First  string
				Second int `csvplus:"-"`
			}
			data := []byte("First,Second\na,1\nb,2")
			var items []Item
			err := csvplus.Unmarshal(data, &items)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Errorf("expected len of %d, got: %d", 2, len(items))
			}
			if items[0].Second != 0 {
				t.Errorf("expected 2, got: %d", items[0].Second)
			}
			if items[1].Second != 0 {
				t.Errorf("expected 2, got: %d", items[1].Second)
			}
		})
	})

	t.Run("column naming errors", func(t *testing.T) {
		t.Run("duplicate col name", func(t *testing.T) {
			// duplicate name so we don't expect the data to be set in either column
			type Item struct {
				First  *int
				Second *int `csvplus:"First"`
			}
			data := []byte("First,Second\na,1\nb,2")
			var items []Item
			err := csvplus.Unmarshal(data, &items)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Errorf("expected len of %d, got: %d", 2, len(items))
			}
			if items[0].First != nil {
				t.Errorf("expected nil, got: %v", items[0].First)
			}
			if items[0].Second != nil {
				t.Errorf("expected 2, got: %d", items[1].First)
			}
		})
	})
}

func TestUnmarshalReader(t *testing.T) {
	type Item struct {
		First  string
		Second int
	}
	data := []byte("First,Second\na,1\nb,2")
	var items []Item
	buf := bytes.NewBuffer(data)
	err := csvplus.UnmarshalReader(buf, &items)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Errorf("expected len of %d, got: %d", 2, len(items))
	}
	if items[0].First != "a" {
		t.Errorf("expected 'a', got: %s", items[0].First)
	}
	if items[0].Second != 1 {
		t.Errorf("expected 1, got: %d", items[0].Second)
	}
	if items[1].First != "b" {
		t.Errorf("expected 'b', got: %s", items[1].First)
	}
	if items[1].Second != 2 {
		t.Errorf("expected 2, got: %d", items[1].Second)
	}
}

func ExampleDecoder_SetCSVReader() {
	type Item struct {
		Name      string     `csvplus:"name"`
		Timestamp *time.Time `csvplus:"when" csvplusFormat:"2006-01"`
	}

	data := []byte("name|when\nRob|1999-11\nRuss|")

	// create a *csv.Reader
	r := csv.NewReader(bytes.NewReader(data))
	// modify to get the required functionality, in this case '|' separated fields
	r.Comma = '|'

	dec := csvplus.NewDecoder(bytes.NewReader(data))
	// set the csv reader to the custom reader
	dec.SetCSVReader(r)

	var items []Item
	err := dec.Decode(&items)
	if err != nil {
		panic(err)
	}

	fmt.Printf("{%s %s}\n", items[0].Name, items[0].Timestamp)
	fmt.Printf("{%s %+v}\n", items[1].Name, items[1].Timestamp)
	// Output:
	// {Rob 1999-11-01 00:00:00 +0000 UTC}
	// {Russ <nil>}
}

func TestMarshal(t *testing.T) { // nolint: gocyclo
	t.Run("no tags", func(t *testing.T) {
		type Item struct {
			First  string
			Second int
		}
		items := []Item{
			{
				"a",
				1,
			},
			{
				"b",
				2,
			},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("First,Second\na,1\nb,2\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("col tags work", func(t *testing.T) {
		type Item struct {
			First  string `csvplus:"first"`
			Second int    `csvplus:"second"`
		}
		items := []Item{
			{
				"a",
				1,
			},
			{
				"b",
				2,
			},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("first,second\na,1\nb,2\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("skip field", func(t *testing.T) {
		type Item struct {
			First  string `csvplus:"-"`
			Second int    `csvplus:"second"`
		}
		items := []Item{
			{
				"a",
				1,
			},
			{
				"b",
				2,
			},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("second\n1\n2\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("time.Time format", func(t *testing.T) {
		type Item struct {
			First time.Time `csvplusFormat:"2006-01"`
		}

		tm, _ := time.Parse("2006-01", "2010-01")

		items := []Item{
			{
				tm,
			},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("First\n2010-01\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("pointer field", func(t *testing.T) {
		type Item struct {
			First *bool
		}

		yes := true
		items := []Item{
			{
				&yes,
			},
			{},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("First\ntrue\n\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("uint", func(t *testing.T) {
		type Item struct {
			First uint
		}

		items := []Item{
			{
				10,
			},
			{},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("First\n10\n0\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("float", func(t *testing.T) {
		type Item struct {
			First float64
		}

		items := []Item{
			{
				10.05,
			},
			{},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("First\n10.05\n0\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("MarshalCSV", func(t *testing.T) {
		type Item struct {
			First *YesNoBool
		}

		yes := YesNoBool(true)
		items := []Item{
			{
				&yes,
			},
			{},
		}
		data, err := csvplus.Marshal(&items)
		if err != nil {
			t.Fatal(err)
		}
		expectedData := []byte("First\nyes\nno\n")
		if string(data) != string(expectedData) {
			t.Errorf("expected: %s, got: %s", expectedData, data)
		}
	})

	t.Run("string pointer fails", func(t *testing.T) {
		a := "not a pointer to a slice"
		_, err := csvplus.Marshal(&a)
		if err == nil {
			t.Error("expected error")
		}
		expectedPrefix := "expected slice"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error prefix, expected: '%s', got: %s", expectedPrefix, err.Error())
		}
	})

	t.Run("slice value fails", func(t *testing.T) {
		var items []string
		_, err := csvplus.Marshal(items)
		if err == nil {
			t.Error("expected error")
		}
		expectedPrefix := "non pointer"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error prefix, expected: '%s', got: %s", expectedPrefix, err.Error())
		}
	})
}
