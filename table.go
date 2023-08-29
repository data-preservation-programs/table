package table

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/fatih/color"
)

type Table struct {
	tab              string
	headerRowColor   *color.Color
	firstColumnColor *color.Color
	padding          int
}

type Option func(*Table)

func WithTab(p string) Option {
	return func(t *Table) {
		t.tab = p
	}
}

func WithHeaderRowColor(c *color.Color) Option {
	return func(t *Table) {
		t.headerRowColor = c
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

var ErrInvalidTag = errors.New("invalid tag")

func parseTag(str string) (*tag, error) {
	t := tag{}
	if str == "" {
		return &t, nil
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
			return nil, errors.Wrapf(ErrInvalidTag, "'%s' should contain :", s)
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
					return nil, errors.Wrapf(ErrInvalidTag, "'%s' is not int", c)
				}
				attributes = append(attributes, color.Attribute(attr))
			}
			t.headerColor = color.New(attributes...)
		case "color":
			var attributes []color.Attribute
			for _, c := range strings.Split(value, ",") {
				attr, err := strconv.Atoi(c)
				if err != nil {
					return nil, errors.Wrapf(ErrInvalidTag, "'%s' is not int", c)
				}
				attributes = append(attributes, color.Attribute(attr))
			}
			t.color = color.New(attributes...)
		case "format":
			t.format = value
		default:
			return nil, errors.Wrapf(ErrInvalidTag, "unrecognized tag %s", s)
		}
	}
	return &t, nil
}

func shouldUseSubTable(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			return false
		}
		firstObj := v.Index(0)
		return firstObj.Kind() == reflect.Struct || firstObj.Kind() == reflect.Ptr && firstObj.Elem().Kind() == reflect.Struct
	case reflect.Struct:
		return true
	case reflect.Ptr:
		return v.Elem().Kind() == reflect.Struct
	default:
		return false
	}
}

// Converts any type to a slice of struct
// Supported input types:
// - []struct
// - []*struct
// - []interface{}
// - struct
// - *struct
// The output is guaranteed to be a slice of struct: []struct
func toSlice(v any) (any, reflect.Type, error) {
	if v == nil {
		return []struct{}{}, reflect.TypeOf(struct{}{}), nil
	}

	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Slice:
		length := value.Len()
		if length == 0 {
			return []struct{}{}, reflect.TypeOf(struct{}{}), nil
		}
		slice := make([]any, length)
		var objType reflect.Type
		for i := 0; i < length; i++ {
			switch value.Index(i).Kind() {
			case reflect.Ptr:
				elemType := value.Index(i).Elem().Type()
				if elemType.Kind() != reflect.Struct {
					return nil, nil, errors.Errorf("unsupported type %s", objType.Kind())
				}
				if objType == nil {
					objType = elemType
				} else if objType != elemType {
					return nil, nil, errors.Errorf("mismatched types %s and %s", objType.Kind(), elemType.Kind())
				}
				slice[i] = value.Index(i).Elem().Interface()
			case reflect.Struct:
				elemType := value.Index(i).Type()
				if objType == nil {
					objType = elemType
				} else if objType != elemType {
					return nil, nil, errors.Errorf("mismatched types %s and %s", objType.Kind(), elemType.Kind())
				}
				slice[i] = value.Index(i).Interface()
			}
		}
		return slice, objType, nil
	case reflect.Struct:
		return []any{v}, value.Type(), nil
	case reflect.Ptr:
		if value.IsNil() {
			return []struct{}{}, reflect.TypeOf(struct{}{}), nil
		}
		referenced := value.Elem()
		if referenced.Type().Kind() != reflect.Struct {
			return nil, nil, errors.Errorf("unsupported type %s", value.Kind())
		}
		return []any{referenced.Interface()}, referenced.Type(), nil
	default:
		return nil, nil, errors.Errorf("unsupported type %s", value.Kind())
	}
}

func (t *Table) Render(v any, verbose bool) (string, error) {
	table, err := t.toTable(v, "", nil, verbose)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return t.render(table, 0), nil
}

func (t *Table) render(table *table, depth int) string {
	var sb strings.Builder
	if table.name != "" {
		for i := 0; i < depth; i++ {
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
		for i := 0; i < depth; i++ {
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
		for i := 0; i < depth; i++ {
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
			content := t.render(&subTable, depth+1)
			sb.WriteString(content)
		}
	}
	return sb.String()
}

func (t *Table) toTable(v any, name string, nameColor *color.Color, verbose bool) (*table, error) {
	slice, objType, err := toSlice(v)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Len() == 0 {
		return &table{name: name, nameColor: nameColor}, nil
	}
	var tags []*tag
	var firstSet bool
	var header headerRow
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		tag, err := parseTag(field.Tag.Get("table"))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if tag.ignore {
			tags = append(tags, tag)
			header = append(header, nil)
			continue
		}
		if tag.verbose && !verbose {
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
			if tags[j].ignore || tags[j].verbose && !verbose {
				cells = append(cells, nil)
				continue
			}
			tag := tags[j]
			field := objValue.Elem().Field(j)
			if tag.expand {
				subTable, err := t.toTable(field.Interface(), tag.header, tag.headerColor, verbose)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				subTables = append(subTables, *subTable)
				cells = append(cells, nil)
				continue
			}
			cell := cell{
				color: tag.color,
			}
			cell.value = fmt.Sprint(field.Interface())
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
		name:   name,
		header: header,
		data:   data,
	}, nil
}