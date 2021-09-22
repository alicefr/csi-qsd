package driver

import "testing"

func Test_parseSizeToByteString(t *testing.T) {
	tests := []struct {
		size    string
		want    string
		wantErr bool
	}{
		{"300M", "300000000", false},
		{"100MB", "100000000", false},
		{"100GB", "100000000000", false},
		{"300 M", "300000000", false},
		{"300 MM", "", true},
		{"300 Mi", "", true},
	}
	for _, tt := range tests {
		t.Run("parsing", func(t *testing.T) {
			got, err := parseSizeToByteString(tt.size)
			if (err != nil) && !tt.wantErr {
				t.Errorf("parseSizeToByteString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseSizeToByteString() = %v, want %v err=%v", got, tt.want, err)
			}
		})
	}
}
