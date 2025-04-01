package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/h2non/gock"
	"github.com/samber/oops"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/constant"
	"github.com/acyumi/xdoc/component/feishu"
	"github.com/acyumi/xdoc/component/progress"
)

// 注册测试套件。
func TestExporterSuite(t *testing.T) {
	suite.Run(t, new(ExporterTestSuite))
}

type ExporterTestSuite struct {
	suite.Suite
	TempDir string
}

func (s *ExporterTestSuite) SetupTest() {
	s.TempDir = s.T().TempDir()
}

func (s *ExporterTestSuite) TearDownTest() {

}

func (s *ExporterTestSuite) TestExecute() {
	oc := export
	defer func() {
		export = oc
	}()
	tests := []struct {
		name           string
		configFile     string
		configContent  []byte
		args           []string
		wantConfigFile string
		wantAppID      string
		wantAppSecret  string
		wantDocURLs    []string
		wantSaveDir    string
		wantError      string
		wantCode       string
	}{
		{
			name:          "纯命令参数执行",
			configFile:    "",
			configContent: nil,
			args: []string{
				"--app-id", "xx",
				"--app-secret", "yy",
				"--urls", "https://invalid.cn/docs/xxx",
				"--dir", "/tmp",
				"--list-only", "true",
				"--quit-automatically", "true",
				"--ext", "docx=docx,doc=docx",
			},
			wantConfigFile: "",
			wantAppID:      "xx",
			wantAppSecret:  "yy",
			wantDocURLs:    []string{"https://invalid.cn/docs/xxx"},
			wantSaveDir:    filepath.Clean("/tmp"),
			wantError:      "不支持的文档来源域名: invalid.cn",
		},
		{
			name:       "指定有效配置文件",
			configFile: filepath.Join(s.TempDir, "test.yaml"),
			configContent: []byte(`
app-id: "xx"
app-secret: "yy"
urls: ["https://invalid.cn/docs/xxx"]
dir: "/tmp"
list-only: true
quit-automatically: true
file:
  extensions:
    docx: "docx"
    doc: "docx"
`),
			args:           []string{"--config", filepath.Join(s.TempDir, "test.yaml")},
			wantConfigFile: filepath.Join(s.TempDir, "test.yaml"),
			wantAppID:      "xx",
			wantAppSecret:  "yy",
			wantDocURLs:    []string{"https://invalid.cn/docs/xxx"},
			wantSaveDir:    filepath.Clean("/tmp"),
			wantError:      "不支持的文档来源域名: invalid.cn",
		},
		{
			name:       "默认配置文件",
			configFile: "",
			configContent: []byte(`
app-id: "xx"
app-secret: "yy"
urls: ["https://invalid.cn/docs/xxx"]
dir: "/tmp"
list-only: true
quit-automatically: true
file:
  extensions:
    docx: "docx"
    doc: "docx"
`),
			wantConfigFile: func() string {
				exePath, err := os.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				return filepath.Join(exeDir, flagNameConfig+".yaml")
			}(),
			wantAppID:     "xx",
			wantAppSecret: "yy",
			wantDocURLs:   []string{"https://invalid.cn/docs/xxx"},
			wantSaveDir:   filepath.Clean("/tmp"),
			wantError:     "不支持的文档来源域名: invalid.cn",
		},
		{
			name:       "不支持的schema",
			configFile: "",
			configContent: []byte(`
app-id: "xx"
app-secret: "yy"
urls: ["feishu://invalid.cn/docs/xxx"]
dir: "/tmp"
list-only: true
quit-automatically: true
file:
  extensions:
    docx: "docx"
    doc: "docx"
`),
			wantConfigFile: func() string {
				exePath, err := os.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				return filepath.Join(exeDir, flagNameConfig+".yaml")
			}(),
			wantAppID:     "xx",
			wantAppSecret: "yy",
			wantDocURLs:   []string{"feishu://invalid.cn/docs/xxx"},
			wantSaveDir:   filepath.Clean("/tmp"),
			wantError:     "url地址必须是http://或https://开头",
			wantCode:      "BadRequest",
		},
		{
			name:       "不支持的ext配置",
			configFile: "",
			configContent: []byte(`
app-id: "xx"
app-secret: "yy"
urls: ["https://invalid.cn/docs/xxx"]
dir: "/tmp"
list-only: true
quit-automatically: true
file:
  extensions: 
    docx: "docx"
    doc: "docx"
`),
			args: []string{"--ext", "xxx"},
			wantConfigFile: func() string {
				exePath, err := os.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				return filepath.Join(exeDir, flagNameConfig+".yaml")
			}(),
			wantAppID:     "",
			wantAppSecret: "",
			wantDocURLs:   nil,
			wantSaveDir:   "",
			wantError:     "invalid argument \"xxx\" for \"--ext\" flag: xxx must be formatted as key=value",
			wantCode:      "",
		},
		{
			name:           "配置文件不存在",
			configFile:     "nonexistent.yaml",
			wantConfigFile: "nonexistent.yaml",
			wantAppID:      "",
			wantAppSecret:  "",
			wantDocURLs:    []string{},
			wantSaveDir:    ".",
			wantError:      "AppID: app-id是必需参数; AppSecret: app-secret是必需参数; DocURLs: urls是必需参数.",
			wantCode:       "InvalidArgument",
		},
		{
			name:           "配置文件类型不对",
			configFile:     filepath.Join(s.TempDir, "ttt.exe"),
			configContent:  []byte(`xxx`),
			args:           []string{"--config", filepath.Join(s.TempDir, "ttt.exe")},
			wantConfigFile: filepath.Join(s.TempDir, "ttt.exe"),
			wantAppID:      "",
			wantAppSecret:  "",
			wantDocURLs:    nil,
			wantSaveDir:    "",
			wantError:      "加载配置文件失败: Unsupported Config Type \"exe\"",
			wantCode:       "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tempFile := tt.configFile
			if tempFile == "" && tt.configContent != nil {
				exePath, err := os.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				tempFile = filepath.Join(exeDir, flagNameConfig+".yaml")
			}
			if tempFile != "" && tt.configContent != nil {
				err := app.Fs.WriteFile(tempFile, tt.configContent, 0644)
				s.Require().NoError(err, "创建测试配置文件失败")
			}
			vip := viper.New()
			args := &argument.Args{}
			root = rootCommand()
			export = exportCommand(vip, args)
			root.AddCommand(export)
			// 这里不能传 nil，因为这会让 cobra 取了 IDE 的参数影响单测，如 Error: unknown shorthand flag: 't' in -testify.m
			if tt.args == nil {
				tt.args = []string{}
			}
			// export.Execute() 内会递归到根命令再执行，跟执行 root.Execute() 的效果是一样的，但是参数要从 root 那里传递
			tt.args = append([]string{"export"}, tt.args...)
			root.SetArgs(tt.args)
			err := export.Execute()
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.Require().EqualError(err, tt.wantError, tt.name)
				var oe oops.OopsError
				if ok := errors.As(err, &oe); ok {
					s.Equal(tt.wantCode, oe.Code(), tt.name)
				}
			} else {
				s.Require().NoError(err, tt.name)
			}
			s.Equal(tt.wantConfigFile, tempFile, tt.name)
			s.Equal(tt.wantAppID, args.AppID, tt.name)
			s.Equal(tt.wantAppSecret, args.AppSecret, tt.name)
			s.Equal(tt.wantDocURLs, args.DocURLs, tt.name)
			s.Equal(tt.wantSaveDir, args.SaveDir, tt.name)
			yes, err := app.Fs.Exists(tempFile)
			s.Require().NoError(err, tt.name)
			if yes {
				err = app.Fs.Remove(tempFile)
				s.Require().NoError(err, tt.name)
			}
		})
	}
}

