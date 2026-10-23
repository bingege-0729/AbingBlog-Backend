package repository

import (
	"errors"

	"abingblog-backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfileRepo struct{ db *gorm.DB }

func NewProfileRepo(db *gorm.DB) *ProfileRepo { return &ProfileRepo{db: db} }

// Get 读取单行档案。行还不存在（首次部署、从没保存过）时返回空档案而不是报错，
// 由调用方决定兜底内容——前台会用自己的默认值渲染，不该因为「没填过」而 500。
func (r *ProfileRepo) Get() (*model.Profile, error) {
	var profile model.Profile
	err := r.db.First(&profile, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.Profile{ID: 1, Stacks: []string{}, Awards: []string{}, Interests: []string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Save 全量覆盖单行档案：主键固定 1，用 OnConflict 做 upsert，
// 这样「先保存」和「后保存」走同一条 SQL，不会出现主键冲突。
func (r *ProfileRepo) Save(profile *model.Profile) error {
	profile.ID = 1
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(profile).Error
}
