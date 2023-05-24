package http

import (
	"context"
	"net/http"

	"github.com/neazossa/common-util-go/monitor/monitor"
)

type (
	Restful interface {
		GET(path string, header map[string]string, param *map[string]interface{}) (*BaseResponse, error)
		POST(path string, header map[string]string, body interface{}) (*BaseResponse, error)
		PUT(path string, header map[string]string, body interface{}) (*BaseResponse, error)
		PATCH(path string, header map[string]string, body interface{}) (*BaseResponse, error)
		DELETE(path string, header map[string]string, body interface{}) (*BaseResponse, error)

		POSTForm(path string, header map[string]string, body map[string]string) (*BaseResponse, error)

		SetBasicAuth(username, password string) Restful

		Monitor(ctx context.Context, monitor monitor.Monitor, requestId string, captureError bool) Restful
	}

	BaseResponse struct {
		Status int
		Header http.Header
		Body   []byte
	}
)
