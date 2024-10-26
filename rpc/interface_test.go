package rpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUserConfigHelpers(t *testing.T) {
	constraints := QueryConstraint{
		Columns: []ColumnConstraint{
			{
				ColumnID: 0,
				Operator: OperatorEqual,
				Value:    "value",
			},
			{
				ColumnID: 2,
				Operator: OperatorEqual,
				Value:    "2024-01-01T00:00:00Z",
			},
			{
				ColumnID: 3,
				Operator: OperatorEqual,
				Value:    1704067200,
			},
			{
				ColumnID: 4,
				Operator: OperatorEqual,
				Value:    "true",
			},
			{
				ColumnID: 5,
				Operator: OperatorEqual,
				Value:    1,
			},
		},
	}

	// Check if we can extract the value from the constraints
	value := constraints.GetColumnConstraint(0).GetStringValue()
	require.Equal(t, "value", value)

	wrongValInt := constraints.GetColumnConstraint(0).GetIntValue()
	require.Equal(t, int64(0), wrongValInt)

	wrongValFloat := constraints.GetColumnConstraint(0).GetFloatValue()
	require.Equal(t, 0.0, wrongValFloat)

	// Ensure the constraints is the equal operator
	isEqual := constraints.GetColumnConstraint(0).IsEqual()
	require.True(t, isEqual)

	// Check if a constraint that doesn't exist returns the zero value
	zeroValueString := constraints.GetColumnConstraint(1).GetStringValue()
	require.Equal(t, "", zeroValueString)

	zeroValueInt := constraints.GetColumnConstraint(1).GetIntValue()
	require.Equal(t, int64(0), zeroValueInt)

	zeroValueFloat := constraints.GetColumnConstraint(1).GetFloatValue()
	require.Equal(t, 0.0, zeroValueFloat)

	// Check for time.Time value
	timeValue := constraints.GetColumnConstraint(2).GetTimeValue()
	require.Equal(t, "2024-01-01T00:00:00Z", timeValue.Format(time.RFC3339))

	// Check for int64 value
	intValue := constraints.GetColumnConstraint(3).GetIntValue()
	require.Equal(t, int64(1704067200), intValue)

	// Check for unix timestamp value
	unixValue := constraints.GetColumnConstraint(3).GetTimeValue()
	require.True(t, unixValue.Equal(time.Unix(1704067200, 0)))

	// Check for bool value
	boolValue := constraints.GetColumnConstraint(4).GetBoolValue()
	require.Equal(t, true, boolValue)

	boolValue = constraints.GetColumnConstraint(5).GetBoolValue()
	require.Equal(t, true, boolValue)

}

func TestPluginConfigHelpers(t *testing.T) {
	config := PluginConfig{
		"apiKey": "1234",
		"count":  int64(42),
		"float":  42.42,
		"array":  []string{"a", "b", "c"},
	}

	// Check if we can extract the value from the config
	apiKey := config.GetString("apiKey")
	require.Equal(t, "1234", apiKey)

	count := config.GetInt("count")
	require.Equal(t, int64(42), count)

	float := config.GetFloat("float")
	require.Equal(t, 42.42, float)

	array := config.GetStringArray("array")
	require.Equal(t, []string{"a", "b", "c"}, array)

	// Ensure a missing key returns the zero value
	zeroValueString := config.GetString("missing")
	require.Equal(t, "", zeroValueString)

	zeroValueInt := config.GetInt("missing")
	require.Equal(t, int64(0), zeroValueInt)

	zeroValueFloat := config.GetFloat("missing")
	require.Equal(t, 0.0, zeroValueFloat)

	zeroValueArray := config.GetStringArray("missing")
	require.Equal(t, []string([]string(nil)), zeroValueArray)

}

func TestQueryConstraintHash(t *testing.T) {
	constraints1 := QueryConstraint{
		Columns: []ColumnConstraint{
			{
				ColumnID: 0,
				Operator: OperatorEqual,
				Value:    "value",
			},
			{
				ColumnID: 2,
				Operator: OperatorEqual,
				Value:    "2024-01-01T00:00:00Z",
			},
		},
		Limit:  10,
		Offset: 5,
	}

	constraints2 := QueryConstraint{
		Columns: []ColumnConstraint{
			{
				ColumnID: 2,
				Operator: OperatorEqual,
				Value:    "2024-01-01T00:00:00Z",
			},
			{
				ColumnID: 0,
				Operator: OperatorEqual,
				Value:    "value",
			},
		},
		Limit:  10,
		Offset: 5,
	}

	constraints3 := QueryConstraint{
		Columns: []ColumnConstraint{
			{
				ColumnID: 0,
				Operator: OperatorGreater,
				Value:    "value",
			},
			{
				ColumnID: 2,
				Operator: OperatorEqual,
				Value:    "2024-01-01T00:00:00Z",
			},
		},
		Limit:  10,
		Offset: 500,
	}

	hash1 := constraints1.Hash()
	hash2 := constraints2.Hash()
	hash3 := constraints3.Hash()

	// Ensure the hash is the same for the same constraints
	require.Equal(t, hash1, hash2)

	// Ensure the hash is different for different constraints
	require.NotEqual(t, hash1, hash3)
	require.NotEqual(t, hash2, hash3)

}
