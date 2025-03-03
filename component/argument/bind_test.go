package argument

import (
	"errors"
	"testing"

	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/acyumi/doc-exporter/component/constant"
)

func TestArgs_Validate(t *testing.T) {
	tests := []struct {
		name      string
		AppID     string
		AppSecret string
		DocURL    string
		SaveDir   string
		expected  string
	}{
		{"AppID 为空", "", "valid_secret", "valid_url", "valid_dir", "AppID: app-id是必需参数."},
		{"AppSecret 为空", "valid_id", "", "valid_url", "valid_dir", "AppSecret: app-secret是必需参数."},
		{"DocURL 为空", "valid_id", "valid_secret", "", "valid_dir", "DocURL: url是必需参数."},
		{"SaveDir 为空", "valid_id", "valid_secret", "valid_url", "", "SaveDir: dir是必需参数."},
		{"所有参数都有效", "valid_id", "valid_secret", "valid_url", "valid_dir", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args Args
			args.AppID = tt.AppID
			args.AppSecret = tt.AppSecret
			args.DocURL = tt.DocURL
			args.SaveDir = tt.SaveDir
			err := args.Validate()
			if tt.expected == "" {
				assert.NoError(t, err, tt.name)
			} else {
				assert.IsType(t, oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				require.True(t, yes, tt.name)
				assert.Equal(t, "InvalidArgument", actualError.Code(), tt.name)
				assert.Equal(t, tt.expected, actualError.Error(), tt.name)
			}
		})
	}
}

func TestArgs_SetFileExtensions(t *testing.T) {
	tests := []struct {
		name     string
		fes      map[string]string
		expected map[constant.DocType]constant.FileExt
	}{
		{
			name:     "空参数",
			expected: map[constant.DocType]constant.FileExt{}},
		{
			name: "一个参数",
			fes:  map[string]string{"doc": "docx"},
			expected: map[constant.DocType]constant.FileExt{
				constant.DocTypeDoc: constant.FileExtDocx,
			},
		},
		{
			name: "多个参数",
			fes:  map[string]string{"doc": "docx", "sheet": "xlsx"},
			expected: map[constant.DocType]constant.FileExt{
				constant.DocTypeDoc:   constant.FileExtDocx,
				constant.DocTypeSheet: constant.FileExtXlsx,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args Args
			args.SetFileExtensions(tt.fes)
			assert.Equal(t, tt.expected, args.FileExtensions, tt.name)
		})
	}
}
