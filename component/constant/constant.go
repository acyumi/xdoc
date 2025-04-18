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

package constant

const (
	Windows = "windows"
)

// DocType 文档类型。
type DocType string

const (
	DocTypeFolder   DocType = "folder"
	DocTypeFile     DocType = "file"
	DocTypeShortcut DocType = "shortcut"
	DocTypeDoc      DocType = "doc"
	DocTypeDocx     DocType = "docx"
	DocTypePDF      DocType = "pdf"
	DocTypeBitable  DocType = "bitable"
	DocTypeSheet    DocType = "sheet"
	DocTypeXls      DocType = "xls"
	DocTypeXlsx     DocType = "xlsx"
	DocTypeMindNote DocType = "mindnote"
	DocTypeSlides   DocType = "slides"
)

// FileExt 下载后保存到本地的文件扩展名。
type FileExt string

const (
	FileExtDocx FileExt = "docx"
	FileExtPDF  FileExt = "pdf"
	FileExtXlsx FileExt = "xlsx"
	FileExtCSV  FileExt = "csv"
)
