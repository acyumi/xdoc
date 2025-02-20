package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"acyumi.com/feishu-doc-exporter/component/argument"
	"acyumi.com/feishu-doc-exporter/component/feishu"
)

const (
	flagNameConfig            = "config"             //    --config
	flagNameAppID             = "app-id"             //    --app
	flagNameAppSecret         = "app-secret"         //    --app
	flagNameURL               = "url"                //    --url
	flagNameDir               = "dir"                //    --dir
	flagNameExt               = "ext"                //    --ext
	flagNameFileExtensions    = "file.extensions"    //
	flagNameListOnly          = "list-only"          // -f --list-only
	flagNameQuitAutomatically = "quit-automatically" // -q --quit-automatically
)

var (
	cmd = &cobra.Command{
		Use:     "doc-exporter",
		Short:   "飞书云文档批量导出器",
		Long:    "这是飞书云文档批量导出、下载到本地的程序",
		Version: "0.0.1",
		RunE:    runE,
	}
)

func init() {
	// 添加 --config 参数
	cmd.PersistentFlags().StringVar(&argument.ConfigFile, flagNameConfig, "", "指定配置文件(默认使用./config.yaml), 配置文件的参数会被命令行参数覆盖")
	// 添加命令行参数
	flags := cmd.Flags()
	flags.String(flagNameAppID, "", "飞书应用ID")
	flags.String(flagNameAppSecret, "", "飞书应用密钥")
	flags.String(flagNameURL, "", "文档地址, 如 https://sample.feishu.cn/wiki/MP4PwXweMi2FydkkG0ScNwBdnLz")
	flags.String(flagNameDir, "", "文档存放目录(本地)")
	flags.StringToString(flagNameExt, map[string]string{}, "文档扩展名映射, 用于指定文档下载后的文件类型, 对应配置文件file.extensions(如 docx=docx,doc=pdf)")
	flags.BoolP(flagNameListOnly, "l", false, "是否只列出云文档信息不进行导出下载")
	flags.BoolP(flagNameQuitAutomatically, "q", false, "是否在下载完成后自动退出程序")

	var err error
	flags.VisitAll(func(flag *pflag.Flag) {
		if err != nil {
			return
		}
		switch flag.Name {
		case flagNameExt:
			// 不绑定，因为要实现 --ext 参数局部覆盖fileExtensions中的key的效果
			return
		default:
			err = viper.BindPFlag(flag.Name, flag)
		}
	})
	if err != nil {
		fmt.Printf("绑定命令行参数到Viper失败: %+v\n", err)
		return
	}
}

