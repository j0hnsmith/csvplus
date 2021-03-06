package csvplus

import (
	"reflect"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

const timeType = "time.Time"
const timeTypePtr = "*time.Time"

// structInfo stores all the field info for a single struct.
type structInfo struct {
	fields       map[int]fieldInfo
	fieldIndices []int
	headerRow    []string // only used when marshaling
}

func newStructInfo() *structInfo {
	return &structInfo{
		fields: make(map[int]fieldInfo),
	}
}

// getTimeFormat gets a suitable time.Parse layout from a csvplusFormat struct tag, defaults to time.RFC3339 if no
// format is found.
func getTimeFormat(sf reflect.StructField) (format string) {
	if sf.Type.String() == timeType || sf.Type.String() == timeTypePtr {
		format = sf.Tag.Get("csvplusFormat")
		switch format {
		case "", "time.RFC3339":
			format = time.RFC3339
		case "time.RFC3339Nano":
			format = time.RFC3339Nano
		}
	}
	return format
}

// Register maps columns in the csv data to struct fields.
func getFieldInfo(st reflect.Type, withoutHeader bool, header []string) []fieldInfo {
	headersMap := make(map[string]int)
	for i, header := range header {
		headersMap[header] = i
	}
	fieldCounts := make(map[string]int)

	ColNameToFieldInfo := make(map[string]fieldInfo)
	var skipCount int

	// iterate struct tags to extract all names
	var fi fieldInfo
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)

		fi = fieldInfo{
			Name:       sf.Name,
			FieldIndex: i,
		}

		tag := sf.Tag.Get("csvplus")

		switch tag {
		case "":
			var found bool
			var colIndex int

			if colIndex, found = headersMap[fi.Name]; found {
				if withoutHeader {
					fi.ColName = strconv.Itoa(i)
				} else {
					fi.ColName = fi.Name
				}
				fi.ColIndex = colIndex
				break
			}

			// try again with first char lowercased
			r, n := utf8.DecodeRuneInString(fi.Name)
			lowerName := string(unicode.ToLower(r)) + fi.Name[n:]
			if colIndex, found := headersMap[lowerName]; found {
				fi.ColName = lowerName
				fi.ColIndex = colIndex
				break
			} else if withoutHeader {
				fi.ColName = strconv.Itoa(i)
				fi.ColIndex = i - skipCount
				break
			}

			// this field isn't mapped to a header row
			continue

		case "-":
			fi.SkipField = true // used only for marshalling, if at all, maybe remove later
			fi.ColName = "-"
			skipCount++
		default:
			fi.ColName = tag
			if colIndex, found := headersMap[fi.ColName]; found {
				fi.ColIndex = colIndex
				break
			}
			continue
		}

		fi.Format = getTimeFormat(sf)

		fieldCounts[fi.ColName]++
		ColNameToFieldInfo[fi.ColName] = fi
	}

	var headerColIndices []int
	var fieldsToStore []fieldInfo
	for colName, seenCount := range fieldCounts {
		if seenCount > 1 {
			// multiple fields map to same name, ignore
			continue
		}
		fi := ColNameToFieldInfo[colName]
		fieldsToStore = append(fieldsToStore, fi)
		if fi.ColName != "" {
			headerColIndices = append(headerColIndices, fi.ColIndex)
		}
	}

	return fieldsToStore
}

// fieldInfo represents a field in a struct with tags parsed and stuct/csv record indices mapped.
type fieldInfo struct {
	Name       string
	FieldIndex int
	ColName    string // only populated for csv data with header rows
	ColIndex   int
	Format     string // only populated for time.Time fields
	SkipField  bool
}

// encRegister is a cache for data needed to marshal, since a
type encRegister struct {
	Fields map[reflect.Type]structInfo
}

// newEncRegister returns an initialised encRegister.
func newEncRegister() encRegister {
	return encRegister{
		Fields: make(map[reflect.Type]structInfo),
	}
}

// defaultEncRegister is a encRegister singleton since there only needs to be one.
var defaultEncRegister = newEncRegister()

// Register introspects and stores the necessary data to marshal csv data.
func (er *encRegister) Register(st reflect.Type) {
	if _, found := er.Fields[st]; found {
		return
	}

	si := newStructInfo()
	for i := 0; i < st.NumField(); i++ {
		fi := fieldInfo{FieldIndex: i}
		sf := st.Field(i)
		fi.ColName = sf.Tag.Get("csvplus")
		switch fi.ColName {
		case "-":
			fi.SkipField = true
		case "":
			fi.ColName = sf.Name
		}

		fi.Name = sf.Name
		if !fi.SkipField {
			fi.ColIndex = i
		}

		if sf.Type.String() == timeType || sf.Type.String() == timeTypePtr {
			fi.Format = getTimeFormat(sf)
		}

		si.fields[fi.FieldIndex] = fi

		if !fi.SkipField {
			si.fieldIndices = append(si.fieldIndices, fi.ColIndex)
			si.headerRow = append(si.headerRow, fi.ColName)
		}
	}

	er.Fields[st] = *si
}

// GetEncodeIndices returns the struct field indices needed to marshal csv data for this type.
func (er *encRegister) GetEncodeIndices(st reflect.Type) []int {
	si, found := er.Fields[st]
	if !found {
		return nil
	}
	return si.fieldIndices
}

// GetEncodeHeaders returns the values for the csv header row for this type.
func (er *encRegister) GetEncodeHeaders(st reflect.Type) []string {
	si, found := er.Fields[st]
	if !found {
		return nil
	}
	return si.headerRow
}
