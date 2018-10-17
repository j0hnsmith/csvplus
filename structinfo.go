package csvplus

import (
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// structInfo stores all the field info for a single struct.
type structInfo struct {
	fields           []fieldInfo
	headerColIndices []int
}

// StructRegister is a container for all csv/struct field mappings.
type StructRegister struct {
	Fields map[reflect.Type]structInfo
}

func NewStructRegister() StructRegister {
	sr := StructRegister{
		Fields: make(map[reflect.Type]structInfo),
	}
	return sr
}

func (sr *StructRegister) Register(st reflect.Type, headers []string) {
	if sr.exists(st) {
		return
	}

	headersMap := make(map[string]int)
	for i, header := range headers {
		headersMap[header] = i
	}
	fieldCounts := make(map[string]int)

	ColNameToFieldInfo := make(map[string]fieldInfo)

	// iterate struct tags to extract all names
	var fi fieldInfo
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)

		fi = fieldInfo{
			Name:       sf.Name,
			FieldIndex: i,
		}

		tag := sf.Tag.Get("csvplus")
		tokens := strings.Split(tag, ",")
		if len(tokens) > 1 && tokens[1] == "omitempty" {
			fi.OmitEmpty = true
		}
		tag = tokens[0]

		switch tag {
		case "":

			var found bool
			var colIndex int

			if colIndex, found = headersMap[fi.Name]; found {
				fi.ColName = fi.Name
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
			}

			// this field isn't mapped to a header row
			continue

		case "-":
			fi.SkipField = true // used only for marshalling, if at all, maybe remove later

		default:
			fi.ColName = tag
			if colIndex, found := headersMap[fi.ColName]; found {
				fi.ColIndex = colIndex
				break
			}
			continue
		}

		if sf.Type.String() == "time.Time" || sf.Type.String() == "*time.Time" {
			fi.Format = sf.Tag.Get("csvplusFormat")
			if fi.Format == "" {
				fi.Format = time.RFC3339
			}
		}

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

	sr.Fields[st] = structInfo{
		fields:           fieldsToStore,
		headerColIndices: headerColIndices,
	}
}

func (sr *StructRegister) exists(rt reflect.Type) bool {
	_, found := sr.Fields[rt]
	return found
}

// fieldInfo represents a field in a struct with tags parsed and stuct/csv record indices mapped.
type fieldInfo struct {
	Name       string
	FieldIndex int
	ColName    string // only populated for csv data with header rows
	ColIndex   int
	Format     string // only populated for time.Time fields
	SkipField  bool
	OmitEmpty  bool
}

var DefaultStructRegister = NewStructRegister()
