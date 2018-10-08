package csvtool_test

import (
	"github.com/j0hnsmith/csvtool"
	"testing"
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
}
