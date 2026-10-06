package binary

import (
	"testing"

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
			name: "unrecognized format",
			err:  xerrors.New("bad magic number 'mZ' in record at offset 0x0: unrecognized file format"),
			want: ErrUnrecognizedExe,
		},
		{
			name: "not a Go executable",
			err:  xerrors.New("not a Go executable"),
			want: ErrNonGoBinary,
		},
		{
			name: "any other error is passed through",
			err:  xerrors.New("unexpected EOF"),
			want: xerrors.New("unexpected EOF"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.Error(), convertError(tt.err).Error())
		})
	}
}
