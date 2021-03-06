package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToPascalCase(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "Converts underscore casing to pascal casing",
			input:  "some_table_name",
			output: "SomeTableName",
		},
		{
			name:   "Converts mixed casing",
			input:  "some_table_nameSomeName",
			output: "SomeTableNameSomeName",
		},
		{
			name:   "Does not change the value if it's pascal casing",
			input:  "SomeName",
			output: "SomeName",
		},
		{
			name:   "Trims single bad character at the end",
			input:  "Some{",
			output: "Some",
		},
		{
			name:   "Trims multiple bad characters at the end",
			input:  "Some{{{",
			output: "Some",
		},

		{
			name: "Handles empty string",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.output, ToPascalCase(tc.input))
		})
	}
}

func TestToUnderscoreCase(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		output string
	}{
		{

			name:   "Converts pascal casing to underscore casing",
			input:  "SomeTableName",
			output: "some_table_name",
		},
		{
			name:   "Converts mixed casing",
			input:  "some_table_nameSomeName",
			output: "some_table_name_some_name",
		},
		{
			name:   "Does not double underscores",
			input:  "some Table Name",
			output: "some_table_name",
		},
		{
			name:   "Respects several capital letters in a row",
			input:  "some TABLE NAME",
			output: "some_table_name",
		},
		{
			name:   "Does not change the value if it's underscore casing",
			input:  "some_name",
			output: "some_name",
		},
		{
			name:   "Trims single bad character at the end",
			input:  "Some{",
			output: "some",
		},
		{
			name:   "Trims multiple bad characters at the end",
			input:  "Some{{{",
			output: "some",
		},
		{
			name:   "Makes a separate upper case word a part of underscored identifier",
			input:  "created_at DESC",
			output: "created_at_desc",
		},
		{
			name:   "Makes a separate upper case word a part of underscored identifier",
			input:  "word$^%^%another _some   *&^%$A%$#%WORD",
			output: "word_another_some_a_word",
		},
		{
			name: "Handles empty string",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.output, ToUnderscoreCase(tc.input))
		})
	}
}
