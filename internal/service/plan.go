package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// PlanService 承载项目计划与排期业务（PRD 6.3 / 权限矩阵表 C）。
type PlanService struct {
	DB *gorm.DB
}

func NewPlanService(db *gorm.DB) *PlanService {
	return &PlanService{DB: db}
}

type CreatePlanInput struct {
	SubProjectID uint      `json:"sub_project_id" binding:"required"`
	Title        string    `json:"title" binding:"required"`
	Note         string    `json:"note"`
	StartTime    time.Time `json:"start_time" binding:"required"`
	EndTime      time.Time `json:"end_time" binding:"required"`
	OwnerID      uint      `json:"owner_id" binding:"required"`
	IssueID      *uint     `json:"issue_id"` // 可选：直接建计划也可关联 Issue
}

type ScheduleItem struct {
	IssueID      uint      `json:"issue_id" binding:"required"`
	Title        string    `json:"title" binding:"required"`
	SubProjectID uint      `json:"sub_project_id" binding:"required"`
	OwnerID      uint      `json:"owner_id" binding:"required"`
	StartTime    time.Time `json:"start_time" binding:"required"`
	EndTime      time.Time `json:"end_time" binding:"required"`
	Note         string    `json:"note"`
}

type ScheduleInput struct {
	Plans []ScheduleItem `json:"plans" binding:"required"`
}

type UpdatePlanInput struct {
	SubProjectID *uint      `json:"sub_project_id"`
	Title        *string    `json:"title"`
	Note         *string    `json:"note"`
	StartTime    *time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
}

type AssignOwnerInput struct {
	OwnerID uint `json:"owner_id" binding:"required"`
}

type LinkIssueInput struct {
	IssueID uint `json:"issue_id" binding:"required"`
}

type PlanStateInput struct {
	Status   *string `json:"status"`
	Progress *int    `json:"progress"`
}

type PlanFilter struct {
	SubProjectID uint   `form:"sub_project_id"`
	OwnerID      uint   `form:"owner_id"`
	IssueID      uint   `form:"issue_id"`
	Status       string `form:"status"`
	Keyword      string `form:"keyword"`
	From         string `form:"from"` // RFC3339，可选：阶段区间过滤（与计划时间重叠）
	To           string `form:"to"`   // RFC3339，可选
}

// CreateDirect 直接新建项目计划，不经过 Issue（US-SCHED-03）。仅管理员。
func (s *PlanService) CreateDirect(actorID, projectID uint, in CreatePlanInput) (*model.Plan, error) {
	var p *model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := validateSubProject(tx, projectID, in.SubProjectID); err != nil {
			return err
		}
		if err := validateOwnerIsMember(tx, projectID, in.OwnerID); err != nil {
			return err
		}
		if err := validatePlanTimes(in.StartTime, in.EndTime); err != nil {
			return err
		}
		if in.IssueID != nil {
			if err := validateIssueInProject(tx, projectID, *in.IssueID); err != nil {
				return err
			}
		}
		p = &model.Plan{
			ProjectID: &projectID, SubProjectID: &in.SubProjectID,
			Source: model.PlanSourceProject,
			Title:  in.Title, Note: in.Note,
			StartTime: normalizePlanTime(in.StartTime), EndTime: normalizePlanTime(in.EndTime),
			OwnerID: in.OwnerID, IssueID: in.IssueID,
			Status: model.PlanNotStarted, CreatedBy: actorID,
		}
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "plan.create", "plan", p.ID, "", toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	setOverdue(p)
	return p, nil
}

