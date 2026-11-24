package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// SyncService 承载移动端一次性批量上传同步（PRD 6.9 / US-SYNC-01~03）。
type SyncService struct {
	DB *gorm.DB
}

func NewSyncService(db *gorm.DB) *SyncService {
	return &SyncService{DB: db}
}

type SyncPlanItem struct {
	ClientID           string     `json:"client_id" binding:"required"` // 客户端生成的稳定标识（幂等键）
	Title              string     `json:"title" binding:"required"`
	Note               string     `json:"note"`
	PlanGroupID        *uint      `json:"plan_group_id"`
	StartTime          time.Time  `json:"start_time" binding:"required"`
	EndTime            time.Time  `json:"end_time" binding:"required"`
	LastSyncedAt       *time.Time `json:"last_synced_at"`      // 客户端上次同步时间（冲突检测用）
	ConflictResolution string     `json:"conflict_resolution"` // keep_local | keep_cloud | save_as_new（默认 save_as_new）
}

type SyncUploadInput struct {
	Plans []SyncPlanItem `json:"plans" binding:"required"`
}

type SyncPlanResult struct {
	ClientID    string     `json:"client_id"`
	NewClientID string     `json:"new_client_id,omitempty"` // save_as_new 时返回新的 client_id
	PlanID      uint       `json:"plan_id,omitempty"`
	Status      string     `json:"status"` // created | updated | kept_cloud | saved_new
	Conflict    bool       `json:"conflict"`
	SyncedAt    *time.Time `json:"synced_at,omitempty"` // 供客户端更新其"上次同步时间"
}

// Upload 批量上传个人计划。幂等：同一 client_id 重复上传不产生副本（US-SYNC-01）。
func (s *SyncService) Upload(actorID uint, in SyncUploadInput) ([]SyncPlanResult, error) {
	var results []SyncPlanResult
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range in.Plans {
			res, err := s.upsertOne(tx, actorID, item)
			if err != nil {
				return err
			}
			results = append(results, res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// RecordBind 记录移动端绑定后端域名的事件（US-OFFLINE-02：绑定写入审计日志）。
func (s *SyncService) RecordBind(actorID uint, info string) error {
	return LogAudit(s.DB, actorID, "sync.bind", "user", actorID, "", info)
}

// upsertOne 处理单条上传：首次创建 / 无冲突更新 / 冲突按策略处理。
func (s *SyncService) upsertOne(tx *gorm.DB, actorID uint, item SyncPlanItem) (SyncPlanResult, error) {
	if err := validatePlanTimes(item.StartTime, item.EndTime); err != nil {
		return SyncPlanResult{}, err
	}
	if item.PlanGroupID != nil {
		if err := validatePlanGroup(tx, actorID, item.PlanGroupID); err != nil {
			return SyncPlanResult{}, err
		}
	}

	var existing model.Plan
	err := tx.Where("owner_id = ? AND client_id = ?", actorID, item.ClientID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 首次上传：创建
		clientID := item.ClientID
		p := model.Plan{
			Source:      model.PlanSourcePersonal,
			OwnerID:     actorID,
			ClientID:    &clientID,
			PlanGroupID: item.PlanGroupID,
			Title:       item.Title, Note: item.Note,
			StartTime: normalizePlanTime(item.StartTime), EndTime: normalizePlanTime(item.EndTime),
			Status: model.PlanNotStarted, CreatedBy: actorID,
		}
		if err := tx.Create(&p).Error; err != nil {
			return SyncPlanResult{}, err
		}
		return SyncPlanResult{ClientID: item.ClientID, PlanID: p.ID, Status: "created", SyncedAt: &p.UpdatedAt}, nil
	}
	if err != nil {
		return SyncPlanResult{}, err
	}

	// 已存在：判断冲突——云端版本比客户端上次同步时间新
	conflict := item.LastSyncedAt != nil && existing.UpdatedAt.After(*item.LastSyncedAt)

	if !conflict {
		// 无冲突：正常更新
		applyItem(&existing, item)
		if err := tx.Save(&existing).Error; err != nil {
			return SyncPlanResult{}, err
		}
		return SyncPlanResult{ClientID: item.ClientID, PlanID: existing.ID, Status: "updated", SyncedAt: &existing.UpdatedAt}, nil
	}

	// 冲突：按策略处理（默认另存为新计划，避免丢数据）
	switch item.ConflictResolution {
	case "keep_local":
		applyItem(&existing, item)
		if err := tx.Save(&existing).Error; err != nil {
			return SyncPlanResult{}, err
		}
		return SyncPlanResult{ClientID: item.ClientID, PlanID: existing.ID, Status: "updated", Conflict: true, SyncedAt: &existing.UpdatedAt}, nil
	case "keep_cloud":
		return SyncPlanResult{ClientID: item.ClientID, PlanID: existing.ID, Status: "kept_cloud", Conflict: true, SyncedAt: &existing.UpdatedAt}, nil
	default: // save_as_new
		newClientID := genClientID()
		np := model.Plan{
			Source:      model.PlanSourcePersonal,
			OwnerID:     actorID,
			ClientID:    &newClientID,
			PlanGroupID: item.PlanGroupID,
			Title:       item.Title, Note: item.Note,
			StartTime: normalizePlanTime(item.StartTime), EndTime: normalizePlanTime(item.EndTime),
			Status: model.PlanNotStarted, CreatedBy: actorID,
		}
		if err := tx.Create(&np).Error; err != nil {
			return SyncPlanResult{}, err
		}
		return SyncPlanResult{ClientID: item.ClientID, NewClientID: newClientID, PlanID: np.ID, Status: "saved_new", Conflict: true, SyncedAt: &np.UpdatedAt}, nil
	}
}

// applyItem 用上传内容覆盖已有计划（不含 client_id、source、owner）。
func applyItem(p *model.Plan, item SyncPlanItem) {
	p.Title = item.Title
	p.Note = item.Note
	p.PlanGroupID = item.PlanGroupID
	p.StartTime = normalizePlanTime(item.StartTime)
	p.EndTime = normalizePlanTime(item.EndTime)
}

// genClientID 生成一个新的客户端标识（save_as_new 时用）。
func genClientID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
