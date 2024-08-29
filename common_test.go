package gt

import "testing"

func TestToUpperCamelCase(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "测试",
			args: args{
				s: "4to7",
			},
			want: "4To7",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToUpperCamelCase(tt.args.s); got != tt.want {
				t.Errorf("ToUpperCamelCase() = %v, want %v", got, tt.want)
			}
		})
	}
}
