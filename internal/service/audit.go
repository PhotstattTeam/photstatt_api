package service

import (
	"encoding/json"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// LogAudit 写入一条审计日志。before / after 传 JSON 字符串（可为空）。
// PRD 七：项目创建/归档、排期、计划改期/删除、成员与角色变更、管理员任命撤销等须留痕。
func LogAudit(db *gorm.DB, actorID uint, action, targetType string, targetID uint, before, after string) error {
	return db.Create(&model.AuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Before:     before,
		After:      after,
	}).Error
}

// toJSON 把结构体序列化成 JSON 字符串，供审计日志记录前后值。
func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
