package handler

import (
	"net/http"

	"abingblog-backend/internal/errcode"
	"abingblog-backend/internal/model"
	"abingblog-backend/internal/response"
	"abingblog-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ProfileHandler 个人档案接口层：前台只读、后台读写同一份数据
type ProfileHandler struct{ svc *service.ProfileService }

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

// Get 公开读取：门户右侧「档案 / 技术栈 / 近况」三张卡的数据源，无需登录
func (h *ProfileHandler) Get(c *gin.Context) {
	profile, err := h.svc.Get()
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toProfileDTO(profile))
}

// AdminGet 后台读取：数据与公开接口相同，单独一条路由是为了让权限边界清晰
func (h *ProfileHandler) AdminGet(c *gin.Context) { h.Get(c) }

// Update 后台全量保存：字段允许留空（前台按空值隐藏对应行），但「定位」必填
func (h *ProfileHandler) Update(c *gin.Context) {
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.CodeBadParam, "请求参数格式错误")
		return
	}
	profile, err := h.svc.Update(service.ProfileInput{
		Headline:    req.Headline,
		Location:    req.Location,
		Summary:     req.Summary,
		Stacks:      req.Stacks,
		Awards:      req.Awards,
		Doing:       req.Doing,
		Interests:   req.Interests,
		Experiences: req.Experiences,
		Theme:       req.Theme,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toProfileDTO(profile))
}

// profileRequest 后台提交体：字段名与前端 Profile 类型一一对应
type profileRequest struct {
	Headline    string   `json:"headline"`
	Location    string   `json:"location"`
	Summary     string   `json:"summary"`
	Stacks      []string `json:"stacks"`
	Awards      []string `json:"awards"`
	Doing       string   `json:"doing"`
	Interests   []string `json:"interests"`
	Experiences []string `json:"experiences"`
	Theme       string   `json:"theme"` // 'sand' | 'ink'，非法值由 service 回落 sand
}

// profileDTO 对外返回体：只暴露前台要渲染的字段，不含 ID / 更新时间
type profileDTO struct {
	Headline    string   `json:"headline"`
	Location    string   `json:"location"`
	Summary     string   `json:"summary"`
	Stacks      []string `json:"stacks"`
	Awards      []string `json:"awards"`
	Doing       string   `json:"doing"`
	Interests   []string `json:"interests"`
	Experiences []string `json:"experiences"`
	Theme       string   `json:"theme"`
}

// toProfileDTO 转换并保证列表字段是 [] 而不是 null，
// 前端拿到 null 会直接放弃兜底，导致界面少一行。
func toProfileDTO(p *model.Profile) profileDTO {
	theme := p.Theme
	if theme == "" {
		theme = service.ThemeSand // 老数据迁移前的空值，对外统一成默认主题
	}
	return profileDTO{
		Headline:    p.Headline,
		Location:    p.Location,
		Summary:     p.Summary,
		Stacks:      emptyIfNil(p.Stacks),
		Awards:      emptyIfNil(p.Awards),
		Doing:       p.Doing,
		Interests:   emptyIfNil(p.Interests),
		Experiences: emptyIfNil(p.Experiences),
		Theme:       theme,
	}
}

func emptyIfNil(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}
