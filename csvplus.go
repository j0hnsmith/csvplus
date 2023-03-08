// Package csvplus marshals/unmarshals CSV data directly from/to slice of structs, types are converted to those
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

// UnmarshalWithoutHeader is used to unmarshal csv data that doesn't have a header row.
func UnmarshalWithoutHeader(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	return NewDecoder(buf).UseHeader(false).Decode(v)
}

// Unmarshaler is the interface implemented by types that can unmarshal a csv record of themselves.
type Unmarshaler interface {
	UnmarshalCSV(string) error
}

// A Decoder reads and decodes CSV records from an input stream. Useful if your data doesn't have a header row.
type Decoder struct {
	headerPassed  bool
	withoutHeader bool
	csvReader     *csv.Reader
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

// UseHeader sets whether the first data row is a header row.
func (dec *Decoder) UseHeader(b bool) *Decoder {
	dec.withoutHeader = !b
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

	var row int
	for {
		record, err := dec.csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error reading csv reader")
		}

		if !dec.headerPassed {
			fis = getFieldInfo(structType, dec.withoutHeader, record)
			dec.headerPassed = true
			if !dec.withoutHeader {
				row++
				continue
			}
		}

		structPZeroValue := reflect.New(structType)

		if err := dec.unmarshalRecord(row, record, structPZeroValue.Interface(), fis); err != nil {
			return err
		}

		containerValue.Set(reflect.Append(containerValue, structPZeroValue.Elem()))
		row++
	}

	return nil
}

// unmarshalRecord sets the values from a single CSV record to the (exported) fields of the struct v.
func (dec *Decoder) unmarshalRecord(row int, record []string, v interface{}, fis []fieldInfo) error { // nolint: gocyclo
	rv := reflect.ValueOf(v)
	s := rv.Elem()

	for _, fi := range fis {
		if fi.SkipField || fi.ColName == "" {
			continue
		}

		if (len(record) - 1) < fi.ColIndex {
			return errors.Errorf("not enough columns in csv data (row %d)", row)
		}

		recVal := record[fi.ColIndex]
		f := s.FieldByName(fi.Name)

		// if field implements csvplus.Unmarshaler use that
		if f.Type().Implements(csvUnmarshalerType) {
			p := reflect.New(f.Type().Elem())
			uc := p.Interface().(Unmarshaler)
			err := uc.UnmarshalCSV(recVal)
			if err != nil {
				return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, errors.Wrapf(err, "%s.UnmarshalCSV()", fi.Name))
			}
			f.Set(reflect.ValueOf(uc))
			continue

		} else if reflect.PtrTo(f.Type()).Implements(csvUnmarshalerType) {

			p := reflect.New(f.Type())
			uc := p.Interface().(Unmarshaler)
			err := uc.UnmarshalCSV(recVal)
			if err != nil {
				return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, errors.Wrapf(err, "%s.UnmarshalCSV()", fi.Name))
			}
			f.Set(reflect.ValueOf(uc).Elem())
			continue
		}

		if recVal == "" {
			// no data to store in field
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
				return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, errors.Wrapf(err, "strconv.ParseInt"))
			}
			f.SetInt(ival)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ival, err := strconv.ParseUint(recVal, 10, 64)
			if err != nil || f.OverflowUint(ival) {
				return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, errors.Wrapf(err, "strconv.ParseUint"))
			}
			f.SetUint(ival)
		case reflect.Float32, reflect.Float64:
			fval, err := strconv.ParseFloat(recVal, 64)
			if err != nil || f.OverflowFloat(fval) {
				return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, errors.Wrapf(err, "strconv.ParseFloat"))
			}
			f.SetFloat(fval)
		case reflect.Bool:
			bval, err := strconv.ParseBool(recVal)
			if err != nil {
				return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, errors.Wrapf(err, "strconv.ParseBool"))
			}
			f.SetBool(bval)
		case reflect.Struct:
			if f.Type().String() == timeType {
				d, err := time.Parse(fi.Format, recVal)
				if err != nil {
					return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, errors.Wrapf(err, "time.Parse %s", fi.Format))
				}
				f.Set(reflect.ValueOf(d))
				break
			}
			fallthrough

		default:
			return newUnmarshalError(fi.ColName, fi.ColIndex, row, recVal, fmt.Errorf("unsupported type %s", f.Type().String()))
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

// MarshalWithoutHeader writes csv data without a header row.
func MarshalWithoutHeader(v interface{}) ([]byte, error) {
	var buf bytes.Buffer

	err := NewEncoder(&buf).UseHeader(false).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// An Encoder writes csv data from a list of struct.
type Encoder struct {
	csvWriter        *csv.Writer
	withoutHeaderRow bool
	encRegister      encRegister
}

// NewEncoder returns an initialised Encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		csvWriter:   csv.NewWriter(w),
		encRegister: defaultEncRegister,
	}
}

// SetCSVWriter allows for using a csv.Writer with custom config (eg | field separator instead of ,).
func (enc *Encoder) SetCSVWriter(r *csv.Writer) *Encoder {
	enc.csvWriter = r
	return enc
}

// UseHeader sets whether to add a header row to the csv data.
func (enc *Encoder) UseHeader(v bool) *Encoder {
	enc.withoutHeaderRow = !v
	return enc
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

	if !enc.withoutHeaderRow {
		err := enc.csvWriter.Write(enc.encRegister.GetEncodeHeaders(st))
		if err != nil {
			return errors.Wrap(err, "unable to write header row")
		}
	}

	containerValue := rv.Elem()

	var record []string
	for i := 0; i < containerValue.Len(); i++ {
		record = nil
		sv := containerValue.Index(i)

		for _, fieldIndex := range enc.encRegister.GetEncodeIndices(st) {
			fv := sv.Field(fieldIndex)

			var m Marshaler
			if fv.Type().Implements(csvMarshalerType) {
				m = fv.Interface().(Marshaler)
			} else if reflect.PtrTo(fv.Type()).Implements(csvMarshalerType) {
				m = fv.Addr().Interface().(Marshaler)
			}
			if m != nil {
				b, err := m.MarshalCSV()
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
	return enc.csvWriter.Error()
}

type UnmarhsalError struct {
	Column string
	Row    int
	Value  string
	RawErr error
}

func newUnmarshalError(colName string, colIndex, row int, value string, err error) UnmarhsalError {
	if colName == "" {
		// no header row, we only have index
		colName = fmt.Sprintf("col idx %d", colIndex)
	}
	return UnmarhsalError{
		Column: colName,
		Row:    row,
		Value:  value,
		RawErr: err,
	}
}

func (um UnmarhsalError) Error() string {
	return fmt.Sprintf("col: %s, row: %d, val: %s, err: %s", um.Column, um.Row, um.Value, um.RawErr.Error())
}
