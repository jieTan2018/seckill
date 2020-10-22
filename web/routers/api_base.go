package routers

import (
	"errors"
	"reflect"

	"github.com/gin-gonic/gin"
)

var (
	resp map[string]interface{}
)

// 获取修改的字段和值
func getFormDatas(stru interface{}, c *gin.Context) (map[string]interface{}, error) {
	// // 组装结构体的字段、tag
	t := reflect.TypeOf(stru)
	if t.Kind() != reflect.Struct { // struct才有NumField()
		return nil, errors.New("check type error, not struct!")
	}

	struNums := t.NumField()
	params := make(map[string]interface{}, struNums)
	// 获取struct的keys、tags
	for i := 0; i < struNums; i++ {
		if tag, ok := t.Field(i).Tag.Lookup("form"); ok { // 仅解析有"form"的tag
			if val := c.PostForm(tag); val != "" { // 仅保存非""的form data
				params[t.Field(i).Name] = val
			}
		}
	}
	return params, nil
}