func (s *ExporterTestSuite) Test_runE() {
	tests := []struct {
		name         string
		args         *argument.Args
		setupMock    func(name string)
		teardownMock func(name string)
		wantError    string
		wantCode     string
	}{
		{
			name: "指定有效配置文件",
			args: &argument.Args{
				AppID:             "xx",
				AppSecret:         "yy",
				DocURLs:           []string{"https://invalid.cn/docs/xxx"},
				SaveDir:           "/tmp",
				ListOnly:          true,
				QuitAutomatically: true,
				FileExtensions: map[constant.DocType]constant.FileExt{
					"docx": "docx",
					"doc":  "docx",
				},
			},
			setupMock:    func(name string) {},
			teardownMock: func(name string) {},
			wantError:    "不支持的文档来源域名: invalid.cn",
			wantCode:     "",
		},
		{
			name: "文档地址不是同一域名",
			args: &argument.Args{
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs: []string{
					"https://invalid.cn/docs/xxx",
					"https://xxxyyy.feishu.cn/docs/xxx",
				},
				SaveDir:           "/tmp",
				ListOnly:          true,
				QuitAutomatically: true,
				FileExtensions: map[constant.DocType]constant.FileExt{
					"docx": "docx",
					"doc":  "docx",
				},
			},
			setupMock:    func(name string) {},
			teardownMock: func(name string) {},
			wantError:    "文档地址不匹配, 请确保所有文档地址都是同一域名",
			wantCode:     "",
		},
		{
			name: "执行下载报错",
			args: &argument.Args{
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs: []string{
					"https://xxxyyy.feishu.cn/docs/xxx",
				},
				SaveDir:           "/tmp",
				ListOnly:          true,
				QuitAutomatically: true,
				FileExtensions: map[constant.DocType]constant.FileExt{
					"docx": "docx",
					"doc":  "docx",
				},
			},
			setupMock: func(name string) {
				// 模拟获取 tenant_access_token 的响应
				gock.New("https://open.feishu.cn").
					Post("/open-apis/auth/v3/tenant_access_token/internal").
					Reply(500).
					JSON(`{"code":500,"msg":"模拟请求失败"}`)
			},
			teardownMock: func(name string) {
				gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "msg:模拟请求失败,code:500",
			wantCode:  "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name)
			err := runE(tt.args)
			tt.teardownMock(tt.name)
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.Require().EqualError(err, tt.wantError, tt.name)
				var oe oops.OopsError
				if ok := errors.As(err, &oe); ok {
					s.Equal(tt.wantCode, oe.Code(), tt.name)
				}
			} else {
				s.Require().NoError(err, tt.name)
			}
		})
	}
}

