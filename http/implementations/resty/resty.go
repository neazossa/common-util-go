package resty

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	httpRest "github.com/neazossa/common-util-go/http/http"
	"github.com/neazossa/common-util-go/logger/logger"
	"github.com/neazossa/common-util-go/monitor/monitor"
)

type (
	Client struct {
		logger       logger.Logger
		client       *resty.Client
		isMonitor    bool
		monitor      monitor.Monitor
		context      context.Context
		requestId    string
		captureError bool
	}

	RequestLog struct {
		URL          string      `json:"url,omitempty"`
		Method       string      `json:"method,omitempty"`
		Token        string      `json:"token,omitempty"`
		AuthScheme   string      `json:"auth_scheme,omitempty"`
		QueryParam   url.Values  `json:"query_param,omitempty"`
		FormData     url.Values  `json:"form_data,omitempty"`
		Header       http.Header `json:"header,omitempty"`
		Time         time.Time   `json:"time,omitempty"`
		Body         interface{} `json:"body,omitempty"`
		ResponseBody string      `json:"response_body,omitempty"`
	}
)

func NewHttpRestyClient(ctx context.Context, logger logger.Logger) httpRest.Restful {
	client := resty.New()

	return &Client{
		logger: logger,
		client: client.
			SetLogger(logger).
			OnBeforeRequest(preRequest(ctx, logger)).
			OnAfterResponse(postResponse(ctx, logger)).
			OnError(onError(ctx, logger)),
	}
}

func preRequest(ctx context.Context, logger logger.Logger) resty.RequestMiddleware {
	return func(restyClient *resty.Client, r *resty.Request) error {
		var (
			requestID      = ""
			requestIDValue = ctx.Value("request_id")
		)
		if requestIDValue != nil {
			requestID = requestIDValue.(string)
		}

		var requestLog = RequestLog{
			URL:        r.URL,
			Method:     r.Method,
			Token:      r.Token,
			AuthScheme: r.AuthScheme,
			QueryParam: r.QueryParam,
			FormData:   r.FormData,
			Header:     r.Header,
			Time:       r.Time,
			Body:       r.Body,
		}

		logger.WithFields(map[string]interface{}{
			"request": requestLog,
		}).Info("preRequest Resty", requestID)
		return nil
	}
}

func postResponse(ctx context.Context, logger logger.Logger) resty.ResponseMiddleware {
	return func(restyClient *resty.Client, r *resty.Response) error {
		var (
			requestID      = ""
			requestIDValue = ctx.Value("request_id")
		)
		request := r.Request
		var requestLog = RequestLog{
			URL:          request.URL,
			Method:       request.Method,
			Token:        request.Token,
			AuthScheme:   request.AuthScheme,
			QueryParam:   request.QueryParam,
			FormData:     request.FormData,
			Header:       request.Header,
			Time:         request.Time,
			Body:         request.Body,
			ResponseBody: string(r.Body()),
		}
		if requestIDValue != nil {
			requestID = requestIDValue.(string)
		}
		logger.WithFields(map[string]interface{}{
			"request": requestLog,
		}).Info("postResponse Resty", requestID)
		return nil
	}
}

func onError(ctx context.Context, logger logger.Logger) resty.ErrorHook {
	return func(request *resty.Request, err error) {
		var (
			requestID      = ""
			requestIDValue = ctx.Value("request_id")
		)
		var requestLog = RequestLog{
			URL:          request.URL,
			Method:       request.Method,
			Token:        request.Token,
			AuthScheme:   request.AuthScheme,
			QueryParam:   request.QueryParam,
			FormData:     request.FormData,
			Header:       request.Header,
			Time:         request.Time,
			Body:         request.Body,
			ResponseBody: err.Error(),
		}
		if requestIDValue != nil {
			requestID = requestIDValue.(string)
		}
		logger.WithFields(map[string]interface{}{
			"request": requestLog,
		}).Error("onError Resty", requestID)
	}
}

func (c *Client) GET(path string, header map[string]string, param *map[string]interface{}) (*httpRest.BaseResponse, error) {
	tr := c.startMonitor("GET", path)

	if param != nil && len(*param) > 0 {
		template := "%s=%s"
		start := true
		for key, value := range *param {
			if start {
				path += "?"
				start = false
			} else {
				path += "&"
			}
			path += fmt.Sprintf(template, key, value)
		}
	}

	request := c.client.R().SetHeaders(header)
	response, err := request.Get(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) POST(path string, header map[string]string, body interface{}) (*httpRest.BaseResponse, error) {
	tr := c.startMonitor("POST", path)

	request := c.client.R().SetHeaders(header).SetBody(body)
	response, err := request.Post(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) POSTForm(path string, header map[string]string, body map[string]string) (*httpRest.BaseResponse, error) {
	tr := c.startMonitor("POST", path)

	request := c.client.R().SetHeaders(header).SetFormData(body)
	response, err := request.Post(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) PUT(path string, header map[string]string, body interface{}) (*httpRest.BaseResponse, error) {
	tr := c.startMonitor("PUT", path)

	request := c.client.R().SetHeaders(header).SetBody(body)
	response, err := request.Put(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) PATCH(path string, header map[string]string, body interface{}) (*httpRest.BaseResponse, error) {
	tr := c.startMonitor("PATCH", path)

	request := c.client.R().SetHeaders(header).SetBody(body)
	response, err := request.Patch(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) DELETE(path string, header map[string]string, body interface{}) (*httpRest.BaseResponse, error) {
	tr := c.startMonitor("DELETE", path)
	request := c.client.R().SetHeaders(header).SetBody(body)

	response, err := request.Delete(path)
	defer c.finishMonitor(tr, response, err)
	return convertToBaseResponse(response), err
}

func (c *Client) SetBasicAuth(username, password string) httpRest.Restful {
	c.client.SetBasicAuth(username, password)

	return &Client{
		logger:       c.logger,
		client:       c.client,
		isMonitor:    true,
		monitor:      c.monitor,
		context:      c.context,
		requestId:    c.requestId,
		captureError: c.captureError,
	}
}

func (c *Client) Monitor(ctx context.Context, mntr monitor.Monitor, requestId string, captureError bool) httpRest.Restful {
	return &Client{
		logger:       c.logger,
		client:       c.client,
		isMonitor:    true,
		monitor:      mntr,
		context:      ctx,
		requestId:    requestId,
		captureError: captureError,
	}
}

func (c *Client) startMonitor(method, url string) monitor.Transaction {
	if c.isMonitor {
		name := fmt.Sprintf("%s %s", method, url)
		return c.monitor.NewTransactionFromContext(c.context, monitor.Tick{
			Operation:       "http",
			TransactionName: name,
			Tags: []monitor.Tag{
				{"requestId", c.requestId},
				{"action", name},
				{"method", method},
			},
		})
	}
	return nil
}

func (c *Client) finishMonitor(transaction monitor.Transaction, response *resty.Response, err error) {
	var (
		statusCode = response.StatusCode()
		status     = response.Status()
	)
	if c.isMonitor {
		if statusCode == 0 {
			statusCode = 500
			status = "InternalError"
		}

		transaction.FinishWithTags([]monitor.Tag{
			{"code", fmt.Sprintf("%d", statusCode)},
			{"status", status},
		})
		if c.captureError {
			c.monitor.Capture(err)
		}
	}
}

func convertToBaseResponse(response *resty.Response) *httpRest.BaseResponse {
	if response == nil {
		return nil
	}

	return &httpRest.BaseResponse{
		Status: response.StatusCode(),
		Header: response.Header(),
		Body:   response.Body(),
	}
}
