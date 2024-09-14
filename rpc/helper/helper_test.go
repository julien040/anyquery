package helper

import (
	"testing"
	"time"
)

func TestSerialization(t *testing.T) {
	tt := []struct {
		name string
		in   interface{}
		out  interface{}
	}{
		{
			name: "string",
			in:   "hello",
			out:  "hello",
		},
		{
			name: "nil pointer to string",
			in:   (*string)(nil),
			out:  nil,
		},
		{
			name: "pointer to string",
			in:   new(string),
			out:  "",
		},
		{
			name: "time",
			in:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			out:  "2020-01-01T00:00:00Z",
		}, {
			name: "slice of structs",
			in: []interface{}{struct {
				Name string
				Age  int
			}{Name: "John", Age: 25}},
			out: `[{"Name":"John","Age":25}]`,
		},
		{
			name: "struct",
			in: struct {
				Name string
				Age  int
			}{Name: "John", Age: 25},
			out: `{"Name":"John","Age":25}`,
		},
		{
			name: "slice",
			in:   []interface{}{},
			out:  nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := Serialize(tc.in)
			if out != tc.out {
				t.Errorf("Expected %v, got %v", tc.out, out)
			}
		})
	}

}