// 测试配置文件加载。
func (s *ExporterTestSuite) Test_loadConfigFromFile() {
	tests := []struct {
		name          string
		configFile    string
		configContent []byte
		wantConfig    string
		wantAppID     string
		wantError     string
		wantCode      string
	}{
		{
			name:       "指定有效配置文件",
			configFile: filepath.Join(s.TempDir, "test.yaml"),
			configContent: []byte(`
app-id: config_app
file:
  extensions:
    doc: pdf
`),
			wantConfig: filepath.Join(s.TempDir, "test.yaml"),
			wantAppID:  "config_app",
		},
		{
			name:       "默认配置文件",
			configFile: "",
			configContent: []byte(`
app-id: default_app
`),
			wantConfig: func() string {
				exePath, err := os.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				return filepath.Join(exeDir, flagNameConfig+".yaml")
			}(),
			wantAppID: "default_app",
		},
		{
			name:       "配置文件不存在",
			configFile: "nonexistent.yaml",
			wantConfig: "nonexistent.yaml",
			wantAppID:  "",
			wantError:  "请检查配置文件权限，或者指定其他位置的配置文件，尝试使用命令行参数继续执行",
			wantCode:   "continue",
		},
		{
			name:          "配置文件类型不对",
			configFile:    filepath.Join(s.TempDir, "ttt.exe"),
			configContent: []byte(`xxx`),
			wantConfig:    filepath.Join(s.TempDir, "ttt.exe"),
			wantAppID:     "",
			wantError:     "Unsupported Config Type \"exe\"",
			wantCode:      "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			configFile := tt.configFile
			if tt.configFile == "" && tt.configContent != nil {
				exePath, err := os.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				configFile = filepath.Join(exeDir, flagNameConfig+".yaml")
			}
			if configFile != "" && tt.configContent != nil {
				err := app.Fs.WriteFile(configFile, tt.configContent, 0644)
				s.Require().NoError(err, "创建测试配置文件失败")
			}
			vip := viper.New()
			args := &argument.Args{ConfigFile: tt.configFile}
			err := loadConfigFromFile(vip, args)
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				var oe oops.OopsError
				ok := errors.As(err, &oe)
				s.True(ok, tt.name)
				s.Require().EqualError(err, tt.wantError, tt.name)
				s.Equal(tt.wantCode, oe.Code(), tt.name)
			} else {
				s.Require().NoError(err, tt.name)
			}
			s.Equal(tt.wantConfig, args.ConfigFile, tt.name)
			s.Equal(tt.wantAppID, vip.GetString(flagNameAppID), tt.name)
			yes, err := app.Fs.Exists(args.ConfigFile)
			s.Require().NoError(err, tt.name)
			if yes {
				err = app.Fs.Remove(args.ConfigFile)
				s.Require().NoError(err, tt.name)
			}
		})
	}
}

