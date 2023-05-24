package uploader

import (
	"context"

	"github.com/neazossa/common-util-go/monitor/monitor"
)

type (
	Uploader interface {
		SetPutOption(opt interface{}) Uploader
		Put(file []byte, fileName string) error

		SetGetOption(opt interface{}) Uploader
		Get(fileName string) ([]byte, error)

		SetRemoveOption(opt interface{}) Uploader
		Remove(fileName string) error

		Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) Uploader
	}
)
