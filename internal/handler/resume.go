package handler

import (
	"io"
	"net/http"
	"net/url"

	"abingblog-backend/internal/errcode"
	"abingblog-backend/internal/response"
	"abingblog-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// 上传请求体上限：文件上限 10MB，再留 1MB 给 multipart 边界和表单字段，
// 超出的请求在读取阶段就被 MaxBytesReader 掐断，不会先把整个 body 读进内存再判大小。
const resumeBodyLimit = 11 << 20

type ResumeHandler struct{ svc *service.ResumeService }

func NewResumeHandler(svc *service.ResumeService) *ResumeHandler {
	return &ResumeHandler{svc: svc}
}

// Meta 公开接口：前台据此决定要不要显示「下载简历」按钮
func (h *ResumeHandler) Meta(c *gin.Context) {
	meta, err := h.svc.Meta()
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, meta)
}

// File 公开下载：直接输出 PDF 本体，支持 Range（浏览器可以边下边看）
func (h *ResumeHandler) File(c *gin.Context) {
	path, ok := h.svc.Path()
	if !ok {
		// 没有文件时不返回 JSON：这个地址是给 <a download> 用的，返回 JSON 用户看到一坨花括号
		c.String(http.StatusNotFound, "简历还没有上传")
		return
	}
	name := h.svc.Filename()
	c.Header("Content-Disposition", "attachment; filename=\"resume.pdf\"; filename*=UTF-8''"+url.PathEscape(name))
	c.Header("Content-Type", "application/pdf")
	// 简历会被替换，不能让浏览器长期缓存旧的；用 ETag/Last-Modified 走协商缓存
	c.Header("Cache-Control", "no-cache")
	c.File(path)
}

// Upload 后台上传/替换简历（multipart，字段名 file）
func (h *ResumeHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, resumeBodyLimit)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.CodeBadParam, "请选择要上传的 PDF 文件")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.CodeBadParam, "读取上传文件失败")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, resumeBodyLimit))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.CodeBadParam, "读取上传文件失败")
		return
	}

	meta, err := h.svc.Save(fileHeader.Filename, data)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, meta)
}

// Delete 后台删除简历（前台随后不再显示下载按钮）
func (h *ResumeHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"available": false})
}
