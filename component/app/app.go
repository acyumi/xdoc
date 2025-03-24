package app

import (
	"time"

	"github.com/spf13/afero"
)

var (
	Fs    = &afero.Afero{Fs: afero.NewOsFs()}
	Sleep = func(duration time.Duration) { time.Sleep(duration) } // 睡眠等待函数
)
