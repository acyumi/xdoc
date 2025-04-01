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
	"time"

	"github.com/spf13/afero"
)

var (
	MarshalIndent = json.MarshalIndent

	Fs    = &afero.Afero{Fs: afero.NewOsFs()}
	Sleep = func(duration time.Duration) { time.Sleep(duration) } // 睡眠等待函数
)
