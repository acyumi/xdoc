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

package feishu

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xlab/treeprint"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/constant"
)

func TestDocumentInfo_GetFileName(t *testing.T) {
	tests := []struct {
		name         string
		documentInfo DocumentInfo
		expected     string
	}{
		{
			name: "Test with normal file name",
			documentInfo: DocumentInfo{
				Name:          "test",
				FileExtension: "txt",
			},
			expected: "test.txt",
		},
		{
			name: "Test with empty file extension",
			documentInfo: DocumentInfo{
				Name:          "test",
				FileExtension: "",
			},
			expected: "test.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.documentInfo.GetFileName()
			require.Equal(t, tt.expected, actual, tt.name)
		})
	}
}

func TestCleanName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Test with normal name",
			input:    "test",
			expected: "test",
		},
		{
			name:     "Test with special characters",
			input:    "test\\/:*?\"<>|",
			expected: "test_________",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := cleanName(tt.input)
			require.Equal(t, tt.expected, actual, tt.name)
		})
	}
}

func TestSetFileExtension(t *testing.T) {
	tests := []struct {
		name         string
		documentNode DocumentNode
		args         argument.Args
		expectedType constant.DocType
		expectedExt  constant.FileExt
	}{
		{
			name: "Test with docx type",
			documentNode: DocumentNode{
				DocumentInfo: DocumentInfo{
					Type: constant.DocTypeDocx,
				},
			},
			args:         argument.Args{FileExtensions: map[constant.DocType]constant.FileExt{}},
			expectedType: constant.DocTypeDocx,
			expectedExt:  constant.FileExtDocx,
		},
		{
			name: "Test with sheet type",
			documentNode: DocumentNode{
				DocumentInfo: DocumentInfo{
					Type: constant.DocTypeSheet,
				},
			},
			args:         argument.Args{FileExtensions: map[constant.DocType]constant.FileExt{}},
			expectedType: constant.DocTypeSheet,
			expectedExt:  constant.FileExtXlsx,
		},
		{
			name: "Test with unknown type",
			documentNode: DocumentNode{
				DocumentInfo: DocumentInfo{
					Type: "unknown",
				},
			},
			args:         argument.Args{FileExtensions: map[constant.DocType]constant.FileExt{}},
			expectedType: "unknown",
			expectedExt:  "unknown",
		},
		{
			name: "Test with file type and xlsx extension",
			documentNode: DocumentNode{
				DocumentInfo: DocumentInfo{
					Type: constant.DocTypeFile,
					Name: "example.xlsx",
				},
			},
			args:         argument.Args{FileExtensions: map[constant.DocType]constant.FileExt{}},
			expectedType: constant.DocTypeSheet,
			expectedExt:  constant.FileExtXlsx,
		},
		{
			name: "Test with file type and docx extension",
			documentNode: DocumentNode{
				DocumentInfo: DocumentInfo{
					Type: constant.DocTypeFile,
					Name: "example.docx",
				},
			},
			args:         argument.Args{FileExtensions: map[constant.DocType]constant.FileExt{}},
			expectedType: constant.DocTypeDocx,
			expectedExt:  constant.FileExtDocx,
		},
		{
			name: "Test with file type and pdf extension",
			documentNode: DocumentNode{
				DocumentInfo: DocumentInfo{
					Type: constant.DocTypeFile,
					Name: "example.pdf",
				},
			},
			args:         argument.Args{FileExtensions: map[constant.DocType]constant.FileExt{}},
			expectedType: constant.DocTypePDF,
			expectedExt:  "pdf",
		},
		{
			name: "Test with file type and no extension",
			documentNode: DocumentNode{
				DocumentInfo: DocumentInfo{
					Type: constant.DocTypeFile,
					Name: "example",
				},
			},
			args:         argument.Args{FileExtensions: map[constant.DocType]constant.FileExt{}},
			expectedType: constant.DocTypeFile,
			expectedExt:  "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFileExtension(&tt.documentNode, &tt.args)
			require.Equal(t, tt.expectedType, tt.documentNode.Type, tt.name)
			require.Equal(t, tt.expectedExt, tt.documentNode.FileExtension, tt.name)
		})
	}
}

