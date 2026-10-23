package response

import (
	"errors"
	"log"
	"net/http"

	"abingblog-backend/internal/errcode"

	"github.com/gin-gonic/gin"
)

// OK 统一成功响应：{code:0, message:"success", data:...}
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

// Fail 统一失败响应：HTTP 状态码体现语义（400/404/500），code 为业务错误码
func Fail(c *gin.Context, httpStatus, bizCode int, message string) {
	c.JSON(httpStatus, gin.H{"code": bizCode, "message": message, "data": nil})
}

// Error 处理 handler 收到的错误：
//   - 业务错误（*errcode.BizError）→ 映射为对应 HTTP 状态码返回给前端
//   - 未知错误 → 只记日志，返回 500，不把内部细节暴露给前端
func Error(c *gin.Context, err error) {
	if be, ok := errors.AsType[*errcode.BizError](err); ok {
		status := http.StatusBadRequest
		switch be.Code {
		case errcode.CodeArticleNotFound, errcode.CodeCategoryNotFound, errcode.CodeTagNotFound:
			status = http.StatusNotFound
		case errcode.CodeLoginFail:
			status = http.StatusUnauthorized
		case errcode.CodeResumeTooLarge:
			status = http.StatusRequestEntityTooLarge
		}
		Fail(c, status, be.Code, be.Msg)
		return
	}
	log.Printf("内部错误: %v", err)
	Fail(c, http.StatusInternalServerError, errcode.CodeInternal, "服务器内部错误")
}