// ScheduleFromIssues 勾选若干 Issue 批量排期，生成 n 个计划，Issue 状态置为"已排期"（US-SCHED-01）。
func (s *PlanService) ScheduleFromIssues(actorID, projectID uint, in ScheduleInput) ([]model.Plan, error) {
	var created []model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}

		for _, item := range in.Plans {
			if err := validateIssueInProject(tx, projectID, item.IssueID); err != nil {
				return err
			}
			if err := validateSubProject(tx, projectID, item.SubProjectID); err != nil {
				return err
			}
			if err := validateOwnerIsMember(tx, projectID, item.OwnerID); err != nil {
				return err
			}
			if err := validatePlanTimes(item.StartTime, item.EndTime); err != nil {
				return err
			}

			issueID := item.IssueID
			p := model.Plan{
				ProjectID: &projectID, SubProjectID: &item.SubProjectID,
				Source: model.PlanSourceProject,
				Title:  item.Title, Note: item.Note,
				StartTime: normalizePlanTime(item.StartTime), EndTime: normalizePlanTime(item.EndTime),
				OwnerID: item.OwnerID, IssueID: &issueID,
				Status: model.PlanNotStarted, CreatedBy: actorID,
			}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			created = append(created, p)

			if err := markIssueScheduled(tx, actorID, item.IssueID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range created {
		setOverdue(&created[i])
	}
	return created, nil
}

// List 列出项目计划（成员可见）。附带派生状态"已延期"。
func (s *PlanService) List(projectID, userID uint, f PlanFilter) ([]model.Plan, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	start, end, err := parseRange(f.From, f.To)
	if err != nil {
		return nil, err
	}

	q := s.DB.Model(&model.Plan{}).Where("project_id = ?", projectID)
	if f.SubProjectID != 0 {
		q = q.Where("sub_project_id = ?", f.SubProjectID)
	}
	if start != nil {
		q = q.Where("end_time >= ?", *start)
	}
	if end != nil {
		q = q.Where("start_time <= ?", *end)
	}
	if f.OwnerID != 0 {
		q = q.Where("owner_id = ?", f.OwnerID)
	}
	if f.IssueID != 0 {
		q = q.Where("issue_id = ?", f.IssueID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("title LIKE ? OR note LIKE ?", like, like)
	}

	var plans []model.Plan
	if err := q.Order("start_time ASC").Find(&plans).Error; err != nil {
		return nil, err
	}
	for i := range plans {
		setOverdue(&plans[i])
	}
	return plans, nil
}

// Get 查看单个计划。
func (s *PlanService) Get(projectID, planID, userID uint) (*model.Plan, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var p model.Plan
	if err := s.DB.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
		return nil, ErrNotFound
	}
	setOverdue(&p)
	return &p, nil
}

// Update 编辑计划标题/提示/起止时间/子项目（US-SCHED-04，仅管理员，修改留痕）。
func (s *PlanService) Update(actorID, projectID, planID uint, in UpdatePlanInput) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(p)

		if in.SubProjectID != nil {
			if err := validateSubProject(tx, projectID, *in.SubProjectID); err != nil {
				return err
			}
			p.SubProjectID = in.SubProjectID
		}
		if in.Title != nil {
			p.Title = *in.Title
		}
		if in.Note != nil {
			p.Note = *in.Note
		}
		start, end := p.StartTime, p.EndTime
		if in.StartTime != nil {
			start = *in.StartTime
		}
		if in.EndTime != nil {
			end = *in.EndTime
		}
		if err := validatePlanTimes(start, end); err != nil {
			return err
		}
		p.StartTime = normalizePlanTime(start)
		p.EndTime = normalizePlanTime(end)

		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "plan.update", "plan", planID, before, toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// AssignOwner 分配/变更负责人（仅管理员）。
func (s *PlanService) AssignOwner(actorID, projectID, planID uint, in AssignOwnerInput) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		if err := validateOwnerIsMember(tx, projectID, in.OwnerID); err != nil {
			return err
		}
		before := toJSON(p)
		p.OwnerID = in.OwnerID
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "plan.assign_owner", "plan", planID, before, toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// LinkIssue 把计划改挂到另一个 Issue（US-SCHED-02）。
func (s *PlanService) LinkIssue(actorID, projectID, planID, newIssueID uint) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		if err := validateIssueInProject(tx, projectID, newIssueID); err != nil {
			return err
		}

		oldIssueID := p.IssueID
		before := toJSON(p)
		p.IssueID = &newIssueID
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "plan.link_issue", "plan", planID, before, toJSON(p)); err != nil {
			return err
		}

		// 原 Issue 若因此失去全部存活计划，自动回退"待处理"
		if oldIssueID != nil && *oldIssueID != newIssueID {
			if err := s.revertIssueIfNoPlans(tx, actorID, oldIssueID); err != nil {
				return err
			}
		}
		return markIssueScheduled(tx, actorID, newIssueID)
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// UnlinkIssue 解除计划与 Issue 的关联（US-SCHED-02）。
func (s *PlanService) UnlinkIssue(actorID, projectID, planID uint) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		oldIssueID := p.IssueID
		if oldIssueID == nil {
			return errors.New("该计划未关联任何 Issue")
		}
		before := toJSON(p)
		p.IssueID = nil
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "plan.unlink_issue", "plan", planID, before, toJSON(p)); err != nil {
			return err
		}
		return s.revertIssueIfNoPlans(tx, actorID, oldIssueID)
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// UpdateState 负责人更新计划状态与进度（PRD US-PLAN-03）。权限：管理员或负责人。
func (s *PlanService) UpdateState(actorID, projectID, planID uint, in PlanStateInput) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		admin, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !admin && p.OwnerID != actorID {
			return ErrForbidden
		}

		before := toJSON(p)
		if in.Status != nil {
			if !validPlanStatus(*in.Status) {
				return errors.New("非法计划状态")
			}
			p.Status = *in.Status
		}
		if in.Progress != nil {
			if *in.Progress < 0 || *in.Progress > 100 {
				return errors.New("进度必须在 0-100 之间")
			}
			p.Progress = *in.Progress
			p.ProgressManual = true // 人工覆盖后，任务状态变化不再自动回写（PRD 4.2）
		}
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "plan.state_update", "plan", planID, before, toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// Cancel 取消计划（仅管理员）。取消后若 Issue 无存活计划则回退待处理（US-SCHED-05）。
func (s *PlanService) Cancel(actorID, projectID, planID uint) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(p)
		p.Status = model.PlanCancelled
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "plan.cancel", "plan", planID, before, toJSON(p)); err != nil {
			return err
		}
		return s.revertIssueIfNoPlans(tx, actorID, p.IssueID)
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// Delete 删除计划（仅管理员）。删除后若 Issue 无存活计划则回退待处理（US-SCHED-05）。
func (s *PlanService) Delete(actorID, projectID, planID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		var p model.Plan
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(p)
		// 级联删除其任务（任务功能后续实现，先防御性清理）
		if err := tx.Where("plan_id = ?", planID).Delete(&model.Task{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&p).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "plan.delete", "plan", planID, before, ""); err != nil {
			return err
		}
		return s.revertIssueIfNoPlans(tx, actorID, p.IssueID)
	})
}

