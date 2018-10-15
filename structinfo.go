package csvplus

import (
	"fmt"
	"reflect"
)

// structInfo stores all the field info for a single struct.
type structInfo struct {
	fields     []fieldInfo
	fieldNames map[string]bool
	colNames   map[string]bool
}

// newStructInfo returns an initialised structInfo.
func newStructInfo() structInfo {
	si := new(structInfo)
	si.fieldNames = make(map[string]bool)
	si.colNames = make(map[string]bool)
	return *si
}

// addField adds the given fieldInfo.
func (si *structInfo) addField(fi fieldInfo) error {
	// check we're good to add
	if _, found := si.fieldNames[fi.Name]; found {
		return fmt.Errorf("field already added: %s", fi.Name)
	}
	if _, found := si.colNames[fi.ColName]; found {
		return fmt.Errorf("col name already added: %s", fi.Name)
	}

	// add the field
	si.fieldNames[fi.Name] = true
	if len(fi.ColName) > 0 {
		si.colNames[fi.ColName] = true
	}
	si.fields = append(si.fields, fi)

	// keep fields sorted in csv col index as that's the order we iterate a record
	//sort.Slice(si.fields, func(i, j int) bool {
	//	return si.fields[i].ColIndex < si.fields[j].ColIndex
	//})
	return nil
}

// getStructFieldIndex returns the struct field index that maps to the given csv column index.
func (si *structInfo) getStructFieldIndex(colIndex int) (int, error) {
	if colIndex > len(si.fields)-1 {
		return 0, fmt.Errorf("colIndex too high, only %d fields registered", len(si.fields))
	}
	return si.fields[colIndex].FieldIndex, nil
}

// StructRegister is a container for all csv/struct field mappings.
type StructRegister struct {
	Fields map[bool]map[reflect.Type]structInfo
}

func NewStructRegister() StructRegister {
	sr := StructRegister{
		Fields: make(map[bool]map[reflect.Type]structInfo),
	}
	sr.Fields[true] = make(map[reflect.Type]structInfo)
	sr.Fields[false] = make(map[reflect.Type]structInfo)
	return sr
}

func (sr *StructRegister) Register(st reflect.Type, hasHeader bool, data []string) error {
	if hasHeader {
		return sr.registerStructWithHeaders(st, data)
	}
	return sr.registerStructWithoutHeaderRow(st, len(data))
}

// registerStructWithoutHeader maps csv record columns to struct field indices for csv data that doesn't have a header
// row
func (sr *StructRegister) registerStructWithoutHeaderRow(st reflect.Type, recordLen int) error {
	if sr.exists(false, st) {
		return nil
	}

	var colIndex int
	for i := 0; i < st.Elem().NumField(); i++ {
		sf := st.Field(i)
		tag, found := sf.Tag.Lookup("csvplus")
		if found && tag == "-" {
			continue
		}

		fi := fieldInfo{
			Name:       sf.Name,
			FieldIndex: i,
			ColName:    "",
			ColIndex:   colIndex,
		}

		if sf.Type.String() == "time.Time" || sf.Type.String() == "*time.Time" {
			fi.Format = sf.Tag.Get("csvplusFormat")
		}

		if err := sr.storeFieldInfo(false, st, fi); err != nil {
			return err
		}

		colIndex++
	}

	// TODO: check number of registered fields is same as recordLen

	return nil
}

func (sr *StructRegister) registerStructWithHeaders(st reflect.Type, headers []string) error {
	if sr.exists(true, st) {
		return nil
	}

	headerColumnMap := make(map[string]int)
	for i, header := range headers {
		headerColumnMap[header] = i
	}

	var fi fieldInfo
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		tag := sf.Tag.Get("csvplus")
		if tag == "-" {
			continue
		}

		fi = fieldInfo{
			Name:       sf.Name,
			FieldIndex: i,
		}

		fi.ColName = tag
		if fi.ColName == "" {
			fi.ColName = sf.Name
		}
		colIndex, found := headerColumnMap[fi.ColName]
		if !found {
			return fmt.Errorf("unable to find column %s in csv header row", fi.ColName)
		}
		fi.ColIndex = colIndex

		if sf.Type.String() == "time.Time" || sf.Type.String() == "*time.Time" {
			fi.Format = sf.Tag.Get("csvplusFormat")
		}

		if err := sr.storeFieldInfo(true, st, fi); err != nil {
			return err
		}
	}

	// TODO: check field count is same as len(headers)

	return nil
}

// storeFieldInfo gets the struct field index that maps to the column field index.
func (sr *StructRegister) storeFieldInfo(hasHeader bool, rt reflect.Type, fi fieldInfo) error {
	si, found := sr.Fields[hasHeader][rt]
	if !found {
		si = newStructInfo()
		sr.Fields[hasHeader][rt] = si
	}
	err := si.addField(fi)
	if err != nil {
		return err
	}
	sr.Fields[hasHeader][rt] = si
	return nil
}

func (sr *StructRegister) GetStructFieldIndex(hasHeader bool, rt reflect.Type, colIndex int) (int, error) {
	si, found := sr.Fields[hasHeader][rt]
	if !found {
		return 0, fmt.Errorf("unregistered type: %s", rt)
	}
	return si.getStructFieldIndex(colIndex)
}

func (sr *StructRegister) exists(hasHeader bool, rt reflect.Type) bool {
	_, found := sr.Fields[hasHeader][rt]
	return found
}

// fieldInfo represents a field in a struct with tags parsed and stuct/csv record indices mapped.
type fieldInfo struct {
	Name       string
	FieldIndex int
	ColName    string // only populated for csv data with header rows
	ColIndex   int
	Format     string // only populated for time.Time fields
}

var DefaultStructRegister = NewStructRegister()