// SDK 使用文档：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/server-side-sdk/golang-sdk-guide/preparations
func main() {
	// TODO 参数校验
	// TODO 补单测
	// TODO 执行日志输出到文件
	// TODO docx 和 pdf 下载后自动去除水印
	// TODO 下载UI程序支持快速滚动到顶部和底部、按ctrl+↑向上滚动10%、按ctrl+↓向下滚动10%
	// TODO 补充更多使用说明，如主程序参数、下载UI状态下的快捷键说明
	// TODO 支持多个url下载
	// TODO 添加支持云空间的文件导出下载
	// TODO 支持跳过已下载文件（将下载进度保存到缓存文件中，每次执行都做一下检查）
	// TODO 添加使用golangci-lint
	// TODO 添加readme.md文档，上传到github
	// 执行命令
	err := cmd.Execute()
	if err != nil {
		fmt.Println("----------------------------------------------")
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

func runE(cmd *cobra.Command, _ []string) error {
	// 加载配置
	err := loadConfig()
	if err != nil {
		return oops.Wrapf(err, "加载配置文件失败")
	}
	// 从 Viper 中读取配置
	argument.AppID = viper.GetString(flagNameAppID)
	argument.AppSecret = viper.GetString(flagNameAppSecret)
	argument.DocURL = viper.GetString(flagNameURL)
	argument.SaveDir = viper.GetString(flagNameDir)
	argument.SaveDir = filepath.Clean(argument.SaveDir)
	argument.FileExtensions = viper.GetStringMapString(flagNameFileExtensions)
	overrides, err := cmd.Flags().GetStringToString(flagNameExt)
	if err != nil {
		return oops.Wrap(err)
	}
	argument.FileExtensions = lo.Assign(argument.FileExtensions, overrides)
	argument.ListOnly = viper.GetBool(flagNameListOnly)
	argument.QuitAutomatically = viper.GetBool(flagNameQuitAutomatically)
	fmt.Println("----------------------------------------------")
	fmt.Printf(" ConfigFile: %s\n", argument.ConfigFile)
	fmt.Printf(" AppID: %s\n", argument.AppID)
	fmt.Printf(" AppSecret: %s\n", argument.AppSecret)
	fmt.Printf(" DocURL: %s\n", argument.DocURL)
	fmt.Printf(" SaveDir: %s\n", argument.SaveDir)
	fmt.Printf(" FileExtensions: %v\n", argument.FileExtensions)
	fmt.Printf(" ListOnly: %v\n", argument.ListOnly)
	fmt.Printf(" QuitAutomatically: %v\n", argument.QuitAutomatically)
	fmt.Println("----------------------------------------------")
	if err = argument.Validate(); err != nil {
		return oops.Wrap(err)
	}
	// 先通过文档地址获取文件类型和token
	docType, token, err := analysisURL(argument.DocURL)
	if err != nil {
		return oops.Wrap(err)
	}
	fmt.Println("阶段1: 读取飞书云文档信息")
	fmt.Println("--------------------------")
	argument.StartTime = time.Now()
	defer func() {
		var ok bool
		if err, ok = recover().(error); ok {
			return
		}
		duration := time.Since(argument.StartTime)
		fmt.Printf("完成飞书云文档操作, 总耗时: %s\n", duration.String())
	}()
	// 创建 Client
	client := lark.NewClient(argument.AppID, argument.AppSecret)
	switch docType {
	case "/wiki":
		fmt.Printf("飞书云文档源: 知识库, 类型: %s, token: %s\n", docType, token)
		err = feishu.DownloadWikiDocuments(client, token)
	case "/wiki/settings":
		fmt.Printf("飞书云文档源: 知识库, 类型: %s, token: %s\n", docType, token)
		err = feishu.DownloadWikiSpaceDocuments(client, token)
	case "/drive/folder", "/docs", "/docx", "/sheets", "/file":
		fmt.Printf("飞书云文档源: 云空间, 类型: %s, token: %s\n", docType, token)
		var typ string
		switch docType {
		case "/docs":
			typ = "doc"
		case "/docx":
			typ = "docx"
		case "/sheets":
			typ = "sheet"
		case "/file":
			typ = "file"
		case "/drive/folder":
			typ = "folder"
		default:
			fmt.Printf("不支持的飞书云文档类型: %s\n", docType)
		}
		err = feishu.DownloadDriveDocuments(client, typ, token)
	default:
		fmt.Printf("不支持的飞书云文档类型: %s\n", docType)
	}
	return oops.Wrap(err)
}

func loadConfig() error {
	// 如果指定了 --config 参数，则使用指定的配置文件
	if argument.ConfigFile != "" {
		viper.SetConfigFile(argument.ConfigFile)
	} else {
		// 默认从程序所在目录读取配置文件
		exePath, err := os.Executable()
		if err != nil {
			return oops.Wrapf(err, "获取程序所在目录失败")
		}
		exeDir := filepath.Dir(exePath)
		viper.AddConfigPath(exeDir)         // 添加程序所在目录为配置文件搜索路径
		viper.SetConfigName(flagNameConfig) // 配置文件名称（不带扩展名）
		viper.SetConfigType("yaml")         // 配置文件类型
		argument.ConfigFile = filepath.Join(exeDir, flagNameConfig+".yaml")
	}
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		var cfnfe viper.ConfigFileNotFoundError
		if ok := errors.As(err, &cfnfe); ok {
			// 配置文件不存在，忽略错误
			fmt.Println("未找到配置文件, 将使用命令行参数")
			return nil
		}
		pathErr := &fs.PathError{}
		if ok := errors.As(err, &pathErr); ok {
			fmt.Println("请检查配置文件权限，或者指定其他位置的配置文件")
			return nil
		}
		// 其他错误
		return oops.Wrap(err)
	}
	return nil
}

func analysisURL(docURL string) (docType, token string, err error) {
	//文件夹 folder_token： https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0
	//文件 file_token：https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1
	//文档 doc_token：https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q
	//新版文档 document_id：https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd
	//电子表格 spreadsheet_token：https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY
	//多维表格 app_token：https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz
	//知识空间 space_id：https://sample.feishu.cn/wiki/settings/7075377271827264924（需要知识库管理员在设置页面获取该地址）
	//知识库节点 node_token：https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
	//
	// https://root:123456@baidu.com:443?dddd=oo&uuu=55#/adfadf/fade
	// temp.Scheme = "https"
	// temp.Opaque = ""
	// temp.User = {username:root,password:123456}
	// temp.Host = "baidu.com:443"
	// addressURL.Path = ""
	// addressURL.RawPath = ""
	// addressURL.OmitHost = false
	// addressURL.ForceQuery = false
	// addressURL.RawQuery = "dddd=oo&uuu=55"
	// addressURL.Fragment = "/adfadf/fade"
	// addressURL.RawFragment = ""
	URL, err := url.Parse(docURL)
	if err != nil {
		return "", "", oops.Code("BadRequest").Wrapf(err, "解析url地址失败：%s", docURL)
	}
	path := URL.Path
	split := strings.Split(path, "/")
	if len(split) < 3 {
		return "", "", oops.Code("BadRequest").
			New("url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q")
	}
	docType = strings.Join(split[0:len(split)-1], "/")
	token = split[len(split)-1]
	return docType, token, nil
}