// revertIssueIfNoPlans 若该 Issue 名下已无存活计划，自动回到"待处理"并留痕（PRD 4.4）。
// "存活计划" = 未被删除且未取消的计划。
func (s *PlanService) revertIssueIfNoPlans(tx *gorm.DB, actorID uint, issueID *uint) error {
	if issueID == nil {
		return nil
	}
	var n int64
	if err := tx.Model(&model.Plan{}).
		Where("issue_id = ? AND status <> ?", *issueID, model.PlanCancelled).
		Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var is model.Issue
	if err := tx.First(&is, *issueID).Error; err != nil {
		return err
	}
	if is.Status != model.IssueScheduled {
		return nil
	}
	before := toJSON(is)
	if err := tx.Model(&is).Update("status", model.IssuePending).Error; err != nil {
		return err
	}
	return LogAudit(tx, actorID, "issue.auto_revert", "issue", is.ID, before, toJSON(is))
}

// markIssueScheduled 把 Issue 置为"已排期"（幂等，留痕）。
func markIssueScheduled(tx *gorm.DB, actorID, issueID uint) error {
	var is model.Issue
	if err := tx.First(&is, issueID).Error; err != nil {
		return err
	}
	if is.Status == model.IssueScheduled {
		return nil
	}
	before := toJSON(is)
	if err := tx.Model(&is).Update("status", model.IssueScheduled).Error; err != nil {
		return err
	}
	return LogAudit(tx, actorID, "issue.scheduled", "issue", issueID, before, toJSON(is))
}