func (s *ExporterTestSuite) Test_analysisURL() {
	// 文件夹 folder_token： https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0
	// 文件 file_token：https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1
	// 文档 doc_token：https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q
	// 新版文档 document_id：https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd
	// 电子表格 spreadsheet_token：https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY
	// 多维表格 app_token：https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz
	// 知识空间 space_id：https://sample.feishu.cn/wiki/settings/7075377271827264924（需要知识库管理员在设置页面获取该地址）
	// 知识库节点 node_token：https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
	type args struct {
		docURL string
	}
	tests := []struct {
		name      string
		args      args
		wantHost  string
		wantType  string
		wantToken string
		wantError string
		wantCode  string
	}{
		{
			name: "url非法0",
			args: args{
				docURL: "+%%",
			},
			wantHost:  "",
			wantType:  "",
			wantToken: "",
			wantError: `解析url地址失败：+%%: parse "+%%": invalid URL escape "%%"`,
			wantCode:  "BadRequest",
		},
		{
			name: "url非法1",
			args: args{
				docURL: "https://sample.feishu.cn",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "",
			wantToken: "",
			wantError: "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:  "BadRequest",
		},
		{
			name: "url非法2",
			args: args{

				docURL: "https://sample.feishu.cn/",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "",
			wantToken: "",
			wantError: "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:  "BadRequest",
		},
		{
			name: "url非法3",
			args: args{
				docURL: "https://sample.feishu.cn/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "",
			wantToken: "",
			wantError: "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:  "BadRequest",
		},
		{
			name: "文件夹",
			args: args{
				docURL: "https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/drive/folder",
			wantToken: "cSJe2JgtFFBwRuTKAJK6baNGUn0",
			wantError: "",
		},
		{
			name: "文件",
			args: args{
				docURL: "https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/file",
			wantToken: "ndqUw1kpjnGNNaegyqDyoQDCLx1",
			wantError: "",
		},
		{
			name: "文档",
			args: args{
				docURL: "https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/docs",
			wantToken: "2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantError: "",
		},
		{
			name: "新版文档",
			args: args{
				docURL: "https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/docx",
			wantToken: "UXEAd6cRUoj5pexJZr0cdwaFnpd",
			wantError: "",
		},
		{
			name: "电子表格",
			args: args{
				docURL: "https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/sheets",
			wantToken: "MRLOWBf6J47ZUjmwYRsN8utLEoY",
			wantError: "",
		},
		{
			name: "多维表格",
			args: args{
				docURL: "https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/base",
			wantToken: "Pc9OpwAV4nLdU7lTy71t6Kmmkoz",
			wantError: "",
		},
		{
			name: "知识空间",
			args: args{
				docURL: "https://sample.feishu.cn/wiki/settings/7075377271827264924",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/wiki/settings",
			wantToken: "7075377271827264924",
			wantError: "",
		},
		{
			name: "知识库节点",
			args: args{
				docURL: "https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/wiki",
			wantToken: "sZdeQp3m4nFGzwqR5vx4vZksMoe",
			wantError: "",
		},
	}
	// logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// logger := slog.Default()
	for _, tt := range tests {
		s.Run(tt.name, func() {
			gotHost, gotType, gotToken, err := analysisURL(tt.args.docURL)
			if err != nil || tt.wantError != "" {
				msg := err.Error()
				s.Equal(tt.wantError, msg)
				oopsError, ok := oops.AsOops(err)
				if ok {
					s.Equal(tt.wantCode, oopsError.Code())
				}
				// 报错时打印日志方便排查
				// fmt.Printf("%+v", err)
				// logger.Error(msg, slog.Any("error", err))
			}
			s.Equal(tt.wantHost, gotHost)
			s.Equal(tt.wantType, gotType)
			s.Equal(tt.wantToken, gotToken)
		})
	}
}

// 测试newCloudClient。
func (s *ExporterTestSuite) Test_newCloudClient() {
	tests := []struct {
		name       string
		host       string
		args       *argument.Args
		wantClient any
		wantError  string
	}{
		{
			name: "飞书客户端",
			host: "sample.feishu.cn",
			args: &argument.Args{
				AppID:     "1111",
				AppSecret: "2222",
			},
			wantClient: &feishu.ClientImpl{},
		},
		{
			name: "进度条UI测试客户端",
			host: "progress.test",
			args: &argument.Args{
				AppID:     "1111",
				AppSecret: "2222",
			},
			wantClient: &progress.TestClient{},
		},
		{
			name:      "不支持",
			host:      "invalid.com",
			wantError: "不支持的文档来源域名: invalid.com",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			actual, err := newCloudClient(tt.args, tt.host)
			if tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.Require().EqualError(err, tt.wantError, tt.name)
				s.Nil(actual, tt.name)
				return
			}
			s.Require().NoError(err, tt.name)
			s.NotNil(actual, tt.name)
			s.IsType(tt.wantClient, actual, tt.name)
		})
	}
}
