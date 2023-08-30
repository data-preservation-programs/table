package table

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

type Table struct {
	tab              string
	headerRowColor   *color.Color
	firstColumnColor *color.Color
	padding          int
	verbose          bool
}

type Option func(*Table)

func WithTab(p string) Option {
	return func(t *Table) {
		t.tab = p
	}
}

func WithVerbose() Option {
	return func(t *Table) {
		t.verbose = true
	}
}

func WithHeaderRowColor(c *color.Color) Option {
	return func(t *Table) {
		t.headerRowColor = c
	}
}

func WithFirstColumnColor(c *color.Color) Option {
	return func(t *Table) {
		t.firstColumnColor = c
	}
}

func WithPaddingSize(p int) Option {
	return func(t *Table) {
		t.padding = p
	}
}

func New(opts ...Option) *Table {
	color.New()
	t := &Table{
		padding:          2,
		tab:              "    ",
		headerRowColor:   color.New(color.FgGreen, color.Underline),
		firstColumnColor: color.New(color.FgYellow),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

type cell struct {
	value string
	color *color.Color
	width int
}

type headerRow = []*cell

type dataRow struct {
	cells     []*cell
	subTables []table
}

type table struct {
	name      string
	nameColor *color.Color
	header    headerRow
	data      []dataRow
}

type tag struct {
	header      string
	headerColor *color.Color
	color       *color.Color
	format      string
	verbose     bool
	ignore      bool
	expand      bool
}

func parseTag(str string) *tag {
	t := tag{
		format: "%v",
	}
	if str == "" {
		return &t
	}
	for _, s := range strings.Split(str, ";") {
		if s == "-" {
			t.ignore = true
			continue
		}
		if s == "verbose" {
			t.verbose = true
			continue
		}
		if s == "expand" {
			t.expand = true
			continue
		}
		i := strings.IndexAny(s, ":")
		if i == -1 {
			return &tag{ignore: true}
		}
		key := s[:i]
		value := s[i+1:]
		switch key {
		case "header":
			t.header = value
		case "headerColor":
			var attributes []color.Attribute
			for _, c := range strings.Split(value, ",") {
				attr, err := strconv.Atoi(c)
				if err != nil {
					return &tag{ignore: true}
				}
				attributes = append(attributes, color.Attribute(attr))
			}
			t.headerColor = color.New(attributes...)
		case "color":
			var attributes []color.Attribute
			for _, c := range strings.Split(value, ",") {
				attr, err := strconv.Atoi(c)
				if err != nil {
					return &tag{ignore: true}
				}
				attributes = append(attributes, color.Attribute(attr))
			}
			t.color = color.New(attributes...)
		case "format":
			t.format = value
		default:
			return &tag{ignore: true}
		}
	}
	return &t
}

// Converts any type to a slice of struct
// Supported input types:
// - []struct
// - []*struct
// - []interface{}
// - struct
// - *struct
// The output is guaranteed to be a slice of struct: []struct
func toSlice(v any) (any, reflect.Type) {
	if v == nil {
		return []struct{}{}, reflect.TypeOf(struct{}{})
	}

	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Slice:
		length := value.Len()
		if length == 0 {
			return []struct{}{}, reflect.TypeOf(struct{}{})
		}
		slice := make([]any, length)
		var objType reflect.Type
		for i := 0; i < length; i++ {
			switch value.Index(i).Kind() {
			case reflect.Ptr:
				elemType := value.Index(i).Elem().Type()
				if elemType.Kind() != reflect.Struct {
					return []struct{}{}, reflect.TypeOf(struct{}{})
				}
				if objType == nil {
					objType = elemType
				} else if objType != elemType {
					return []struct{}{}, reflect.TypeOf(struct{}{})
				}
				slice[i] = value.Index(i).Elem().Interface()
			case reflect.Struct:
				elemType := value.Index(i).Type()
				if objType == nil {
					objType = elemType
				} else if objType != elemType {
					return []struct{}{}, reflect.TypeOf(struct{}{})
				}
				slice[i] = value.Index(i).Interface()
			}
		}
		return slice, objType
	case reflect.Struct:
		return []any{v}, value.Type()
	case reflect.Ptr:
		if value.IsNil() {
			return []struct{}{}, reflect.TypeOf(struct{}{})
		}
		referenced := value.Elem()
		if referenced.Type().Kind() != reflect.Struct {
			return []struct{}{}, reflect.TypeOf(struct{}{})
		}
		return []any{referenced.Interface()}, referenced.Type()
	default:
		return []struct{}{}, reflect.TypeOf(struct{}{})
	}
}

func (t *Table) Render(v any) string {
	table := t.toTable(v, "", nil)
	return t.render(table, 0)
}

func (t *Table) render(table *table, depth int) string {
	var sb strings.Builder
	if table.name != "" {
		for i := 0; i < 2*depth-1; i++ {
			sb.WriteString(t.tab)
		}
		name := table.name
		if table.nameColor != nil {
			name = table.nameColor.Sprint(name)
		}
		sb.WriteString(name)
		sb.WriteString("\n")
	}
	if len(table.header) > 0 {
		for i := 0; i < 2*depth; i++ {
			sb.WriteString(t.tab)
		}
		for _, h := range table.header {
			if h == nil {
				continue
			}
			format := fmt.Sprintf("%%-%ds", h.width+t.padding)
			value := fmt.Sprintf(format, h.value)
			if h.color != nil {
				value = h.color.Sprint(value)
			}
			sb.WriteString(value)
		}
		sb.WriteString("\n")
	}
	for _, row := range table.data {
		for i := 0; i < 2*depth; i++ {
			sb.WriteString(t.tab)
		}
		for i, cell := range row.cells {
			if cell == nil {
				continue
			}
			format := fmt.Sprintf("%%-%ds", table.header[i].width+t.padding)
			value := fmt.Sprintf(format, cell.value)
			if cell.color != nil {
				value = cell.color.Sprint(value)
			}
			sb.WriteString(value)
		}
		sb.WriteString("\n")
		for _, subTable := range row.subTables {
			if len(subTable.data) == 0 {
				continue
			}
			content := t.render(&subTable, depth+1)
			sb.WriteString(content)
		}
	}
	return sb.String()
}

func (t *Table) toTable(v any, name string, nameColor *color.Color) *table {
	slice, objType := toSlice(v)
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Len() == 0 {
		return &table{name: name, nameColor: nameColor}
	}
	var tags []*tag
	var firstSet bool
	var header headerRow
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		if !field.IsExported() {
			tags = append(tags, &tag{ignore: true})
			header = append(header, nil)
			continue
		}
		tag := parseTag(field.Tag.Get("table"))
		if tag.ignore {
			tags = append(tags, tag)
			header = append(header, nil)
			continue
		}
		if tag.verbose && !t.verbose {
			tags = append(tags, tag)
			header = append(header, nil)
			continue
		}
		if tag.header == "" {
			tag.header = field.Name
		}
		if tag.headerColor == nil {
			tag.headerColor = t.headerRowColor
		}
		if tag.color == nil && !firstSet {
			tag.color = t.firstColumnColor
			firstSet = true
		}
		if tag.expand {
			header = append(header, nil)
		} else {
			header = append(header, &cell{
				value: tag.header,
				color: tag.headerColor,
				width: len(tag.header),
			})
		}
		tags = append(tags, tag)
	}
	var data []dataRow
	for i := 0; i < sliceValue.Len(); i++ {
		objValue := sliceValue.Index(i)
		var cells []*cell
		var subTables []table
		for j := 0; j < objType.NumField(); j++ {
			if tags[j].ignore || tags[j].verbose && !t.verbose {
				cells = append(cells, nil)
				continue
			}
			tag := tags[j]
			field := objValue.Elem().Field(j)
			if tag.expand {
				subTable := t.toTable(field.Interface(), tag.header, tag.headerColor)
				subTables = append(subTables, *subTable)
				cells = append(cells, nil)
				continue
			}
			cell := cell{
				color: tag.color,
			}
			if tm, ok := field.Interface().(time.Time); ok && !strings.HasPrefix(tag.format, "%") {
				cell.value = tm.Format(tag.format)
			} else if field.Kind() == reflect.Ptr && field.IsNil() {
				cell.value = "<nil>"
			} else if field.Kind() == reflect.Ptr {
				cell.value = fmt.Sprintf(tag.format, field.Elem().Interface())
			} else {
				cell.value = fmt.Sprintf(tag.format, field.Interface())
			}
			if header[j].width < len(cell.value) {
				header[j].width = len(cell.value)
			}
			cells = append(cells, &cell)
		}
		data = append(data, dataRow{
			cells:     cells,
			subTables: subTables,
		})
	}
	return &table{
		name:      name,
		nameColor: nameColor,
		header:    header,
		data:      data,
	}
}
