package service

import (
	"encoding/json"

	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/errs"
)

// TransToArray2 序列化已经展平好的配置数据。
func TransToArray2(data interface{}) ([]byte, error) {
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, errs.New("", "", "", 0, "JSON序列化", "MarshalIndent失败: %v", err)
	}
	return buf, nil
}
