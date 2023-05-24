package shared

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	MandatoryHeader struct {
		XToken string `json:"X-Token" form:"X-Token" header:"X-Token"`
	}

	CommonRedisOTPPayload struct {
		InputAttempt int        `json:"input_attempt,omitempty"`
		SentAttempt  int        `json:"sent_attempt,omitempty"`
		Expiry       time.Time  `json:"expiry"`
		Freeze       *time.Time `json:"freeze,omitempty"`
		Token        string     `json:"token"`
	}

	CommonRedisOTPPayloads []CommonRedisOTPPayload

	Now func() time.Time
)

var (
	TimeFunc Now = time.Now
)

func BindMandatoryHeader(c echo.Context) (MandatoryHeader, error) {
	var (
		mandatoryHeader = MandatoryHeader{}
	)

	if err := (&echo.DefaultBinder{}).BindHeaders(c, &mandatoryHeader); err != nil {
		return mandatoryHeader, err
	}
	return mandatoryHeader, nil
}

func (c CommonRedisOTPPayload) IsExpired() bool {
	return TimeFunc().After(c.Expiry)
}

func (c CommonRedisOTPPayload) IsFrozen() bool {
	return c.Freeze != nil && !TimeFunc().After(*c.Freeze)
}

func (c CommonRedisOTPPayload) CanResend(maxAttempt int) bool {
	return c.SentAttempt < maxAttempt
}

func (c CommonRedisOTPPayload) CanValidate(maxAttempt int) bool {
	return c.InputAttempt < maxAttempt
}

func (c CommonRedisOTPPayload) ValidateToken(token string) error {
	return bcrypt.CompareHashAndPassword([]byte(c.Token), []byte(token))
}

func (c *CommonRedisOTPPayload) SetToken(token string, cost int) error {
	password, err := bcrypt.GenerateFromPassword([]byte(token), cost)
	if err != nil {
		return err
	}
	c.Token = string(password)
	return nil
}

func (c CommonRedisOTPPayload) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *CommonRedisOTPPayload) UnmarshalBinary(data []byte) error {
	err := json.Unmarshal(data, &c)
	return err
}

func (c CommonRedisOTPPayloads) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *CommonRedisOTPPayloads) UnmarshalBinary(data []byte) error {
	err := json.Unmarshal(data, &c)
	return err
}

func HttpToGrpcError(httpStatus int, err error) error {
	if err == nil {
		return nil
	}

	switch httpStatus {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return status.New(codes.InvalidArgument, err.Error()).Err()
	case http.StatusUnauthorized:
		return status.New(codes.Unauthenticated, err.Error()).Err()
	case http.StatusForbidden:
		return status.New(codes.PermissionDenied, err.Error()).Err()
	case http.StatusNotFound:
		return status.New(codes.NotFound, err.Error()).Err()
	case http.StatusTooManyRequests:
		return status.New(codes.Unavailable, err.Error()).Err()
	case http.StatusBadGateway:
		return status.New(codes.Unavailable, err.Error()).Err()
	case http.StatusServiceUnavailable:
		return status.New(codes.Unavailable, err.Error()).Err()
	case http.StatusGatewayTimeout:
		return status.New(codes.Unavailable, err.Error()).Err()
	case http.StatusInternalServerError:
		return status.New(codes.Internal, err.Error()).Err()
	default:
		return status.New(codes.Unknown, err.Error()).Err()
	}
}

func GrpcToHttpStatus(err error) int {
	errStats, _ := status.FromError(err)

	switch errStats.Code() {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func GrpcToHttpStatusError(err error) (int, error) {
	errStats, _ := status.FromError(err)

	switch errStats.Code() {
	case codes.OK:
		return http.StatusOK, nil
	case codes.InvalidArgument:
		return http.StatusBadRequest, errors.New(errStats.Message())
	case codes.Unauthenticated:
		return http.StatusUnauthorized, errors.New(errStats.Message())
	case codes.PermissionDenied:
		return http.StatusForbidden, errors.New(errStats.Message())
	case codes.NotFound:
		return http.StatusNotFound, errors.New(errStats.Message())
	case codes.Unavailable:
		return http.StatusServiceUnavailable, errors.New(errStats.Message())
	default:
		return http.StatusInternalServerError, errors.New(errStats.Message())
	}
}

func FakeTime(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) Now {
	return func() time.Time {
		return time.Date(year, month, day, hour, min, sec, nsec, loc)
	}
}
