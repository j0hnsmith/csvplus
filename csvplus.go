// Package csvplus unmarshals CSV data directly into a slice of structs, types are converted to those
// matching the fields on the struct. Layout strings can be provided via struct tags for time.Time fields.
package csvplus

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
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
	return NewDecoder(buf).Decode(v)
}

// UnmarshalReader is the same as Unmarshal but takes it's input data from an io.Reader.
func UnmarshalReader(r io.Reader, v interface{}) error {
	return NewDecoder(r).Decode(v)
}

// Unmarshaler is the interface implemented by types that can unmarshal a csv record of themselves.
type Unmarshaler interface {
	UnmarshalCSV(string) error
}

// A Decoder reads and decodes CSV records from an input stream. Useful if your data doesn't have a header row.
type Decoder struct {
	headerPassed bool
	csvReader    *csv.Reader
}

// NewDecoder reads and decodes CSV records from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		csvReader: csv.NewReader(r),
	}
}

// SetCSVReader allows for using a custom csv.Reader (eg | field separator instead of ,).
func (dec *Decoder) SetCSVReader(r *csv.Reader) *Decoder {
	dec.csvReader = r
	return dec
}

func (dec *Decoder) UseHeader(b bool) *Decoder {
	dec.withHeader = b
	return dec
}

// Decode reads reads csv recorder into v.
func (dec *Decoder) Decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("non pointer %s", rt)
	}
	if rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("expected slice to store data in, got %s", rv.Elem().Type())
	}

	containerValue := rv.Elem()
	structType := rt.Elem().Elem()
	var fis []fieldInfo

	for {
		record, err := dec.csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error reading csv reader")
		}

		if !dec.headerPassed {
			// register struct
			fis = getFieldInfo(structType, record)
			dec.headerPassed = true
			continue
		}

		structPZeroValue := reflect.New(structType)

		if err := dec.unmarshalRecord(record, structPZeroValue.Interface(), fis); err != nil {
			return err
		}

		containerValue.Set(reflect.Append(containerValue, structPZeroValue.Elem()))
	}

	return nil
}

// unmarshalRecord sets the values from a single CSV record to the (exported) fields of the struct v.
func (dec *Decoder) unmarshalRecord(record []string, v interface{}, fis []fieldInfo) error { // nolint: gocyclo
	rv := reflect.ValueOf(v)
	s := rv.Elem()

	for _, fi := range fis {
		if fi.SkipField || fi.ColName == "" {
			continue
		}

		recVal := record[fi.ColIndex]
		if recVal == "" {
			// no data to store in field
			continue
		}

		f := s.FieldByName(fi.Name)

		// if field implements csvplus.Unmarshaler use that
		if f.Type().Implements(csvUnmarshalerType) {
			p := reflect.New(f.Type().Elem())
			uc := p.Interface().(Unmarshaler)
			err := uc.UnmarshalCSV(recVal)
			if err != nil {
				return errors.Wrapf(err, "error calling %s.UnmarshalCSV()", fi.Name)
			}
			f.Set(reflect.ValueOf(uc))
			continue

		} else if reflect.PtrTo(f.Type()).Implements(csvUnmarshalerType) {

			p := reflect.New(f.Type())
			uc := p.Interface().(Unmarshaler)
			err := uc.UnmarshalCSV(recVal)
			if err != nil {
				return errors.Wrapf(err, "error calling %s.UnmarshalCSV()", fi.Name)
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
			f.SetString(recVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ival, err := strconv.ParseInt(recVal, 10, 64)
			if err != nil || f.OverflowInt(ival) {
				return errors.Wrapf(err, "unable to convert %s to int in field %s", recVal, fi.Name)
			}
			f.SetInt(ival)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ival, err := strconv.ParseUint(recVal, 10, 64)
			if err != nil || f.OverflowUint(ival) {
				return errors.Wrapf(err, "unable to convert %s to uint in field %s", recVal, fi.Name)
			}
			f.SetUint(ival)
		case reflect.Float32, reflect.Float64:
			fval, err := strconv.ParseFloat(recVal, 64)
			if err != nil || f.OverflowFloat(fval) {
				return errors.Wrapf(err, "unable to convert %s to float in field %s", recVal, fi.Name)
			}
			f.SetFloat(fval)
		case reflect.Bool:
			bval, err := strconv.ParseBool(recVal)
			if err != nil {
				return errors.Wrapf(err, "unable to convert %s to bool in field %s", recVal, fi.Name)
			}
			f.SetBool(bval)
		case reflect.Struct:
			if f.Type().String() == timeType {
				d, err := time.Parse(fi.Format, recVal)
				if err != nil {
					return errors.Wrapf(err, "invalid layout format for field %s", fi.Name)
				}
				f.Set(reflect.ValueOf(d))
				break
			}
			fallthrough

		default:
			return fmt.Errorf("unsupported type for %s: %s", fi.Name, f.Type().String())
		}
	}

	return nil
}

var csvUnmarshalerType = reflect.TypeOf(new(Unmarshaler)).Elem()
var csvMarshalerType = reflect.TypeOf(new(Marshaler)).Elem()

// Marshaler is the interface implemented by types that can marshal a csv value (string) of themselves.
type Marshaler interface {
	MarshalCSV() ([]byte, error)
}

// Marshal marshals v into csv data.
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalWriter marshals v into the given writer.
func MarshalWriter(v interface{}, w io.Writer) error {
	return NewEncoder(w).Encode(v)
}

// An Encoder writes csv data from a list of struct.
type Encoder struct {
	headerPassed bool
	csvWriter    *csv.Writer
	encRegister  encRegister
}

// NewEncoder returns an initialised Encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		csvWriter:   csv.NewWriter(w),
		encRegister: defaultEncRegister,
	}
}

