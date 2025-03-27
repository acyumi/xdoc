package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/samber/oops"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/cloud"
	"github.com/acyumi/xdoc/component/feishu"
	"github.com/acyumi/xdoc/component/progress"
)

const (
	flagNameConfig            = "config"             //    --config
	flagNameVerbose           = "verbose"            // -V --verbose
	flagNameAppID             = "app-id"             //    --app
	flagNameAppSecret         = "app-secret"         //    --app
	flagNameURL               = "url"                //    --url
	flagNameDir               = "dir"                //    --dir
	flagNameExt               = "ext"                //    --ext
	flagNameFileExtensions    = "file.extensions"    //
	flagNameListOnly          = "list-only"          // -f --list-only
	flagNameQuitAutomatically = "quit-automatically" // -q --quit-automatically
)

var export = exportCommand(vip, args)

func exportCommand(vip *viper.Viper, args *argument.Args) *cobra.Command {
	export := &cobra.Command{
		Use:   "export",
		Short: "飞书云文档批量导出器",
		Long:  "这是飞书云文档批量导出、下载到本地的程序",
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := loadConfig(cmd, vip, args)
			if err != nil {
				return oops.Wrap(err)
			}
			return runE(args)
		},
	}
	err := setFlags(export, vip, args)
	if err != nil {
		panic(err)
	}
	return export
}

func init() {
	// 加到根命令中
	root.AddCommand(export)
}

func setFlags(cmd *cobra.Command, vip *viper.Viper, args *argument.Args) (err error) {
	// 添加 --config 参数
	cmd.PersistentFlags().StringVar(&args.ConfigFile, flagNameConfig, "", "指定配置文件(默认使用./config.yaml), 配置文件的参数会被命令行参数覆盖")
	// 添加命令行参数
	flags := cmd.Flags()
	flags.BoolP(flagNameVerbose, "V", false, "是否显示详细日志")
	flags.String(flagNameAppID, "", "飞书应用ID")
	flags.String(flagNameAppSecret, "", "飞书应用密钥")
	flags.String(flagNameURL, "", "文档地址, 如 https://sample.feishu.cn/wiki/MP4PwXweMi2FydkkG0ScNwBdnLz")
	flags.String(flagNameDir, "", "文档存放目录(本地)")
	flags.StringToString(flagNameExt, map[string]string{}, "文档扩展名映射, 用于指定文档下载后的文件类型, 对应配置文件file.extensions(如 docx=docx,doc=pdf)")
	flags.BoolP(flagNameListOnly, "l", false, "是否只列出云文档信息不进行导出下载")
	flags.BoolP(flagNameQuitAutomatically, "q", false, "是否在下载完成后自动退出程序")

	flags.VisitAll(func(flag *pflag.Flag) {
		if err != nil {
			return
		}
		switch flag.Name {
		case flagNameExt:
			// 不绑定，因为要实现 --ext 参数局部覆盖fileExtensions中的key的效果
			return
		default:
			err = vip.BindPFlag(flag.Name, flag)
		}
	})
	return oops.Wrapf(err, "绑定命令行参数到Viper失败")
}

func loadConfig(cmd *cobra.Command, vip *viper.Viper, args *argument.Args) (err error) {
	err = loadConfigFromFile(vip, args)
	if err != nil {
		var oe oops.OopsError
		if ok := errors.As(err, &oe); !ok || oe.Code() != "continue" {
			return oops.Wrapf(err, "加载配置文件失败")
		}
		fmt.Println(oe.Error())
	}
	// 从 Viper 中读取配置
	args.Verbose = vip.GetBool(flagNameVerbose)
	args.AppID = vip.GetString(flagNameAppID)
	args.AppSecret = vip.GetString(flagNameAppSecret)
	args.DocURL = vip.GetString(flagNameURL)
	args.SaveDir = vip.GetString(flagNameDir)
	args.SaveDir = filepath.Clean(args.SaveDir)
	args.SetFileExtensions(vip.GetStringMapString(flagNameFileExtensions))
	overrides, err := cmd.Flags().GetStringToString(flagNameExt)
	if err != nil {
		return oops.Wrap(err)
	}
	args.SetFileExtensions(overrides)
	args.ListOnly = vip.GetBool(flagNameListOnly)
	args.QuitAutomatically = vip.GetBool(flagNameQuitAutomatically)
	return nil
}

