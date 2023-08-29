package table

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/require"
)

type Parent struct {
	// Use header to specify the column name
	ID int `table:"header:id_header"`
	// Specify the color override for the header
	RedHeader string `table:"headerColor:31,4"`
	// Specify the color override for the row
	RedRow string `table:"color:31,4"`
	// Only print if verbose is true
	Verbose string `table:"verbose"`
	// Specify how to format the value of this field
	Format int `table:"format:%d"`
	// Ignore this field
	Ignore string `table:"-"`
	// All children will be expanded if the value is not nil
	FirstChild    *Child  `table:"expand"`
	OtherChildren []Child `table:"expand"`
	NoExpand      Child
}

type Child struct {
	Key   string
	Value string
	Child *Child `table:"expand"`
}

func TestRender(t *testing.T) {
	color.NoColor = false
	p := Parent{
		ID:        1,
		RedHeader: "red header",
		RedRow:    "red row",
		Verbose:   "verbose",
		Format:    1,
		Ignore:    "ignore",
		FirstChild: &Child{
			Key:   "a",
			Value: "av",
			Child: &Child{
				Key:   "grand",
				Value: "child",
			},
		},
		OtherChildren: []Child{{
			Key:   "b",
			Value: "bv",
		}},
		NoExpand: Child{
			Key:   "c",
			Value: "cv",
		},
	}

	table, err := New().toTable(p, "", nil, true)
	require.NoError(t, err)
	require.NotNil(t, table)

	str, err := New().Render([]Parent{p, p}, true)
	require.NoError(t, err)
	fmt.Println(str)
}

func TestParseTag(t *testing.T) {
	tag, err := parseTag("header:header_name;headerColor:31,4;color:32;verbose;format:%d")
	require.NoError(t, err)
	require.False(t, tag.ignore)
	require.Equal(t, "header_name", tag.header)
	require.Equal(t, "%d", tag.format)
	require.True(t, tag.verbose)
	tag.color.EnableColor()
	require.Equal(t, "\u001B[32mtest\u001B[0m", tag.color.Sprint("test"))
	tag.headerColor.EnableColor()
	require.Equal(t, "\u001B[31;4mtest\u001B[0m", tag.headerColor.Sprint("test"))
}

func TestToSlice(t *testing.T) {
	tests := []struct {
		v   any
		len int
	}{
		{v: nil, len: 0},
		{
			v: &Child{
				Key:   "a",
				Value: "B",
			}, len: 1,
		},
		{
			v: Child{
				Key:   "a",
				Value: "B",
			}, len: 1,
		},
		{
			v: []Child{{
				Key:   "a",
				Value: "B",
			}}, len: 1,
		},
		{
			v: []any{Child{
				Key:   "a",
				Value: "B",
			}}, len: 1,
		},
		{
			v: []*Child{&Child{
				Key:   "a",
				Value: "B",
			}}, len: 1,
		},
		{
			v: []any{&Child{
				Key:   "a",
				Value: "B",
			}}, len: 1,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result, _, err := toSlice(test.v)
			require.NoError(t, err)
			resultLen := reflect.ValueOf(result).Len()
			require.Equal(t, test.len, resultLen)
		})
	}
}
