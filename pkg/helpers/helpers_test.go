package helpers

import (
	"testing"
)

func TestMergeMapStringString(t *testing.T) {
	m := make(map[string]string)
	limitadorName := "limitador"
	type args struct {
		existing *map[string]string
		desired  map[string]string
	}
	tests := []struct {
		name       string
		args       args
		wantUpdate bool
	}{
		{
			name: "nil pointer",
			args: args{
				existing: nil,
				desired: map[string]string{
					LabelKeyApp:               "limitador",
					LabelKeyLimitadorResource: limitadorName,
				},
			},
			wantUpdate: false,
		},
		{
			name: "empty map",
			args: args{
				existing: &m,
				desired: map[string]string{
					LabelKeyApp:               "limitador",
					LabelKeyLimitadorResource: limitadorName,
				},
			},
			wantUpdate: true,
		},
		{
			name: "update happened",
			args: args{
				existing: &map[string]string{
					"user-added-key": "value",
				},
				desired: map[string]string{
					LabelKeyApp:               "limitador",
					LabelKeyLimitadorResource: limitadorName,
				},
			},
			wantUpdate: true,
		},
		{
			name: "no update happened",
			args: args{
				existing: &map[string]string{
					LabelKeyApp:               "limitador",
					LabelKeyLimitadorResource: limitadorName,
				},
				desired: map[string]string{
					LabelKeyApp:               "limitador",
					LabelKeyLimitadorResource: limitadorName,
				},
			},
			wantUpdate: false,
		}, {
			name: "preserve hardcoded values",
			args: args{
				existing: &map[string]string{
					LabelKeyApp: "blah",
				},
				desired: map[string]string{
					LabelKeyApp:               "limitador",
					LabelKeyLimitadorResource: limitadorName,
				},
			},
			wantUpdate: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeMapStringString(tt.args.existing, tt.args.desired)
			if got != tt.wantUpdate {
				t.Errorf("MergeMapStringString() got = %v, wantUpdate %v", got, tt.wantUpdate)
			}
		})
	}
}
