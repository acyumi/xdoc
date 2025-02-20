package main

import (
	"testing"

	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
)

func Test_analysis(t *testing.T) {
	//文件夹 folder_token： https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0
	//文件 file_token：https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1
	//文档 doc_token：https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q
	//新版文档 document_id：https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd
	//电子表格 spreadsheet_token：https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY
	//多维表格 app_token：https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz
	//知识空间 space_id：https://sample.feishu.cn/wiki/settings/7075377271827264924（需要知识库管理员在设置页面获取该地址）
	//知识库节点 node_token：https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
	type args struct {
		docURL string
	}
	tests := []struct {
		name        string
		args        args
		wantDocType string
		wantToken   string
		wantErr     string
		wantCode    string
	}{
		{
			name: "url非法1",
			args: args{
				docURL: "https://sample.feishu.cn",
			},
			wantDocType: "",
			wantToken:   "",
			wantErr:     "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:    "BadRequest",
		},
		{
			name: "url非法2",
			args: args{

				docURL: "https://sample.feishu.cn/",
			},
			wantDocType: "",
			wantToken:   "",
			wantErr:     "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:    "BadRequest",
		},
		{
			name: "url非法3",
			args: args{
				docURL: "https://sample.feishu.cn/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			},
			wantDocType: "",
			wantToken:   "",
			wantErr:     "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:    "BadRequest",
		},
		{
			name: "文件夹",
			args: args{
				docURL: "https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			},
			wantDocType: "/drive/folder",
			wantToken:   "cSJe2JgtFFBwRuTKAJK6baNGUn0",
			wantErr:     "",
		},
		{
			name: "文件",
			args: args{
				docURL: "https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1",
			},
			wantDocType: "/file",
			wantToken:   "ndqUw1kpjnGNNaegyqDyoQDCLx1",
			wantErr:     "",
		},
		{
			name: "文档",
			args: args{
				docURL: "https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			},
			wantDocType: "/docs",
			wantToken:   "2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantErr:     "",
		},
		{
			name: "新版文档",
			args: args{
				docURL: "https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd",
			},
			wantDocType: "/docx",
			wantToken:   "UXEAd6cRUoj5pexJZr0cdwaFnpd",
			wantErr:     "",
		},
		{
			name: "电子表格",
			args: args{
				docURL: "https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY",
			},
			wantDocType: "/sheets",
			wantToken:   "MRLOWBf6J47ZUjmwYRsN8utLEoY",
			wantErr:     "",
		},
		{
			name: "多维表格",
			args: args{
				docURL: "https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz",
			},
			wantDocType: "/base",
			wantToken:   "Pc9OpwAV4nLdU7lTy71t6Kmmkoz",
			wantErr:     "",
		},
		{
			name: "知识空间",
			args: args{
				docURL: "https://sample.feishu.cn/wiki/settings/7075377271827264924",
			},
			wantDocType: "/wiki/settings",
			wantToken:   "7075377271827264924",
			wantErr:     "",
		},
		{
			name: "知识库节点",
			args: args{
				docURL: "https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe",
			},
			wantDocType: "/wiki",
			wantToken:   "sZdeQp3m4nFGzwqR5vx4vZksMoe",
			wantErr:     "",
		},
	}
	//logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	//logger := slog.Default()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDocType, gotToken, err := analysisURL(tt.args.docURL)
			if err != nil {
				msg := err.Error()
				assert.Equal(t, tt.wantErr, msg)
				oopsError, ok := oops.AsOops(err)
				if ok {
					assert.Equal(t, tt.wantCode, oopsError.Code())
				}
				// 报错时打印日志方便排查
				// fmt.Printf("%+v", err)
				// logger.Error(msg, slog.Any("error", err))
			}
			assert.Equal(t, tt.wantDocType, gotDocType)
			assert.Equal(t, tt.wantToken, gotToken)
		})
	}
}