// validatePlanTimes 校验起止时间。
func validatePlanTimes(start, end time.Time) error {
	if !end.After(start) {
		return errors.New("结束时间必须晚于开始时间")
	}
	return nil
}

// normalizePlanTime 时间归零到 5 分钟刻度（PRD 7）。
func normalizePlanTime(t time.Time) time.Time {
	return t.Truncate(5 * time.Minute)
}

// validateOwnerIsMember 校验负责人是项目成员。
func validateOwnerIsMember(tx *gorm.DB, projectID, ownerID uint) error {
	ok, err := IsProjectMember(tx, projectID, ownerID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("负责人必须是项目成员")
	}
	return nil
}

// validateIssueInProject 校验 Issue 属于该项目。
func validateIssueInProject(tx *gorm.DB, projectID, issueID uint) error {
	var count int64
	if err := tx.Model(&model.Issue{}).
		Where("id = ? AND project_id = ?", issueID, projectID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("Issue %d 不存在或不属于该项目", issueID)
	}
	return nil
}

// validPlanStatus 校验计划状态取值（取消只能走 Cancel 接口）。
func validPlanStatus(s string) bool {
	switch s {
	case model.PlanNotStarted, model.PlanInProgress, model.PlanDone:
		return true
	}
	return false
}

// setOverdue 计算派生状态"已延期"（PRD 4.4）：结束时间已过且未完成/未取消。
func setOverdue(p *model.Plan) {
	p.Overdue = p.EndTime.Before(time.Now()) &&
		p.Status != model.PlanDone && p.Status != model.PlanCancelled
}

// parseRange 解析 RFC3339 的 from/to 时间区间。
func parseRange(from, to string) (*time.Time, *time.Time, error) {
	var start, end *time.Time
	if from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return nil, nil, errors.New("from 参数格式应为 RFC3339")
		}
		start = &t
	}
	if to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return nil, nil, errors.New("to 参数格式应为 RFC3339")
		}
		end = &t
	}
	return start, end, nil
}

// RestoreAutoProgress 恢复进度自动计算（PRD US-PLAN-03 的"恢复自动"）。
// 权限：管理员或计划负责人。
func (s *PlanService) RestoreAutoProgress(actorID, projectID, planID uint) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
			return ErrNotFound
		}
		admin, err := IsProjectAdminOrSys(tx, projectID, actorID)
		if err != nil {
			return err
		}
		if !admin && p.OwnerID != actorID {
			return ErrForbidden
		}
		before := toJSON(p)
		p.ProgressManual = false
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "plan.restore_auto_progress", "plan", planID, before, toJSON(p)); err != nil {
			return err
		}
		return recomputePlanProgress(tx, planID)
	})
	if err != nil {
		return nil, err
	}
	// 重算发生在事务内，事务提交后重新读取拿到最新 progress
	if err := s.DB.First(&p, planID).Error; err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// recomputePlanProgress 按"已完成任务数 / 任务总数"重算计划进度。
// 仅当计划未被人工覆盖（progress_manual=false）时回写；无任务时不动（PRD 4.2）。
func recomputePlanProgress(tx *gorm.DB, planID uint) error {
	var plan model.Plan
	if err := tx.First(&plan, planID).Error; err != nil {
		return err
	}
	if plan.ProgressManual {
		return nil
	}
	var total, done int64
	if err := tx.Model(&model.Task{}).Where("plan_id = ?", planID).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return nil // 没有任务时由负责人手工填写，不自动算
	}
	if err := tx.Model(&model.Task{}).Where("plan_id = ? AND status = ?", planID, model.TaskDone).Count(&done).Error; err != nil {
		return err
	}
	progress := int((done*100 + total/2) / total) // 四舍五入
	return tx.Model(&plan).Update("progress", progress).Error
}
