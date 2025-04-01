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
