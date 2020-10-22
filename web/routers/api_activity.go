package routers

import (
	"net/http"
	"seckill/web/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func activityList(c *gin.Context) { // GET => get all activity
	resp = map[string]interface{}{
		"code": ErrSuccess,
		"msg":  "success!",
	}
	ret, err := activity.GetActivityList()
	if err != nil {
		resp["code"] = ErrNotFound
		resp["msg"] = "not found activity!"
		c.JSON(http.StatusNotFound, resp)
		return
	}
	resp["datas"] = ret
	c.JSON(http.StatusOK, resp)
}

func activityDetail(c *gin.Context) {
	resp = map[string]interface{}{
		"code": ErrSuccess,
		"msg":  "success!",
	}
	aId := c.Param("aid")
	if _, err := strconv.Atoi(aId); err != nil {
		resp["code"] = ErrParamsInvalid
		resp["msg"] = "params validation failed!"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	ret, err := activity.ActivityWithId(aId)
	if err != nil {
		resp["code"] = ErrNotFound
		resp["msg"] = "not found activity!"
		c.JSON(http.StatusNotFound, resp)
		return
	}
	c.JSON(http.StatusOK, ret)
}

func createActivity(c *gin.Context) {
	resp = map[string]interface{}{
		"code": ErrSuccess,
		"msg":  "success!",
	}
	prod := models.NewActivity()
	if err := c.ShouldBind(prod); err != nil {
		logs.Errorf("params validation failed! err:%v", err)
		resp["code"] = ErrParamsInvalid
		resp["msg"] = "params validation failed!"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	err := activity.AddActivity(prod)
	if err != nil {
		resp["code"] = ErrDBCreateFailed
		resp["msg"] = DBOpertorErrInfo
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func updateActivity(c *gin.Context) {
	resp = map[string]interface{}{
		"code": ErrSuccess,
		"msg":  "success!",
	}
	aId := c.Param("aid")
	if _, err := strconv.Atoi(aId); err != nil {
		resp["code"] = ErrParamsInvalid
		resp["msg"] = "params validation failed!"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	producUpdate := models.NewActivityUpdateValid()
	// 获取form data, 验证form data
	if err := c.ShouldBind(producUpdate); err != nil {
		logs.Errorf("params validation failed! err:%v", err)
		resp["code"] = ErrParamsInvalid
		resp["msg"] = "params validation failed!"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	// 获取待更新字段
	params, err := getFormDatas(*producUpdate, c)
	if err != nil {
		logs.Errorf("getFormDatas func params invalid! err:%v", err)
		resp["code"] = http.StatusInternalServerError
		resp["msg"] = "interface service error!"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	// 没有修改, 或者请求中的字段不对应!
	if len(params) == 0 { // 无修改
		logs.Warn("no request params or params validation failed!")
		resp["code"] = ErrParamsInvalid
		resp["msg"] = "params validation failed!"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	// 根据id修改数据库记录属性
	err = activity.UpdateActivity(aId, params)
	if err != nil {
		resp["code"] = ErrNotFound
		resp["msg"] = "could not found activity by id!"
		c.JSON(http.StatusNotFound, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func deleteActivity(c *gin.Context) {
	resp = map[string]interface{}{
		"code": ErrSuccess,
		"msg":  "success!",
	}
	aId := c.Param("aid")
	if _, err := strconv.Atoi(aId); err != nil {
		resp["code"] = ErrParamsInvalid
		resp["msg"] = "params validation failed!"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	// 物理删除某记录
	err := activity.DeleteActivity(aId)
	if err != nil {
		resp["code"] = ErrNotFound
		resp["msg"] = "could not delete activity by id!"
		c.JSON(http.StatusNotFound, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}
