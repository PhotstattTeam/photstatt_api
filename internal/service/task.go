package service

import (
	"time"

	"gorm.io/gorm"

	"consiliumous-backend/internal/model"
)

// TaskService 承载计划任务业务（PRD US-PLAN-02/03 + 权限矩阵表 C）。
type TaskService struct {
	DB *gorm.DB
}

func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{DB: db}
}

type CreateTaskInput struct {
	Title      string     `json:"title" binding:"required"`
	AssigneeID uint       `json:"assignee_id"`
	Sort       int        `json:"sort"`
	DueAt      *time.Time `json:"due_at"`
}

type UpdateTaskInput struct {
	Title      *string    `json:"title"`
	AssigneeID *uint      `json:"assignee_id"`
	Sort       *int       `json:"sort"`
	DueAt      *time.Time `json:"due_at"`
}

type TaskStatusInput struct {
	Status string `json:"status" binding:"required,oneof=todo in_progress done"`
}

// List 列出计划下的任务（项目成员可见）。
func (s *TaskService) List(projectID, planID, userID uint) ([]model.Task, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	if _, err := s.loadPlan(s.DB, projectID, planID); err != nil {
		return nil, err
	}
	var tasks []model.Task
	err = s.DB.Where("plan_id = ?", planID).Order("sort ASC, id ASC").Find(&tasks).Error
	return tasks, err
}

// Create 新增任务。权限：系统管理员/项目管理员/计划负责人（PRD 表 C：仅限本计划）。
func (s *TaskService) Create(actorID, projectID, planID uint, in CreateTaskInput) (*model.Task, error) {
	var t *model.Task
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		plan, err := s.loadPlan(tx, projectID, planID)
		if err != nil {
			return err
		}
		ok, err := s.canManage(tx, projectID, actorID, plan)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if in.AssigneeID != 0 {
			if err := validateOwnerIsMember(tx, projectID, in.AssigneeID); err != nil {
				return err
			}
		}
		t = &model.Task{
			PlanID: planID, Title: in.Title, AssigneeID: in.AssigneeID,
			Sort: in.Sort, DueAt: in.DueAt, Status: model.TaskTodo,
		}
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "task.create", "task", t.ID, "", toJSON(t)); err != nil {
			return err
		}
		return recomputePlanProgress(tx, planID)
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Update 编辑任务（标题/负责人/排序/截止时间）。权限同 Create。
func (s *TaskService) Update(actorID, projectID, planID, taskID uint, in UpdateTaskInput) (*model.Task, error) {
	var t model.Task
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		plan, err := s.loadPlan(tx, projectID, planID)
		if err != nil {
			return err
		}
		ok, err := s.canManage(tx, projectID, actorID, plan)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Where("id = ? AND plan_id = ?", taskID, planID).First(&t).Error; err != nil {
			return ErrNotFound
		}
		if in.AssigneeID != nil && *in.AssigneeID != 0 {
			if err := validateOwnerIsMember(tx, projectID, *in.AssigneeID); err != nil {
				return err
			}
		}
		before := toJSON(t)
		if in.Title != nil {
			t.Title = *in.Title
		}
		if in.AssigneeID != nil {
			t.AssigneeID = *in.AssigneeID
		}
		if in.Sort != nil {
			t.Sort = *in.Sort
		}
		if in.DueAt != nil {
			t.DueAt = in.DueAt
		}
		if err := tx.Save(&t).Error; err != nil {
			return err
		}
		return LogAudit(tx, actorID, "task.update", "task", taskID, before, toJSON(t))
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete 删除任务。权限同 Create。删除后重算计划进度。
func (s *TaskService) Delete(actorID, projectID, planID, taskID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		plan, err := s.loadPlan(tx, projectID, planID)
		if err != nil {
			return err
		}
		ok, err := s.canManage(tx, projectID, actorID, plan)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		var t model.Task
		if err := tx.Where("id = ? AND plan_id = ?", taskID, planID).First(&t).Error; err != nil {
			return ErrNotFound
		}
		before := toJSON(t)
		if err := tx.Delete(&t).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "task.delete", "task", taskID, before, ""); err != nil {
			return err
		}
		return recomputePlanProgress(tx, planID)
	})
}

// UpdateStatus 更新任务状态。权限：管理员/计划负责人，或"该任务的负责人"（PRD 表 C）。
func (s *TaskService) UpdateStatus(actorID, projectID, planID, taskID uint, in TaskStatusInput) (*model.Task, error) {
	var t model.Task
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		plan, err := s.loadPlan(tx, projectID, planID)
		if err != nil {
			return err
		}
		if err := tx.Where("id = ? AND plan_id = ?", taskID, planID).First(&t).Error; err != nil {
			return ErrNotFound
		}
		ok, err := s.canUpdateStatus(tx, projectID, actorID, plan, t)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		before := toJSON(t)
		t.Status = in.Status
		if err := tx.Save(&t).Error; err != nil {
			return err
		}
		if err := LogAudit(tx, actorID, "task.status_update", "task", taskID, before, toJSON(t)); err != nil {
			return err
		}
		return recomputePlanProgress(tx, planID)
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// loadPlan 加载计划（限定属于该项目）。
func (s *TaskService) loadPlan(tx *gorm.DB, projectID, planID uint) (*model.Plan, error) {
	var p model.Plan
	if err := tx.Where("id = ? AND project_id = ?", planID, projectID).First(&p).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

// canManage 是否可增删改任务：系统管理员/项目管理员/计划负责人。
func (s *TaskService) canManage(tx *gorm.DB, projectID, actorID uint, plan *model.Plan) (bool, error) {
	if sys, err := IsSystemAdmin(tx, actorID); err != nil {
		return false, err
	} else if sys {
		return true, nil
	}
	if admin, err := IsProjectAdmin(tx, projectID, actorID); err != nil {
		return false, err
	} else if admin {
		return true, nil
	}
	return plan.OwnerID == actorID, nil
}

// canUpdateStatus 是否可更新任务状态：管理员/计划负责人，或"本人负责的任务"。
func (s *TaskService) canUpdateStatus(tx *gorm.DB, projectID, actorID uint, plan *model.Plan, task model.Task) (bool, error) {
	if ok, err := s.canManage(tx, projectID, actorID, plan); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	// 其他项目成员：仅自己负责的任务
	if task.AssigneeID != actorID {
		return false, nil
	}
	return IsProjectMember(tx, projectID, actorID)
}