func runE(args *argument.Args) (err error) {
	args.StartTime = time.Now()
	defer func() {
		duration := time.Since(args.StartTime)
		fmt.Printf("完成飞书云文档操作, 总耗时: %s\n", duration.String())
		if err != nil {
			return
		}
		err, _ = recover().(error)
	}()
	fmt.Println("----------------------------------------------")
	fmt.Printf(" ConfigFile: %s\n", args.ConfigFile)
	fmt.Printf(" Verbose: %v\n", args.Verbose)
	fmt.Printf(" AppID: %s\n", args.Desensitize(args.AppID))
	fmt.Printf(" AppSecret: %s\n", args.Desensitize(args.AppSecret))
	fmt.Printf(" DocURL: %s\n", args.Desensitize(args.DocURL))
	fmt.Printf(" SaveDir: %s\n", args.SaveDir)
	fmt.Printf(" FileExtensions: %v\n", args.FileExtensions)
	fmt.Printf(" ListOnly: %v\n", args.ListOnly)
	fmt.Printf(" QuitAutomatically: %v\n", args.QuitAutomatically)
	fmt.Println("----------------------------------------------")
	if err = args.Validate(); err != nil {
		return oops.Wrap(err)
	}
	// 先通过文档地址获取文件类型和token
	host, typ, token, err := analysisURL(args.DocURL)
	if err != nil {
		return oops.Wrap(err)
	}
	fmt.Println("阶段1: 读取飞书云文档信息")
	fmt.Println("--------------------------")
	// 创建 Client
	client, err := newCloudClient(args, host)
	if err != nil {
		return oops.Wrap(err)
	}
	// 下载文档
	return client.DownloadDocuments(typ, token)
}

func loadConfigFromFile(vip *viper.Viper, args *argument.Args) error {
	// 如果指定了 --config 参数，则使用指定的配置文件
	if args.ConfigFile != "" {
		vip.SetConfigFile(args.ConfigFile)
	} else {
		// 默认从程序所在目录读取配置文件
		exePath, err := os.Executable()
		if err != nil {
			return oops.Wrapf(err, "获取程序所在目录失败")
		}
		exeDir := filepath.Dir(exePath)
		vip.AddConfigPath(exeDir)         // 添加程序所在目录为配置文件搜索路径
		vip.SetConfigName(flagNameConfig) // 配置文件名称（不带扩展名）
		vip.SetConfigType("yaml")         // 配置文件类型
		args.ConfigFile = filepath.Join(exeDir, flagNameConfig+".yaml")
	}
	// 读取配置文件
	if err := vip.ReadInConfig(); err != nil {
		var cfnfe viper.ConfigFileNotFoundError
		if ok := errors.As(err, &cfnfe); ok {
			// 配置文件不存在，忽略错误
			return oops.Code("continue").New("未找到配置文件, 将使用命令行参数，尝试使用命令行参数继续执行")
		}
		pathErr := &fs.PathError{}
		if ok := errors.As(err, &pathErr); ok {
			return oops.Code("continue").New("请检查配置文件权限，或者指定其他位置的配置文件，尝试使用命令行参数继续执行")
		}
		// 其他错误
		return oops.Wrap(err)
	}
	return nil
}

func analysisURL(docURL string) (host, typ, token string, err error) {
	// 文件夹 folder_token： https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0
	// 文件 file_token：https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1
	// 文档 doc_token：https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q
	// 新版文档 document_id：https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd
	// 电子表格 spreadsheet_token：https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY
	// 多维表格 app_token：https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz
	// 知识空间 space_id：https://sample.feishu.cn/wiki/settings/7075377271827264924（需要知识库管理员在设置页面获取该地址）
	// 知识库节点 node_token：https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
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
		return host, typ, token, oops.Code("BadRequest").Wrapf(err, "解析url地址失败：%s", docURL)
	}
	if URL.Scheme != "http" && URL.Scheme != "https" {
		return host, typ, token, oops.Code("BadRequest").Errorf("url地址必须是http://或https://开头")
	}
	host = strings.Split(URL.Host, ":")[0]
	path := URL.Path
	split := strings.Split(path, "/")
	if len(split) < 3 {
		return host, typ, token, oops.Code("BadRequest").
			New("url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q")
	}
	typ = strings.Join(split[0:len(split)-1], "/")
	token = split[len(split)-1]
	return host, typ, token, nil
}

func newCloudClient(args *argument.Args, host string) (cloud.Client, error) {
	switch {
	// TODO 可通过配置覆盖
	case strings.HasSuffix(host, "feishu.cn"):
		// 创建 飞书客户端
		return feishu.NewClient(args), nil
	case host == "progress.test":
		// 创建 进度条测试客户端
		return progress.NewTestClient(args), nil
	default:
		return nil, oops.Errorf("不支持的文档来源域名: %s", host)
	}
}
