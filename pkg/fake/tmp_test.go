// Copyright Red Hat
package fake

import "testing"

func Test_helloworld(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "succeed",
			want: "hello world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helloworld(); got != tt.want {
				t.Errorf("helloworld() = %v, want %v", got, tt.want)
			}
		})
	}
}
