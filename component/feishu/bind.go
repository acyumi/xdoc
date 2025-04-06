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
	"fmt"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/samber/oops"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/constant"
)

type Args struct {
	*argument.Args
	Enabled        bool                                  // 是否启用
	AppID          string                                // 应用ID
	AppSecret      string                                // 应用密钥
	DocURLs        []string                              // 文档地址
	SaveDir        string                                // 文档存放目录(本地)
	FileExtensions map[constant.DocType]constant.FileExt // 文档扩展名映射, 用于指定文档下载后的文件类型
	ListOnly       bool                                  // 是否只列出云文档信息不进行导出下载
}

func (a Args) Validate() error {
	return oops.Code("InvalidArgument").Wrap(
		validation.ValidateStruct(&a,
			validation.Field(&a.AppID, validation.Required.Error("app-id是必需参数")),
			validation.Field(&a.AppSecret, validation.Required.Error("app-secret是必需参数")),
			validation.Field(&a.DocURLs, validation.Required.Error("urls是必需参数")),
			validation.Field(&a.SaveDir, validation.Required.Error("dir是必需参数")),
		))
}

func (a *Args) SetFileExtensions(fes map[string]string) {
	if a.FileExtensions == nil {
		a.FileExtensions = map[constant.DocType]constant.FileExt{}
	}
	for k, v := range fes {
		a.FileExtensions[constant.DocType(k)] = constant.FileExt(v)
	}
}

func (a *Args) DesensitizeSlice(str ...string) (res []string) {
	for _, s := range str {
		s = a.Desensitize(s)
		res = append(res, s)
	}
	return res
}

func (a *Args) Desensitize(str string) string {
	if a.Verbose {
		return str
	}
	if len(str) < 4 {
		return str
	}
	if strings.Contains(str, "http://") || strings.Contains(str, "https://") {
		// 将域名脱敏
		// https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
		// -> https://sam***.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
		split := strings.Split(str, "/")
		http := split[0]
		host := split[2]
		if hostSplit := strings.Split(host, "."); len(hostSplit[0]) > 3 {
			hostSplit[0] = hostSplit[0][0:3] + strings.Repeat("*", len(hostSplit[0])-3)
			host = strings.Join(hostSplit, ".")
		}
		var middle string
		if len(split) < 6 {
			middle = split[3]
		} else {
			middle = strings.Join(split[3:len(split)-1], "/")
		}
		token := split[len(split)-1]
		return fmt.Sprintf("%s//%s/%s/%s", http, host, middle, token)
	}
	return str[0:4] + strings.Repeat("*", len(str)-4)
}
