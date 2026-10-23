package errcode

import "fmt"

// 业务错误码：与 docs/api-v1.md 的「业务错误码」表保持一致
const (
	CodeOK               = 0    // 成功
	CodeBadParam         = 1000 // 请求参数错误
	CodeLoginFail        = 1001 // 用户名或密码错误
	CodeArticleNotFound  = 1002 // 文章不存在
	CodeCategoryNotEmpty = 1003 // 分类下存在文章，不可删除
	CodeCategoryNotFound = 1004 // 分类不存在
	CodeTagNotFound      = 1005 // 标签不存在
	CodeTitleEmpty       = 1006 // 文章标题不能为空
	CodeBadStatus        = 1007 // 文章状态非法（只能 draft/published）
	CodeNameEmpty        = 1008 // 名称不能为空
	CodeNameExists       = 1009 // 名称已存在
	CodeProjectNotFound  = 1010 // 项目不存在
	CodeSlugEmpty        = 1011 // 项目 slug 不能为空
	CodeSlugExists       = 1012 // 项目 slug 已存在
	CodeInternal         = 5000 // 服务器内部错误

	// 友链 51xx
	CodeFriendNameEmpty       = 5100 // 友链名称不能为空
	CodeFriendURLEmpty        = 5101 // 友链地址不能为空
	CodeFriendNotFound        = 5102 // 友链不存在
	CodeFriendNotOwner        = 5103 // 只能操作自己提交的友链
	CodeFriendVisitorRequired = 5104 // 缺少访客标识（X-Visitor-ID）
	CodeFriendBadStatus       = 5105 // 友链状态非法（只能 0/1/2）

	// 个人档案 52xx
	CodeProfileHeadlineEmpty = 5200 // 定位不能为空
	CodeProfileTooLong       = 5201 // 字段超长（定位/所在地/最近在做/简介/列表项）

	// 简历文件 53xx
	CodeResumeEmpty    = 5300 // 上传的文件为空
	CodeResumeTooLarge = 5301 // 文件超过大小上限
	CodeResumeInvalid  = 5302 // 不是 PDF 文件
)

// BizError 业务错误：service 层抛出，handler 层统一映射为 HTTP 响应
type BizError struct {
	Code int
	Msg  string
}

func (e *BizError) Error() string { return fmt.Sprintf("[业务错误 %d] %s", e.Code, e.Msg) }

// New 创建一个业务错误
func New(code int, msg string) *BizError { return &BizError{Code: code, Msg: msg} }
