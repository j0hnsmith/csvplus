package csvtool_test

import (
	"strings"
	"testing"
	"time"

	"github.com/j0hnsmith/csvtool"
)

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
	Field time.Time `csvtool:"format:time.RFC3339Nano"`
}

type DateTimeRFC struct {
	Field time.Time `csvtool:"format:time.RFC3339"`
}

type DateTimeFormat struct {
	Field time.Time `csvtool:"format:2006-01"`
}

type DateTimeNoTag struct {
	Field time.Time
}

type MyString string

type Custom struct {
	Field MyString
}

func TestUnmarshal(t *testing.T) {

	t.Run("string pointer fails", func(t *testing.T) {
		a := "not a pointer to a struct"
		record := []string{"1"}
		err := csvtool.Unmarshal(record, &a)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("struct fails", func(t *testing.T) {
		a := Int{Field: 1}
		record := []string{"1"}
		err := csvtool.Unmarshal(record, a)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("struct fields length mismatch", func(t *testing.T) {
		a := Int{Field: 1}
		record := []string{"1", "2"}
		err := csvtool.Unmarshal(record, &a)
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
		err := csvtool.Unmarshal(record, s)
		if err != nil {
			t.Fatal(err)
		}
		if s.Field != 0 {
			t.Error("expected 0 (empty value)")
		}
	})

	t.Run("unsupported field type", func(t *testing.T) {
		record := []string{"foo"}
		s := new(Custom)
		err := csvtool.Unmarshal(record, s)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "unsupported type for Field: csvtool_test.MyString"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error prefix, expected: '%s', got: %s", expectedPrefix, err.Error())
		}
	})

	t.Run("int", func(t *testing.T) {
		record := []string{"1"}
		s := new(Int)
		err := csvtool.Unmarshal(record, s)
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
		err := csvtool.Unmarshal(record, s)
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
		err := csvtool.Unmarshal(record, s)
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
		err := csvtool.Unmarshal(record, s)
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
		err := csvtool.Unmarshal(record, s)
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
		err := csvtool.Unmarshal(record, s)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "unable to convert foo to float32 in field Field"
		if !strings.HasPrefix(err.Error(), expectedPrefix) {
			t.Errorf("wrong error, expected: '%s'", expectedPrefix)
		}
	})

	t.Run("float64", func(t *testing.T) {
		record := []string{"1.0"}
		s := new(Float64)
		err := csvtool.Unmarshal(record, s)
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
		err := csvtool.Unmarshal(record, s)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedPrefix := "unable to convert foo to float64 in field Field"
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
				err := csvtool.Unmarshal(record, s)
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
		err := csvtool.Unmarshal(record, s)
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
			err := csvtool.Unmarshal(record, s)
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
			err := csvtool.Unmarshal(record, s)
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
			err := csvtool.Unmarshal(record, s)
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
			err := csvtool.Unmarshal(record, s)
			if err == nil {
				t.Error("expected error because time.Time field without a layout in a struct tag")
			}
		})

		t.Run("invalid format", func(t *testing.T) {
			dt := time.Now().UTC()
			dts := dt.Format("invalid format")
			record := []string{dts}
			s := new(DateTimeRFC)
			err := csvtool.Unmarshal(record, s)
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