func Test_deduplication(t *testing.T) {
	tests := []struct {
		name     string
		dns      []*DocumentNode
		expected []*DocumentNode
	}{
		{
			name: "只有一个文档，不用去重",
			dns: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:          "test",
						Type:          constant.DocTypeDocx,
						FileExtension: constant.FileExtDocx,
					},
				},
			},
			expected: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:          "test",
						Type:          constant.DocTypeDocx,
						FileExtension: constant.FileExtDocx,
					},
				},
			},
		},
		{
			name: "两棵相同的树",
			dns: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:  "folder",
						Type:  "folder",
						Token: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								Token:         "file1",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:  "folder",
						Type:  "folder",
						Token: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								Token:         "file1",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
			},
			expected: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:  "folder",
						Type:  "folder",
						Token: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								Token:         "file1",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
			},
		},
		{
			name: "树3(树2(树1))", // 用()表示包含
			dns: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:          "file2",
						Type:          constant.DocTypeDocx,
						Token:         "file2",
						FileExtension: constant.FileExtDocx,
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:  "folder1",
						Type:  "folder",
						Token: "folder1",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								Token:         "file1",
								FileExtension: constant.FileExtDocx,
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "file2",
								Type:          constant.DocTypeDocx,
								Token:         "file2",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:  "root",
						Type:  "folder",
						Token: "root",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder1",
								Type:  "folder",
								Token: "folder1",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										Token:         "file1",
										FileExtension: constant.FileExtDocx,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										Token:         "file2",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder2",
								Type:  "folder",
								Token: "folder2",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										Token:         "file3",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
					},
				},
			},
			expected: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:  "root",
						Type:  "folder",
						Token: "root",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder1",
								Type:  "folder",
								Token: "folder1",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										Token:         "file1",
										FileExtension: constant.FileExtDocx,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										Token:         "file2",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder2",
								Type:  "folder",
								Token: "folder2",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										Token:         "file3",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "树2(树1(树3))", // 用()表示包含
			dns: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:  "folder1",
						Type:  "folder",
						Token: "folder1",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								Token:         "file1",
								FileExtension: constant.FileExtDocx,
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "file2",
								Type:          constant.DocTypeDocx,
								Token:         "file2",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:  "root",
						Type:  "folder",
						Token: "root",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder1",
								Type:  "folder",
								Token: "folder1",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										Token:         "file1",
										FileExtension: constant.FileExtDocx,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										Token:         "file2",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder2",
								Type:  "folder",
								Token: "folder2",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										Token:         "file3",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:          "file2",
						Type:          constant.DocTypeDocx,
						Token:         "file2",
						FileExtension: constant.FileExtDocx,
					},
				},
			},
			expected: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:  "root",
						Type:  "folder",
						Token: "root",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder1",
								Type:  "folder",
								Token: "folder1",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										Token:         "file1",
										FileExtension: constant.FileExtDocx,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										Token:         "file2",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder2",
								Type:  "folder",
								Token: "folder2",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										Token:         "file3",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "树2(树1(树3))树4(树5)", // 用()表示包含
			dns: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:  "folder1",
						Type:  "folder",
						Token: "folder1",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								Token:         "file1",
								FileExtension: constant.FileExtDocx,
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "file2",
								Type:          constant.DocTypeDocx,
								Token:         "file2",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:  "root",
						Type:  "folder",
						Token: "root",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder1",
								Type:  "folder",
								Token: "folder1",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										Token:         "file1",
										FileExtension: constant.FileExtDocx,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										Token:         "file2",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder2",
								Type:  "folder",
								Token: "folder2",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										Token:         "file3",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:          "file2",
						Type:          constant.DocTypeDocx,
						Token:         "file2",
						FileExtension: constant.FileExtDocx,
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:          "file4",
						Type:          constant.DocTypeDocx,
						Token:         "file4",
						FileExtension: constant.FileExtDocx,
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file5",
								Type:          constant.DocTypeDocx,
								Token:         "file5",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:          "file5",
						Type:          constant.DocTypeDocx,
						Token:         "file5",
						FileExtension: constant.FileExtDocx,
					},
				},
			},
			expected: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:  "root",
						Type:  "folder",
						Token: "root",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder1",
								Type:  "folder",
								Token: "folder1",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										Token:         "file1",
										FileExtension: constant.FileExtDocx,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										Token:         "file2",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:  "folder2",
								Type:  "folder",
								Token: "folder2",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										Token:         "file3",
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:          "file4",
						Type:          constant.DocTypeDocx,
						Token:         "file4",
						FileExtension: constant.FileExtDocx,
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file5",
								Type:          constant.DocTypeDocx,
								Token:         "file5",
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := deduplication(tt.dns)
			require.Len(t, tt.expected, len(actual), tt.name)
			require.EqualValues(t, tt.expected, actual, tt.name)
		})
	}
}

