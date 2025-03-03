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