// SetCSVWriter allows for using a csv.Writer with custom config (eg | field separator instead of ,).
func (enc *Encoder) SetCSVWriter(r *csv.Writer) {
	enc.csvWriter = r
}

// Encode encodes v into csv data.
func (enc *Encoder) Encode(v interface{}) error { // nolint: gocyclo
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("non pointer %s", rt)
	}
	if rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("expected slice, got %s", rv.Elem().Type())
	}

	st := reflect.TypeOf(v).Elem().Elem()
	enc.encRegister.Register(st)

	err := enc.csvWriter.Write(enc.encRegister.GetEncodeHeaders(st))
	if err != nil {
		return errors.Wrap(err, "unable to write header row")
	}

	containerValue := rv.Elem()

	var record []string
	for i := 0; i < containerValue.Len(); i++ {
		record = nil
		sv := containerValue.Index(i)

		for _, fieldIndex := range enc.encRegister.GetEncodeIndices(st) {
			fv := sv.Field(fieldIndex)

			if fv.Type().Implements(csvMarshalerType) {
				u := fv.Interface().(Marshaler)
				b, err := u.MarshalCSV()
				if err != nil {
					return err
				}
				record = append(record, string(b))
				continue
			}

			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					record = append(record, "")
					continue
				}

				// dereference
				fv = fv.Elem()
			}

			switch fv.Kind() {
			case reflect.String:
				record = append(record, fv.String())
				continue
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				record = append(record, strconv.Itoa(int(fv.Int())))
				continue
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				record = append(record, strconv.Itoa(int(fv.Uint())))
				continue
			case reflect.Float32, reflect.Float64:
				// TODO: consider fmt.Sprintf("%.6f", fv.Float()), this could come from a struct tag
				record = append(record, strconv.FormatFloat(fv.Float(), 'f', -1, 64))
				continue
			case reflect.Bool:
				record = append(record, strconv.FormatBool(fv.Bool()))
				continue
			case reflect.Struct:
				if fv.Type().String() == timeType {
					t := fv.Interface().(time.Time)
					record = append(record, t.Format(enc.encRegister.Fields[st].fields[fieldIndex].Format))
					continue
				}

				record = append(record, fv.String())
				continue
			}
		}

		if err := enc.csvWriter.Write(record); err != nil {
			return err
		}
	}

	enc.csvWriter.Flush()
	if err := enc.csvWriter.Error(); err != nil {
		return err
	}

	return nil
}
