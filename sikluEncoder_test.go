package gosiklu

import "testing"

func Test_passwordEncode(t *testing.T) {
	type args struct {
		password string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"test1", args{"password"}, "cGFzc3dvcmQ91"},
		{"test2", args{"passWord2647"}, "cGFzc1dvcmQyNjQ30"},
		{"test3", args{""}, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := passwordEncode(tt.args.password); got != tt.want {
				t.Errorf("passwordEncode() = %v, want %v", got, tt.want)
			}
		})
	}
}
