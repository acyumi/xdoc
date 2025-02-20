package feishu

import (
	"reflect"
	"time"

	"github.com/cenkalti/backoff/v5"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
)

var (
	maxRetryCount = 5
	codeErrorType = reflect.TypeOf(larkcore.CodeError{})
)

// SendWithRetry 飞书上层重试客户端
// 本来想做 RetryHttpClient 的，实现 larkcore.HttpClient
// 但是要自己解析 http.Response 的内容反序列化得到内容再判断是否需要重试
// 想了想还是算了，换成上层重试
func SendWithRetry[R any](operation func(count int) (R, error)) (resp R, err error) {
	ebo := backoff.NewExponentialBackOff()
	ebo.InitialInterval = time.Second
	ebo.Multiplier = 2.0
	ebo.MaxInterval = 5 * time.Second
	ebo.RandomizationFactor = 0.2
	ticker := backoff.NewTicker(ebo)
	defer ticker.Stop()

	// Ticks will continue to arrive when the previous operation is still running,
	// so operations that take a while to fail could run in quick succession.
	var count int
	for range ticker.C {
		count++
		resp, err = operation(count)
		// 飞书SDK从代码层面报错，那就是有问题了，不需要重试，如果是合适的响应错误，那就重试
		if err != nil {
			break
		}
		val := reflect.Indirect(reflect.ValueOf(resp))
		if val.Kind() != reflect.Struct {
			break
		}
		ceField := val.FieldByName("CodeError")
		if !ceField.IsValid() || ceField.Type() != codeErrorType {
			break
		}
		codeError := ceField.Interface().(larkcore.CodeError)
		switch codeError.Code {
		// https://open.feishu.cn/document/server-docs/api-call-guide/frequency-control
		// https://open.feishu.cn/document/server-docs/api-call-guide/generic-error-code
		case 99991400, 11232:
			if count >= maxRetryCount {
				return
			}
			continue
		default:
			return
		}
	}
	return resp, err
}