func TestDocumentNodeToInfoList(t *testing.T) {
	tests := []struct {
		name         string
		documentNode *DocumentNode
		expected     []*DocumentInfo
	}{
		{
			name: "Test with nested folders and files",
			documentNode: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name: "root",
					Type: "folder",
				},
				Children: []*DocumentNode{
					{
						DocumentInfo: DocumentInfo{
							Name: "folder1",
							Type: "folder",
						},
						Children: []*DocumentNode{
							{
								DocumentInfo: DocumentInfo{
									Name:          "file1",
									Type:          constant.DocTypeDocx,
									FileExtension: constant.FileExtDocx,
								},
							},
							{
								DocumentInfo: DocumentInfo{
									Name:          "file2",
									Type:          constant.DocTypeDocx,
									FileExtension: constant.FileExtDocx,
								},
							},
						},
					},
					{
						DocumentInfo: DocumentInfo{
							Name: "folder2",
							Type: "folder",
						},
						Children: []*DocumentNode{
							{
								DocumentInfo: DocumentInfo{
									Name:          "file3",
									Type:          constant.DocTypeDocx,
									FileExtension: constant.FileExtDocx,
								},
							},
						},
					},
				},
			},
			expected: []*DocumentInfo{
				{
					Name:     "root",
					Type:     "folder",
					FilePath: "",
				},
				{
					Name:     "folder1",
					Type:     "folder",
					FilePath: "",
				},
				{
					Name:          "file1",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      "",
				},
				{
					Name:          "file2",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      "",
				},
				{
					Name:     "folder2",
					Type:     "folder",
					FilePath: "",
				},
				{
					Name:          "file3",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := documentNodeToInfoList(tt.documentNode)
			require.Len(t, tt.expected, len(actual), tt.name)
			require.EqualValues(t, tt.expected, actual, tt.name)
		})
	}
}

func TestDocumentNodesToInfoList(t *testing.T) {
	tests := []struct {
		name          string
		documentNodes []*DocumentNode
		saveDir       string
		expected      []*DocumentInfo
	}{
		{
			name: "Test with single file",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:          "test",
						Type:          constant.DocTypeDocx,
						FileExtension: constant.FileExtDocx,
					},
				},
			},
			saveDir: "saveDir",
			expected: []*DocumentInfo{
				{
					Name:          "test",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      strings.Join([]string{"saveDir", "test.docx"}, string(os.PathSeparator)),
				},
			},
		},
		{
			name: "Test with folder and files",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name: "folder",
						Type: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								FileExtension: constant.FileExtDocx,
							},
						},
					},
				},
			},
			saveDir: "saveDir",
			expected: []*DocumentInfo{
				{
					Name:     "folder",
					Type:     "folder",
					FilePath: strings.Join([]string{"saveDir", "folder"}, string(os.PathSeparator)),
				},
				{
					Name:          "file1",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      strings.Join([]string{"saveDir", "folder", "file1.docx"}, string(os.PathSeparator)),
				},
			},
		},
		{
			name: "Test with nested folders and files",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name: "root",
						Type: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name: "folder1",
								Type: "folder",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtDocx,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name: "folder2",
								Type: "folder",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtDocx,
									},
								},
							},
						},
					},
				},
			},
			saveDir: "saveDir",
			expected: []*DocumentInfo{
				{
					Name:     "root",
					Type:     "folder",
					FilePath: strings.Join([]string{"saveDir", "root"}, string(os.PathSeparator)),
				},
				{
					Name:     "folder1",
					Type:     "folder",
					FilePath: strings.Join([]string{"saveDir", "root", "folder1"}, string(os.PathSeparator)),
				},
				{
					Name:          "file1",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      strings.Join([]string{"saveDir", "root", "folder1", "file1.docx"}, string(os.PathSeparator)),
				},
				{
					Name:          "file2",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      strings.Join([]string{"saveDir", "root", "folder1", "file2.docx"}, string(os.PathSeparator)),
				},
				{
					Name:     "folder2",
					Type:     "folder",
					FilePath: strings.Join([]string{"saveDir", "root", "folder2"}, string(os.PathSeparator)),
				},
				{
					Name:          "file3",
					Type:          constant.DocTypeDocx,
					FileExtension: constant.FileExtDocx,
					FilePath:      strings.Join([]string{"saveDir", "root", "folder2", "file3.docx"}, string(os.PathSeparator)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := documentNodesToInfoList(tt.documentNodes, tt.saveDir)
			require.Len(t, tt.expected, len(actual), tt.name)
			require.EqualValues(t, tt.expected, actual, tt.name)
		})
	}
}

