package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// PersonalPlanService 承载个人计划与个人计划组业务（PRD 6.5 + 权限矩阵表 D）。
// 个人计划在普通成员之间完全不互通：只有本人与系统管理员两端可见。
type PersonalPlanService struct {
	DB *gorm.DB
}

func NewPersonalPlanService(db *gorm.DB) *PersonalPlanService {
	return &PersonalPlanService{DB: db}
}

type CreatePersonalPlanInput struct {
	Title       string    `json:"title" binding:"required"`
	Note        string    `json:"note"`
	PlanGroupID *uint     `json:"plan_group_id"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
}

// UpdatePersonalPlanInput 全量更新语义（前端总是传完整对象）。
type UpdatePersonalPlanInput struct {
	Title       string    `json:"title" binding:"required"`
	Note        string    `json:"note"`
	PlanGroupID *uint     `json:"plan_group_id"` // null/省略 = 未分组
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
}

type PersonalPlanStateInput struct {
	Status   *string `json:"status"`
	Progress *int    `json:"progress"`
}

type PlanGroupInput struct {
	Name string `json:"name" binding:"required"`
	Sort int    `json:"sort"`
}

type PersonalPlanFilter struct {
	PlanGroupID uint   `form:"plan_group_id"`
	Status      string `form:"status"`
	From        string `form:"from"` // RFC3339，可选
	To          string `form:"to"`   // RFC3339，可选
}

// AssignedPlanDTO 虚拟"项目计划组"中的计划，附带项目/子项目名用于分组展示。
type AssignedPlanDTO struct {
	ID             uint      `json:"id"`
	Title          string    `json:"title"`
	Note           string    `json:"note"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Progress       int       `json:"progress"`
	ProgressManual bool      `json:"progress_manual"`
	Status         string    `json:"status"`
	Source         string    `json:"source"`
	OwnerID        uint      `json:"owner_id"`
	IssueID        *uint     `json:"issue_id"`
	ProjectID      *uint     `json:"project_id"`
	SubProjectID   *uint     `json:"sub_project_id"`
	Overdue        bool      `json:"overdue,omitempty"`
	ProjectName    string    `json:"project_name"`
	SubProjectName string    `json:"sub_project_name"`
}

