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

package cmd

import (
	"github.com/samber/oops"
	"github.com/spf13/viper"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
)

// Execute 基于接口定义统一初始化和绑定参数后执行。
func Execute(root command) (*argument.Args, error) {
	vip := app.NewViper()
	args := &argument.Args{}
	err := prepareCommand(root, vip, args)
	if err != nil {
		return args, oops.Wrap(err)
	}
	err = root.get().Execute()
	return args, oops.Wrap(err)
}

func prepareCommand(cmd command, vip *viper.Viper, args *argument.Args) error {
	cmd.init(vip, args)
	err := cmd.bind()
	if err != nil {
		return oops.Wrap(err)
	}
	c := cmd.get()
	for _, child := range cmd.children() {
		err = prepareCommand(child, vip, args)
		if err != nil {
			return oops.Wrap(err)
		}
		cc := child.get()
		if c != cc {
			c.AddCommand(cc)
		}
	}
	return nil
}
