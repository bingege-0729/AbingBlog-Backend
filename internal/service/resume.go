package service

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"abingblog-backend/internal/errcode"
	"abingblog-backend/internal/model"
	"abingblog-backend/internal/repository"
)

// 简历只保留一份：上传即覆盖，文件名统一固定，
// 这样前台下载链接永远不变，不需要在数据库里存一堆历史文件。
const (
	resumeMaxSize  = 10 << 20 // 10MB，简历 PDF 远超不了这个量级
	resumeFileName = "resume.pdf"
)

type ResumeService struct {
	repo *repository.ResumeRepo
	dir  string
}

func NewResumeService(repo *repository.ResumeRepo, dir string) *ResumeService {
	return &ResumeService{repo: repo, dir: dir}
}

// ResumeMeta 对外暴露的简历信息（前台据此决定要不要显示下载按钮）
type ResumeMeta struct {
	Available bool      `json:"available"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *ResumeService) path() string {
	return filepath.Join(s.dir, resumeFileName)
}

// Meta 读取当前简历信息。没上传过返回 Available=false，而不是报错——
// 前台只是不显示下载按钮。
func (s *ResumeService) Meta() (*ResumeMeta, error) {
	record, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &ResumeMeta{Available: false}, nil
	}
	// 元信息有记录但文件被删了（比如手动清理了 volume）：以文件为准，避免给出死链
	if _, err := os.Stat(s.path()); err != nil {
		return &ResumeMeta{Available: false, Filename: record.Filename}, nil
	}
	return &ResumeMeta{
		Available: true,
		Filename:  record.Filename,
		Size:      record.Size,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

// Save 校验并落盘新的简历 PDF，然后更新元信息。
func (s *ResumeService) Save(rawName string, data []byte) (*ResumeMeta, error) {
	if len(data) == 0 {
		return nil, errcode.New(errcode.CodeResumeEmpty, "文件为空")
	}
	if len(data) > resumeMaxSize {
		return nil, errcode.New(errcode.CodeResumeTooLarge, "简历不能超过 10MB")
	}
	// 只认 PDF 文件头：改后缀名不算，避免传上来一个 HTML 被当附件下载
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, errcode.New(errcode.CodeResumeInvalid, "只支持 PDF 文件")
	}
	name := strings.TrimSpace(rawName)
	if name == "" {
		name = "resume.pdf"
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	// 先写临时文件再 rename：中途失败不会把已有简历弄坏（rename 在同盘上是原子的）
	tmp := s.path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return nil, fmt.Errorf("写入简历文件失败: %w", err)
	}
	if err := os.Rename(tmp, s.path()); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("替换简历文件失败: %w", err)
	}

	record := &model.ResumeFile{Filename: name, Size: int64(len(data))}
	if err := s.repo.Save(record); err != nil {
		return nil, err
	}
	return s.Meta()
}

func (s *ResumeService) Delete() error {
	if err := os.Remove(s.path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除简历文件失败: %w", err)
	}
	return s.repo.Delete()
}

// Path 返回简历文件的绝对/相对路径，供 handler 用 http.ServeContent 输出。
// 第二个返回值表示当前是否真的有一份可下载的文件。
func (s *ResumeService) Path() (string, bool) {
	info, err := os.Stat(s.path())
	if err != nil || info.IsDir() {
		return "", false
	}
	return s.path(), true
}

// Filename 下载时给浏览器看的文件名
func (s *ResumeService) Filename() string {
	record, err := s.repo.Get()
	if err != nil || record == nil || record.Filename == "" {
		return resumeFileName
	}
	return record.Filename
}

// UpdatedAt 用于下载响应里的 Last-Modified，方便浏览器判断要不要重新拉
func (s *ResumeService) UpdatedAt() (time.Time, bool) {
	info, err := os.Stat(s.path())
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}
