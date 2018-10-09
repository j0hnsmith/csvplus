package csvtool_test

import (
	"testing"
	"time"

	"github.com/j0hnsmith/csvtool"
)

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

	t.Run("int", func(t *testing.T) {
		record := []string{"1"}
		s := new(Int)
		err := csvtool.Unmarshal(record, s)
		if err != nil {
			t.Error(err)
		}
		if s.Field != 1 {
			t.Error("expected 1")
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
			t.Error(err)
		}
		if s.Field != float32(1.0) {
			t.Error("expected 1.0")
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
	})
}