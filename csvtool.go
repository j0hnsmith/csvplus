package csvtool

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Unmarshal sets the values from the record to the fields of the struct (v). The fields in record must be in the same
// order as the fields in the struct, the fields on the struct must be exported.
func Unmarshal(record []string, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || reflect.ValueOf(v).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}
	s := rv.Elem()
	if s.NumField() != len(record) {
		//return &FieldMismatch{s.NumField(), len(record)}
		return fmt.Errorf("field number mismatch, %d in record vs %d in struct", len(record), s.NumField(), len(record))
	}
	for i := 0; i < s.NumField(); i++ {
		if len(record[i]) == 0 {
			// empty record
			continue
		}

		f := s.Field(i)

		if f.Kind() == reflect.Ptr {
			// the field is a pointer so we create a new pointer initialised with a zero value
			val := reflect.New(f.Type().Elem())
			// set the struct field to the initialised pointer
			f.Set(val)
			// and switch f from the field to 'thing' that we actually now want to set
			f = val.Elem()
		}

		fieldName := s.Type().Field(i).Name

		switch f.Type().String() {
		case "string":
			f.SetString(record[i])
		case "int":
			ival, err := strconv.ParseInt(record[i], 10, 0)
			if err != nil {
				return errors.Wrapf(err, "error processing %s", fieldName)
			}
			f.SetInt(ival)
		case "float64":
			fval, err := strconv.ParseFloat(record[i], 64)
			if err != nil {
				return errors.Wrapf(err, "error processing %s", fieldName)
			}
			f.SetFloat(fval)
		case "float32":
			fval, err := strconv.ParseFloat(record[i], 32)
			if err != nil {
				return errors.Wrapf(err, "error processing %s", fieldName)
			}
			f.SetFloat(fval)
		case "bool":
			bval, err := strconv.ParseBool(record[i])
			if err != nil {
				return errors.Wrapf(err, "error processing %s", fieldName)
			}
			f.SetBool(bval)
		case "time.Time":
			expr := `csvtool:"format:(.+)"`
			re := regexp.MustCompile(expr)
			matches := re.FindStringSubmatch(string(s.Type().Field(i).Tag))
			if len(matches) < 2 {
				return fmt.Errorf("time.Time fields (%s) must have a struct tag that matches the format '%s', with the submatch being a valid time.Parse layout", fieldName, expr)
			}
			format := matches[1]
			if format == "time.RFC3339" {
				format = time.RFC3339
			} else if format == "time.RFC3339Nano" {
				format = time.RFC3339Nano
			}
			d, err := time.Parse(format, record[i])
			if err != nil {
				return errors.Wrapf(err, "unable to convert %s using layout %s for field %s", record[i], format, fieldName)
			}
			f.Set(reflect.ValueOf(d))

		default:
			return fmt.Errorf("unsupported type for %s: %s", fieldName, f.Type().String())
		}
	}
	return nil
}
