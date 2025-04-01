// Copyright 2025 acyumi <417064257@qq.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argument

import (
	"errors"
	"testing"

	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/acyumi/xdoc/component/constant"
)

func TestArgs_Validate(t *testing.T) {
	tests := []struct {
		name      string
		AppID     string
		AppSecret string
		DocURLs   []string
		SaveDir   string
		expected  string
	}{
		{"AppID 为空", "", "valid_secret", []string{"valid_url"}, "valid_dir", "AppID: app-id是必需参数."},
		{"AppSecret 为空", "valid_id", "", []string{"valid_url"}, "valid_dir", "AppSecret: app-secret是必需参数."},
		{"DocURLs 为空", "valid_id", "valid_secret", []string{}, "valid_dir", "DocURLs: urls是必需参数."},
		{"SaveDir 为空", "valid_id", "valid_secret", []string{"valid_url"}, "", "SaveDir: dir是必需参数."},
		{"所有参数都有效", "valid_id", "valid_secret", []string{"valid_url"}, "valid_dir", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args Args
			args.AppID = tt.AppID
			args.AppSecret = tt.AppSecret
			args.DocURLs = tt.DocURLs
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

func TestArgs_DesensitizeSlice(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		verbose  bool
		expected []string
	}{
		{
			name: "url1",
			args: []string{
				"https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
				"https://xyzccc.feishu.cn/drive/folder/asdlfkjasfopweqprobpzxiqpo8",
			},
			verbose: true,
			expected: []string{
				"https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
				"https://xyzccc.feishu.cn/drive/folder/asdlfkjasfopweqprobpzxiqpo8",
			},
		},
		{
			name: "url1.脱敏",
			args: []string{
				"https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
				"https://xyzccc.feishu.cn/drive/folder/asdlfkjasfopweqprobpzxiqpo8",
			},
			verbose: false,
			expected: []string{
				"https://sam***.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
				"https://xyz***.feishu.cn/drive/folder/asdlfkjasfopweqprobpzxiqpo8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args Args
			args.Verbose = tt.verbose
			actual := args.DesensitizeSlice(tt.args...)
			assert.Equal(t, tt.expected, actual, tt.name)
		})
	}
}

func TestArgs_Desensitize(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		verbose  bool
		expected string
	}{
		{
			name:     "空参数1",
			str:      "",
			verbose:  false,
			expected: "",
		},
		{
			name:     "空参数2",
			str:      "",
			verbose:  true,
			expected: "",
		},
		{
			name:     "小于四个字符",
			str:      "xyz",
			verbose:  true,
			expected: "xyz",
		},
		{
			name:     "小于四个字符.脱敏",
			str:      "xyz",
			verbose:  false,
			expected: "xyz",
		},
		{
			name:     "大于四个字符",
			str:      "xyz...",
			verbose:  true,
			expected: "xyz...",
		},
		{
			name:     "大于四个字符.脱敏",
			str:      "xyz...",
			verbose:  false,
			expected: "xyz.**",
		},
		{
			name:     "url",
			str:      "https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe",
			verbose:  true,
			expected: "https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe",
		},
		{
			name:     "url.脱敏",
			str:      "https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe",
			verbose:  false,
			expected: "https://sam***.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe",
		},
		{
			name:     "url1",
			str:      "https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			verbose:  true,
			expected: "https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
		},
		{
			name:     "url1.脱敏",
			str:      "https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			verbose:  false,
			expected: "https://sam***.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args Args
			args.Verbose = tt.verbose
			actual := args.Desensitize(tt.str)
			assert.Equal(t, tt.expected, actual, tt.name)
		})
	}
}
