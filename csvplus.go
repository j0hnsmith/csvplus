// Unmarshal CSV data directly into a list of structs, types are converted to those
// matching the fields on the struct. Struct fields must be in the same order as the records in the CSV data.
package csvplus

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Unmarshal parses the csv encoded data and stores the result in the slice pointed to by v.
// The number of records in row of the csv data must match the number of exported fields in the struct.
// For common types (eg int, bool, float64...) a standard conversion from a string is applied. If a type implements
// the Unmarshaler interface, that will be used to unmarshal the record instead.
// This function assumes the csv data has a header row (which is skipped), see the Decoder type if your data doesn't
// have a header row.
func Unmarshal(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	return NewDecoder(buf, true).Decode(v)
}

// UnmarshalReader is the same as Unmarshal but takes it's input data from an io.Reader.
func UnmarshalReader(r io.Reader, v interface{}) error {
	return NewDecoder(r, true).Decode(v)
}

// Unmarshaler is the interface implemented by types that can unmarshal a csv record of themselves.
type Unmarshaler interface {
	UnmarshalCSV(string) error
}

// A Decoder reads and decodes CSV records from an input stream. Useful if your data doesn't have a header row.
type Decoder struct {
	r            io.Reader
	HasHeaderRow bool
	headerPassed bool
}

// NewDecoder reads and decodes CSV records from r.
func NewDecoder(r io.Reader, hasHeaderRow bool) *Decoder {
	return &Decoder{
		r:            r,
		HasHeaderRow: hasHeaderRow,
	}
}

// Decode reads reads csv recorder into v.
func (dec *Decoder) Decode(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("non pointer %s", reflect.ValueOf(v).Type())
	}

	value := reflect.ValueOf(v).Elem()
	csvReader := csv.NewReader(dec.r)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error reading csv reader")
		}
		if !dec.headerPassed {
			dec.headerPassed = true
			continue
		}

		elem := reflect.New(reflect.TypeOf(v).Elem().Elem())

		if err := UnmarshalRecord(record, elem.Interface()); err != nil {
			return err
		}

		value.Set(reflect.Append(value, elem.Elem()))
	}

	return nil
}

// UnmarshalRecord sets the values from a single CSV record to the fields of the struct v. The fields in record must be
// in the same order as the fields in the struct, the fields on the struct must be exported.
func UnmarshalRecord(record []string, v interface{}) error { // nolint: gocyclo
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || reflect.ValueOf(v).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}
	s := rv.Elem()
	if s.NumField() != len(record) {
		return fmt.Errorf("field number mismatch, %d in record vs %d in struct", len(record), s.NumField())
	}
	for i := 0; i < s.NumField(); i++ {
		if len(record[i]) == 0 {
			// empty record
			continue
		}

		f := s.Field(i)
		fieldName := s.Type().Field(i).Name

		// if field implements csvplus.Unmarshaler use that
		if f.Type().Implements(csvUnmarshalerType) {
			p := reflect.New(f.Type().Elem())
			uc := p.Interface().(Unmarshaler)
			err := uc.UnmarshalCSV(record[i])
			if err != nil {
				return errors.Wrapf(err, "error calling %s.UnmarshalCSV()", fieldName)
			}
			f.Set(reflect.ValueOf(uc))
			continue

		} else if reflect.PtrTo(f.Type()).Implements(csvUnmarshalerType) {

			p := reflect.New(f.Type())
			uc := p.Interface().(Unmarshaler)
			err := uc.UnmarshalCSV(record[i])
			if err != nil {
				return errors.Wrapf(err, "error calling %s.UnmarshalCSV()", fieldName)
			}
			f.Set(reflect.ValueOf(uc).Elem())
			continue
		}

		if f.Kind() == reflect.Ptr {
			// the field is a pointer so we create a new pointer initialised with a zero value
			val := reflect.New(f.Type().Elem())
			// set the struct field to the initialised pointer
			f.Set(val)
			// and switch f from the field to 'thing' that we actually now want to set
			f = val.Elem()
		}

		switch f.Kind() {
		case reflect.String:
			f.SetString(record[i])
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ival, err := strconv.ParseInt(record[i], 10, 64)
			if err != nil || f.OverflowInt(ival) {
				return errors.Wrapf(err, "unable to convert %s to int in field %s", record[i], fieldName)
			}
			f.SetInt(ival)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ival, err := strconv.ParseUint(record[i], 10, 64)
			if err != nil || f.OverflowUint(ival) {
				return errors.Wrapf(err, "unable to convert %s to uint in field %s", record[i], fieldName)
			}
			f.SetUint(ival)
		case reflect.Float32, reflect.Float64:
			fval, err := strconv.ParseFloat(record[i], 64)
			if err != nil || f.OverflowFloat(fval) {
				return errors.Wrapf(err, "unable to convert %s to float in field %s", record[i], fieldName)
			}
			f.SetFloat(fval)
		case reflect.Bool:
			bval, err := strconv.ParseBool(record[i])
			if err != nil {
				return errors.Wrapf(err, "unable to convert %s to bool in field %s", record[i], fieldName)
			}
			f.SetBool(bval)
		case reflect.Struct:
			if f.Type().String() == "time.Time" {
				expr := `csvplus:"format:(.+)"`
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
					return errors.Wrapf(err, "invalid layout format for field %s", fieldName)
				}
				f.Set(reflect.ValueOf(d))
				break
			}
			fallthrough

		default:
			return fmt.Errorf("unsupported type for %s: %s", fieldName, f.Type().String())
		}
	}

	return nil
}

var csvUnmarshalerType = reflect.TypeOf(new(Unmarshaler)).Elem()
