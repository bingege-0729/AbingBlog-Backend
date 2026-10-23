package service

import (
	"strings"

	"abingblog-backend/internal/errcode"
	"abingblog-backend/internal/model"
	"abingblog-backend/internal/repository"
)

// ProfileService 个人档案：单行数据的读写 + 入库前的清洗校验。
type ProfileService struct{ repo *repository.ProfileRepo }

func NewProfileService(repo *repository.ProfileRepo) *ProfileService {
	return &ProfileService{repo: repo}
}

// ProfileInput 后台提交上来的档案（已由 handler 做过 JSON 绑定）
type ProfileInput struct {
	Headline    string
	Location    string
	Summary     string
	Stacks      []string
	Awards      []string
	Doing       string
	Interests   []string
	Experiences []string
	Theme       string
}

// 站点可选配色。前端 src/styles/tokens.css 里定义了同名主题，
// 后端只做白名单校验，不知道每套主题长什么样。
const (
	ThemeSand = "sand"
	ThemeInk  = "ink"
)

// normalizeTheme 把配色收敛到白名单：传空、传脏值一律回落 sand。
// 这里刻意不报错——配色是展示偏好，不该因为一个小字段把整份档案的保存打断。
func normalizeTheme(value string) string {
	if strings.TrimSpace(value) == ThemeInk {
		return ThemeInk
	}
	return ThemeSand
}

// 字段长度上限，与前端输入框的 maxlength 对齐；超长直接拒绝，
// 避免把整段简历塞进「一句话定位」把前台排版撑坏。
const (
	profileHeadlineMax = 60
	profileLocationMax = 40
	profileDoingMax    = 60
	profileSummaryMax  = 500
	profileItemMax     = 60  // 单个技术栈 / 奖项 / 兴趣的长度
	profileExpMax      = 160 // 单条经历：「时间段 | 标题 | 说明」拼起来比普通列表项长
	profileListMax     = 20  // 单个列表最多几条
)

func (s *ProfileService) Get() (*model.Profile, error) {
	return s.repo.Get()
}

// Update 全量覆盖档案。定位是右侧档案卡的主标题，必填；
// 其余字段允许留空，前台会按空值隐藏对应的行。
func (s *ProfileService) Update(in ProfileInput) (*model.Profile, error) {
	headline := strings.TrimSpace(in.Headline)
	if headline == "" {
		return nil, errcode.New(errcode.CodeProfileHeadlineEmpty, "定位不能为空")
	}
	if len([]rune(headline)) > profileHeadlineMax {
		return nil, errcode.New(errcode.CodeProfileTooLong, "定位不能超过 60 个字")
	}

	location := strings.TrimSpace(in.Location)
	if len([]rune(location)) > profileLocationMax {
		return nil, errcode.New(errcode.CodeProfileTooLong, "所在地不能超过 40 个字")
	}

	doing := strings.TrimSpace(in.Doing)
	if len([]rune(doing)) > profileDoingMax {
		return nil, errcode.New(errcode.CodeProfileTooLong, "最近在做不能超过 60 个字")
	}

	summary := strings.TrimSpace(in.Summary)
	if len([]rune(summary)) > profileSummaryMax {
		return nil, errcode.New(errcode.CodeProfileTooLong, "简介不能超过 500 个字")
	}

	stacks, err := cleanList("技术栈", in.Stacks)
	if err != nil {
		return nil, err
	}
	awards, err := cleanList("奖项", in.Awards)
	if err != nil {
		return nil, err
	}
	interests, err := cleanList("兴趣", in.Interests)
	if err != nil {
		return nil, err
	}
	experiences, err := cleanListWithMax("经历", in.Experiences, profileExpMax)
	if err != nil {
		return nil, err
	}

	profile := &model.Profile{
		Headline:    headline,
		Location:    location,
		Summary:     summary,
		Stacks:      stacks,
		Awards:      awards,
		Doing:       doing,
		Interests:   interests,
		Experiences: experiences,
		Theme:       normalizeTheme(in.Theme),
	}
	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return s.repo.Get()
}

// cleanList 清洗字符串列表：逐项 trim、丢掉空项、限制条数与单项长度。
// 返回非 nil 切片，保证 JSON 序列化出 [] 而不是 null。
func cleanList(field string, items []string) ([]string, error) {
	return cleanListWithMax(field, items, profileItemMax)
}

// cleanListWithMax 同 cleanList，但可以指定单项长度上限（经历条目比普通列表项长）。
func cleanListWithMax(field string, items []string, maxItem int) ([]string, error) {
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if len([]rune(value)) > maxItem {
			return nil, errcode.New(errcode.CodeProfileTooLong, field+"单项过长")
		}
		out = append(out, value)
		if len(out) > profileListMax {
			return nil, errcode.New(errcode.CodeProfileTooLong, field+"最多 20 条")
		}
	}
	return out, nil
}

// EnsureSeed 首次启动时写入一份初始档案，避免后台打开就是空表单。
// 只在表里没有行时插入，已有数据永不覆盖。
func (s *ProfileService) EnsureSeed() error {
	current, err := s.repo.Get()
	if err != nil {
		return err
	}
	if current.Headline != "" {
		return nil
	}
	_, err = s.Update(ProfileInput{
		Headline:  "后端开发 · AI 应用方向",
		Location:  "广州",
		Summary:   "专注可靠的后端系统与克制的界面。这里放我写过的文章、做过的项目，以及最近在折腾的东西。",
		Stacks:    []string{"Go", "Java", "Vue", "MySQL", "Redis", "Docker"},
		Doing:     "",
		Interests: []string{"音乐"},
		Theme:     ThemeSand,
		Experiences: []string{
			"2024.09 – 2028.06 | 广东金融学院 · 计算机科学与技术 | 本科在读",
			"2026.06 – 2026.09 | 后端开发实习生 · 广州 | Golang / Kratos，支付链路与飞书数据导入",
		},
	})
	return err
}