func TestPrintTree(t *testing.T) {
	tests := []struct {
		name           string
		documentNodes  []*DocumentNode
		expectedOutput string
	}{
		{
			name: "Test with single file",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:          "test",
						Type:          constant.DocTypeDocx,
						FileExtension: constant.FileExtDocx,
						CanDownload:   true,
					},
				},
			},
			expectedOutput: `
/tmp
└─ test.docx
`,
		},
		{
			name: "Test with two file",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name:          "test1",
						Type:          constant.DocTypeDocx,
						FileExtension: constant.FileExtDocx,
						CanDownload:   true,
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name:          "test2",
						Type:          constant.DocTypeDocx,
						FileExtension: constant.FileExtDocx,
						CanDownload:   true,
					},
				},
			},
			expectedOutput: `
/tmp
├─ test1.docx
└─ test2.docx
`,
		},
		{
			name: "Test with folder and files",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name: "folder1",
						Type: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								FileExtension: constant.FileExtDocx,
								CanDownload:   true,
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "file2",
								Type:          constant.DocTypeMindNote,
								FileExtension: "mindnote",
								CanDownload:   false,
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "file3",
								Type:          constant.DocTypeSlides,
								FileExtension: "slides",
								CanDownload:   false,
							},
						},
					},
				},
				{
					DocumentInfo: DocumentInfo{
						Name: "folder2",
						Type: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file1",
								Type:          constant.DocTypeDocx,
								FileExtension: constant.FileExtDocx,
								CanDownload:   true,
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "file2",
								Type:          constant.DocTypeMindNote,
								FileExtension: "mindnote",
								CanDownload:   false,
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "file3",
								Type:          constant.DocTypeSlides,
								FileExtension: "slides",
								CanDownload:   false,
							},
						},
					},
				},
			},
			expectedOutput: `
/tmp
├─ folder1
│   ├─ file1.docx
│   ├─ file2.mindnote（不可下载）
│   └─ file3.slides（不可下载）
└─ folder2
    ├─ file1.docx
    ├─ file2.mindnote（不可下载）
    └─ file3.slides（不可下载）
`,
		},
		{
			name: "Test with nested folders and files",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name: "root",
						Type: "folder",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name: "folder1",
								Type: "folder",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtDocx,
										CanDownload:   true,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtDocx,
										CanDownload:   true,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name: "folder2",
								Type: "folder",
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file3",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtDocx,
										CanDownload:   true,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name: "folder3",
								Type: constant.DocTypeFolder,
							},
							Children: []*DocumentNode{},
						},
					},
				},
			},
			expectedOutput: `
/tmp
└─ root
    ├─ folder1
    │   ├─ file1.docx
    │   └─ file2.docx
    ├─ folder2
    │   └─ file3.docx
    └─ folder3
`,
		},
		{
			name: "Test with nested files(to be folders) and files",
			documentNodes: []*DocumentNode{
				{
					DocumentInfo: DocumentInfo{
						Name: "root",
						Type: constant.DocTypeFolder,
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:          "file0",
								Type:          constant.DocTypeDocx,
								FileExtension: constant.FileExtDocx,
								CanDownload:   true,
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file1",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtPDF,
										CanDownload:   true,
									},
								},
								{
									DocumentInfo: DocumentInfo{
										Name:          "file2",
										Type:          constant.DocTypeMindNote,
										FileExtension: "mindnote",
										CanDownload:   false,
									},
								},
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:          "folder3",
								Type:          constant.DocTypeDocx,
								FileExtension: constant.FileExtDocx,
								CanDownload:   true,
							},
							Children: []*DocumentNode{
								{
									DocumentInfo: DocumentInfo{
										Name:          "file4",
										Type:          constant.DocTypeDocx,
										FileExtension: constant.FileExtDocx,
										CanDownload:   true,
									},
								},
							},
						},
					},
				},
			},
			expectedOutput: `
/tmp
└─ root
    ├─ file0.docx
    ├─ file0
    │   ├─ file1.pdf
    │   └─ file2.mindnote（不可下载）
    ├─ folder3.docx
    └─ folder3
        └─ file4.docx
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual strings.Builder
			treeprint.EdgeTypeMid = "├─"
			treeprint.EdgeTypeEnd = "└─"
			treeprint.IndentSize = 3
			printTree(&actual, treeprint.NewWithRoot("/tmp"), tt.documentNodes, 0, 0)
			// printTree(os.Stdout, treeprint.NewWithRoot("/tmp"), &tt.documentNode, 0, 0)
			require.Equal(t, tt.expectedOutput, actual.String(), tt.name)
		})
	}
}

func TestGetName(t *testing.T) {
	tests := []struct {
		name                  string
		inputName             string
		inputType             constant.DocType
		duplicateNameIndexMap map[string]int
		expected              string
	}{
		{
			name:                  "Test with normal name",
			inputName:             "test",
			inputType:             "docx",
			duplicateNameIndexMap: map[string]int{},
			expected:              "test",
		},
		{
			name:                  "Test with duplicate name",
			inputName:             "test",
			inputType:             "docx",
			duplicateNameIndexMap: map[string]int{"test": 0},
			expected:              "test1",
		},
		{
			name:                  "Test with docx type empty name",
			inputName:             "",
			inputType:             "docx",
			duplicateNameIndexMap: map[string]int{},
			expected:              "未命名新版文档1",
		},
		{
			name:                  "Test with doc type empty name",
			inputName:             "",
			inputType:             "doc",
			duplicateNameIndexMap: map[string]int{},
			expected:              "未命名旧版文档1",
		},
		{
			name:                  "Test with sheet type empty name",
			inputName:             "",
			inputType:             "sheet",
			duplicateNameIndexMap: map[string]int{},
			expected:              "未命名电子表格1",
		},
		{
			name:                  "Test with bitable type empty name",
			inputName:             "",
			inputType:             "bitable",
			duplicateNameIndexMap: map[string]int{},
			expected:              "未命名多维表格1",
		},
		{
			name:                  "Test with mindnote type empty name",
			inputName:             "",
			inputType:             "mindnote",
			duplicateNameIndexMap: map[string]int{},
			expected:              "未命名思维笔记1",
		},
		{
			name:                  "Test with slides type empty name",
			inputName:             "",
			inputType:             "slides",
			duplicateNameIndexMap: map[string]int{},
			expected:              "未命名幻灯片1",
		},
		{
			name:                  "Test with unknown type empty name",
			inputName:             "",
			inputType:             "unknown",
			duplicateNameIndexMap: map[string]int{},
			expected:              "未命名飞书文档1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getName(tt.inputName, tt.inputType, tt.duplicateNameIndexMap)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestDoExportAndDownload(t *testing.T) {
	tests := []struct {
		name            string
		setupMock       func(*MockTask)
		wantErr         string
		expectValidated bool
		expectRan       bool
		expectClosed    bool
	}{
		{
			name: "验证失败应关闭任务",
			setupMock: func(m *MockTask) {
				m.EXPECT().Validate().Return(errors.New("invalid token")).Once()
			},
			wantErr:         `invalid token`,
			expectValidated: true,
			expectRan:       false,
			expectClosed:    false,
		},
		{
			name: "任务运行失败应关闭资源",
			setupMock: func(m *MockTask) {
				m.EXPECT().Validate().Return(nil).Once()
				m.EXPECT().Run().Return(errors.New("network error")).Once()
				m.EXPECT().Close().Return().Once()
			},
			wantErr:         `network error`,
			expectValidated: true,
			expectRan:       true,
			expectClosed:    true,
		},
		{
			name: "成功执行应正常关闭",
			setupMock: func(m *MockTask) {
				m.EXPECT().Validate().Return(nil).Once()
				m.EXPECT().Run().Return(nil).Once()
				m.EXPECT().Close().Return().Once()
			},
			wantErr:         ``,
			expectValidated: true,
			expectRan:       true,
			expectClosed:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 初始化Mock对象
			mt := NewMockTask(t)
			tt.setupMock(mt)

			// 执行测试
			err := doExportAndDownload(mt)

			// 断言错误
			if err != nil {
				require.EqualError(t, err, tt.wantErr, tt.name)
			}

			// 验证方法调用
			if tt.expectValidated {
				mt.AssertCalled(t, "Validate")
			} else {
				mt.AssertNotCalled(t, "Validate")
			}
			if tt.expectRan {
				mt.AssertCalled(t, "Run")
			} else {
				mt.AssertNotCalled(t, "Run")
			}
			if tt.expectClosed {
				mt.AssertCalled(t, "Close")
			} else {
				mt.AssertNotCalled(t, "Close")
			}
		})
	}
}
