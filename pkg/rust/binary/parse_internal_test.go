package binary

import (
	"testing"

	rustaudit "github.com/microsoft/go-rustaudit"
	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func TestConvertError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "unknown file format",
			err:  rustaudit.ErrUnknownFileFormat,
			want: ErrUnrecognizedExe,
		},
		{
			name: "no Rust dependency info",
			err:  rustaudit.ErrNoRustDepInfo,
			want: ErrNonRustBinary,
		},
		{
			name: "any other error is passed through",
			err:  xerrors.New("some other error"),
			want: xerrors.New("some other error"),
		},
		{
			name: "nil error",
			err:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertError(tt.err)
			if tt.want == nil {
				assert.NoError(t, got)
				return
			}
			assert.Equal(t, tt.want.Error(), got.Error())
		})
	}
}
