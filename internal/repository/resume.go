package repository

import (
	"errors"

	"abingblog-backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResumeRepo struct{ db *gorm.DB }

func NewResumeRepo(db *gorm.DB) *ResumeRepo { return &ResumeRepo{db: db} }

// Get 读取简历元信息。没有上传过时返回 nil，由 service 决定怎么表达「没有」。
func (r *ResumeRepo) Get() (*model.ResumeFile, error) {
	var resume model.ResumeFile
	err := r.db.First(&resume, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &resume, nil
}

// Save 覆盖单行元信息（主键固定 1），用 upsert 避免「第一次存」和「之后替换」走两条路径。
func (r *ResumeRepo) Save(resume *model.ResumeFile) error {
	resume.ID = 1
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(resume).Error
}

func (r *ResumeRepo) Delete() error {
	return r.db.Delete(&model.ResumeFile{}, 1).Error
}
