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

package progress

import (
	"io"
	"os"
	"path/filepath"

	"github.com/samber/oops"

	"github.com/acyumi/xdoc/component/app"
)

type Writer struct {
	FileKey  string   // 文件key
	FilePath string   // 文件写入路径
	Program  IProgram // 程序，显示进度
	Total    int64    // 文件总大小
	Wrote    int64    // 文件已写盘大小
	Walked   float64  // 文件写盘前进度条已走过的占比，如 0.2
}

func (pw *Writer) WriteFile(reader io.Reader) error {
	// 创建目录
	dirPath := filepath.Dir(pw.FilePath)
	err := app.Fs.MkdirAll(dirPath, 0o755)
	if err != nil {
		return oops.Wrap(err)
	}
	// 保存文件
	file, err := app.Fs.OpenFile(pw.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return oops.Wrap(err)
	}
	// 将数据写入文件，同时更新进度
	teeReader := io.TeeReader(reader, pw)
	_, err = io.Copy(file, teeReader)
	if er := file.Close(); er != nil && err == nil {
		err = er
	}
	return oops.Wrap(err)
}

// Write 实现了 io.Writer 接口。
func (pw *Writer) Write(p []byte) (int, error) {
	n := len(p)
	pw.Wrote += int64(n)
	pg := pw.Progress()
	// 更新进度
	pw.Program.Update(pw.FilePath, pg, StatusDownloading, "total: %d, wrote: %d", pw.Total, pw.Wrote)
	return n, nil
}

func (pw *Writer) Progress() float64 {
	return pw.Walked + float64(pw.Wrote)/float64(pw.Total)*(1.0-pw.Walked)
}
