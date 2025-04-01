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
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/h2non/gock"
	"github.com/spf13/afero"

	"github.com/acyumi/xdoc/component/app"
)

var (
	mu                     sync.Mutex
	testSuiteAuthenticated bool // 是否已经模拟登录到飞书
)

// checkAuthenticated 检查是否已经登录，并不是所有单测都需要，但只需要登录一次即可，按需调用。
func checkAuthenticated() {
	mu.Lock()
	defer mu.Unlock()
	if !testSuiteAuthenticated {
		// 模拟获取 tenant_access_token 的响应
		// 飞书sdk中只会请求一次，然后缓存起来
		gock.New("https://open.feishu.cn").
			Post("/open-apis/auth/v3/tenant_access_token/internal").
			Reply(200).
			JSON(`{
    "code": 0,
    "msg": "ok",
    "tenant_access_token": "t-caecc734c2e3328a62489fe0648c4b98779515d3",
    "expire": 7200
}`)
		testSuiteAuthenticated = true
	}
}

func testInitBackOff(ebo *backoff.ExponentialBackOff) {
	ebo.InitialInterval = 10 * time.Millisecond
	ebo.Multiplier = 2.0
	ebo.MaxInterval = 100 * time.Millisecond
	ebo.RandomizationFactor = 0.2
	checkAuthenticated()
}

func useMemMapFs() {
	useFs(&afero.Afero{Fs: afero.NewMemMapFs()})
}

func useFs(fs *afero.Afero) {
	mu.Lock()
	defer mu.Unlock()
	app.Fs = fs
}

func cleanSleep() {
	mu.Lock()
	defer mu.Unlock()
	app.Sleep = func(duration time.Duration) { /* 单测时不需要睡眠等待 */ }
}
