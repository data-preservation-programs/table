package table

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

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

	table := New().toTable(p, "", nil)
	require.NotNil(t, table)

	str := New(WithVerbose(), WithTab("  "), WithPaddingSize(2), WithFirstColumnColor(nil), WithHeaderRowColor(nil)).Render([]Parent{p, p})
	fmt.Println(str)
}

func TestParseTag(t *testing.T) {
	tag := parseTag("header:header_name;headerColor:31,4;color:32;verbose;format:%d")
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
			v: []*Child{{
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
			result, _ := toSlice(test.v)
			resultLen := reflect.ValueOf(result).Len()
			require.Equal(t, test.len, resultLen)
		})
	}
}

type Person struct {
	ID           int
	Name         string
	AverageScore int     `table:"header:Average Score"`
	Grade        string  `table:"color:96;headerColor:96,4"`
	Scores       []Score `table:"headerColor:34,4;expand"`
}

type Score struct {
	Subject  string
	Score    float32
	GradedAt time.Time `table:"header:Graded At;format:2006-01-02"`
}

func TestDemo(*testing.T) {
	people := []Person{
		{
			ID:           1,
			Name:         "John",
			AverageScore: 85,
			Grade:        "A",
			Scores: []Score{
				{
					Subject:  "Math",
					Score:    90,
					GradedAt: time.Now(),
				},
				{
					Subject:  "Science",
					Score:    80,
					GradedAt: time.Now(),
				},
			},
		},
		{
			ID:           2,
			Name:         "Joe",
			AverageScore: 75,
			Grade:        "B",
			Scores: []Score{
				{
					Subject:  "Math",
					Score:    80,
					GradedAt: time.Now(),
				},
				{
					Subject:  "Science",
					Score:    70,
					GradedAt: time.Now(),
				},
			},
		},
	}
	str := New().Render(people)
	fmt.Println(str)
}