// Create 创建个人计划（来源=个人自建）。仅本人或系统管理员。
func (s *PersonalPlanService) Create(actorID, ownerID uint, in CreatePersonalPlanInput) (*model.Plan, error) {
	var p *model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := s.canAccess(actorID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if err := validatePlanTimes(in.StartTime, in.EndTime); err != nil {
			return err
		}
		if err := validatePlanGroup(tx, ownerID, in.PlanGroupID); err != nil {
			return err
		}
		p = &model.Plan{
			Source:      model.PlanSourcePersonal,
			PlanGroupID: in.PlanGroupID,
			Title:       in.Title, Note: in.Note,
			StartTime: normalizePlanTime(in.StartTime), EndTime: normalizePlanTime(in.EndTime),
			OwnerID: ownerID, Status: model.PlanNotStarted, CreatedBy: actorID,
		}
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "personal_plan.create", "plan", p.ID, "", toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	setOverdue(p)
	return p, nil
}

// List 列出某用户的个人计划。仅本人或系统管理员。
func (s *PersonalPlanService) List(actorID, ownerID uint, f PersonalPlanFilter) ([]model.Plan, error) {
	ok, err := s.canAccess(actorID, ownerID)
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

	q := s.DB.Model(&model.Plan{}).Where("owner_id = ? AND source = ?", ownerID, model.PlanSourcePersonal)
	if f.PlanGroupID != 0 {
		q = q.Where("plan_group_id = ?", f.PlanGroupID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if start != nil {
		q = q.Where("end_time >= ?", *start)
	}
	if end != nil {
		q = q.Where("start_time <= ?", *end)
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

// Get 查看单个个人计划。仅本人或系统管理员。
func (s *PersonalPlanService) Get(actorID, ownerID, planID uint) (*model.Plan, error) {
	ok, err := s.canAccess(actorID, ownerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var p model.Plan
	if err := s.DB.Where("id = ? AND owner_id = ? AND source = ?", planID, ownerID, model.PlanSourcePersonal).First(&p).Error; err != nil {
		return nil, ErrNotFound
	}
	setOverdue(&p)
	return &p, nil
}

// Update 编辑个人计划（全量）。仅本人或系统管理员。
func (s *PersonalPlanService) Update(actorID, ownerID, planID uint, in UpdatePersonalPlanInput) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := s.canAccess(actorID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if err := tx.Where("id = ? AND owner_id = ? AND source = ?", planID, ownerID, model.PlanSourcePersonal).First(&p).Error; err != nil {
			return ErrNotFound
		}
		if err := validatePlanTimes(in.StartTime, in.EndTime); err != nil {
			return err
		}
		if err := validatePlanGroup(tx, ownerID, in.PlanGroupID); err != nil {
			return err
		}
		before := toJSON(p)
		p.Title = in.Title
		p.Note = in.Note
		p.PlanGroupID = in.PlanGroupID
		p.StartTime = normalizePlanTime(in.StartTime)
		p.EndTime = normalizePlanTime(in.EndTime)
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "personal_plan.update", "plan", planID, before, toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// Delete 删除个人计划。仅本人或系统管理员。
func (s *PersonalPlanService) Delete(actorID, ownerID, planID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := s.canAccess(actorID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		var p model.Plan
		if err := tx.Where("id = ? AND owner_id = ? AND source = ?", planID, ownerID, model.PlanSourcePersonal).First(&p).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(p)
		if err := tx.Delete(&p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "personal_plan.delete", "plan", planID, before, "")
	})
}

// UpdateState 更新个人计划状态/进度。仅本人或系统管理员。
func (s *PersonalPlanService) UpdateState(actorID, ownerID, planID uint, in PersonalPlanStateInput) (*model.Plan, error) {
	var p model.Plan
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := s.canAccess(actorID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if err := tx.Where("id = ? AND owner_id = ? AND source = ?", planID, ownerID, model.PlanSourcePersonal).First(&p).Error; err != nil {
			return ErrNotFound
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
			p.ProgressManual = true
		}
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "personal_plan.state_update", "plan", planID, before, toJSON(p))
	})
	if err != nil {
		return nil, err
	}
	setOverdue(&p)
	return &p, nil
}

// ListAssigned 虚拟"项目计划组"：聚合"负责人=当前用户且来源=项目排期"的计划（US-ME-03）。
// 已归档项目下的计划不出现。
func (s *PersonalPlanService) ListAssigned(userID uint) ([]AssignedPlanDTO, error) {
	var rows []AssignedPlanDTO
	err := s.DB.Model(&model.Plan{}).
		Select("plans.id, plans.title, plans.note, plans.start_time, plans.end_time, plans.progress, plans.progress_manual, plans.status, plans.source, plans.owner_id, plans.issue_id, plans.project_id, plans.sub_project_id, projects.name AS project_name, sub_projects.name AS sub_project_name").
		Joins("JOIN projects ON projects.id = plans.project_id").
		Joins("LEFT JOIN sub_projects ON sub_projects.id = plans.sub_project_id").
		Where("plans.owner_id = ? AND plans.source = ? AND projects.status <> ?", userID, model.PlanSourceProject, model.ProjectArchived).
		Order("plans.start_time ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range rows {
		rows[i].Overdue = rows[i].EndTime.Before(now) &&
			rows[i].Status != model.PlanDone && rows[i].Status != model.PlanCancelled
	}
	return rows, nil
}

// ---- 个人计划组 ----

// ListGroups 列出计划组。仅本人或系统管理员。
func (s *PersonalPlanService) ListGroups(actorID, ownerID uint) ([]model.PlanGroup, error) {
	ok, err := s.canAccess(actorID, ownerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var groups []model.PlanGroup
	err = s.DB.Where("user_id = ?", ownerID).Order("sort ASC, id ASC").Find(&groups).Error
	return groups, err
}

// CreateGroup 创建计划组。仅本人或系统管理员。
func (s *PersonalPlanService) CreateGroup(actorID, ownerID uint, in PlanGroupInput) (*model.PlanGroup, error) {
	ok, err := s.canAccess(actorID, ownerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	g := model.PlanGroup{UserID: ownerID, Name: in.Name, Sort: in.Sort}
	if err := s.DB.Create(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// UpdateGroup 修改计划组（改名/排序）。仅本人或系统管理员。
func (s *PersonalPlanService) UpdateGroup(actorID, ownerID, groupID uint, in PlanGroupInput) (*model.PlanGroup, error) {
	var g model.PlanGroup
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := s.canAccess(actorID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if err := tx.Where("id = ? AND user_id = ?", groupID, ownerID).First(&g).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(g)
		g.Name = in.Name
		g.Sort = in.Sort
		if err := tx.Save(&g).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "plan_group.update", "plan_group", groupID, before, toJSON(g))
	})
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// DeleteGroup 删除计划组。组内计划变为"未分组"，不随组删除（US-ME-02）。
func (s *PersonalPlanService) DeleteGroup(actorID, ownerID, groupID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		ok, err := s.canAccess(actorID, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		var g model.PlanGroup
		if err := tx.Where("id = ? AND user_id = ?", groupID, ownerID).First(&g).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(g)
		// 组内计划置为未分组
		if err := tx.Model(&model.Plan{}).
			Where("plan_group_id = ? AND owner_id = ?", groupID, ownerID).
			Update("plan_group_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Delete(&g).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "plan_group.delete", "plan_group", groupID, before, "")
	})
}

// LogAdminView 系统管理员查看他人个人计划时留痕（PRD 7）。
func (s *PersonalPlanService) LogAdminView(actorID, ownerID uint) error {
	return LogAudit(s.DB, actorID, "admin.view_personal_plans", "user", ownerID, "", "")
}

// canAccess 个人计划/计划组可见性：仅本人与系统管理员（PRD 权限矩阵表 D）。
func (s *PersonalPlanService) canAccess(actorID, ownerID uint) (bool, error) {
	if actorID == ownerID {
		return true, nil
	}
	return IsSystemAdmin(s.DB, actorID)
}

// validatePlanGroup 校验计划组属于该用户。
func validatePlanGroup(tx *gorm.DB, ownerID uint, groupID *uint) error {
	if groupID == nil {
		return nil
	}
	var count int64
	if err := tx.Model(&model.PlanGroup{}).
		Where("id = ? AND user_id = ?", *groupID, ownerID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("计划组不存在或不属于该用户")
	}
	return nil
}
