package shared

import (
	"encoding/json"
)

type (
	commonStructWithRequestId struct {
		RequestId string `json:"requestId"`
	}
)

func GetRequestId(req interface{}) (string, error) {
	data := commonStructWithRequestId{}
	mar, _ := json.Marshal(req)
	err := json.Unmarshal(mar, &data)
	if err != nil {
		return "", err
	}
	return data.RequestId, nil
}

