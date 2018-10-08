package csvtool_test

import (
	"github.com/j0hnsmith/csvtool"
	"testing"
)

type Int struct {
	Field int
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
}
