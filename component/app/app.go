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

package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var (
	MarshalIndent = json.MarshalIndent
	Executable    = os.Executable

	Fs    = &afero.Afero{Fs: afero.NewOsFs()}
	Sleep = func(duration time.Duration) { time.Sleep(duration) } // 睡眠等待函数
)

func NewViper() *viper.Viper {
	vip := viper.New()
	vip.SetFs(Fs)
	return vip
}

func Fprintln(out io.Writer, a ...any) {
	_, _ = fmt.Fprintln(out, a...)
}

func Fprint(out io.Writer, a ...any) {
	_, _ = fmt.Fprint(out, a...)
}

func Fprintf(out io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(out, format, a...)
}
