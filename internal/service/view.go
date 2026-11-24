package service

import (
	"consiliumous-backend/internal/model"
)

// MindmapSubProject 思维导图中的子项目节点。
type MindmapSubProject struct {
	ID    uint          `json:"id"`
	Name  string        `json:"name"`
	Plans []MindmapPlan `json:"plans"`
}

// MindmapPlan 思维导图中的计划节点。
type MindmapPlan struct {
	ID     uint          `json:"id"`
	Title  string        `json:"title"`
	Status string        `json:"status"`
	Tasks  []MindmapTask `json:"tasks"`
}

// MindmapTask 思维导图中的任务节点。
type MindmapTask struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// MindmapResult 项目思维导图数据（层级：项目 → 子项目 → 计划 → 任务，US-VIEW-05）。
type MindmapResult struct {
	ProjectID   uint                `json:"project_id"`
	ProjectName string              `json:"project_name"`
	SubProjects []MindmapSubProject `json:"sub_projects"`
}

// Mindmap 生成项目思维导图数据，受阶段（from/to）筛选影响。
func (s *PlanService) Mindmap(projectID, userID uint, from, to string) (*MindmapResult, error) {
	ok, err := IsProjectMemberOrSys(s.DB, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	var project model.Project
	if err := s.DB.First(&project, projectID).Error; err != nil {
		return nil, ErrNotFound
	}
	start, end, err := parseRange(from, to)
	if err != nil {
		return nil, err
	}

	var subProjects []model.SubProject
	if err := s.DB.Where("project_id = ?", projectID).Order("sort ASC, id ASC").Find(&subProjects).Error; err != nil {
		return nil, err
	}

	q := s.DB.Model(&model.Plan{}).Where("project_id = ? AND source = ? AND status <> ?", projectID, model.PlanSourceProject, model.PlanCancelled)
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

	var tasks []model.Task
	if len(plans) > 0 {
		planIDs := make([]uint, 0, len(plans))
		for _, p := range plans {
			planIDs = append(planIDs, p.ID)
		}
		if err := s.DB.Where("plan_id IN ?", planIDs).Order("sort ASC, id ASC").Find(&tasks).Error; err != nil {
			return nil, err
		}
	}

	// 建立索引
	taskByPlan := map[uint][]MindmapTask{}
	for _, t := range tasks {
		taskByPlan[t.PlanID] = append(taskByPlan[t.PlanID], MindmapTask{ID: t.ID, Title: t.Title, Status: t.Status})
	}
	planBySub := map[uint][]MindmapPlan{}
	for _, p := range plans {
		spID := uint(0)
		if p.SubProjectID != nil {
			spID = *p.SubProjectID
		}
		planBySub[spID] = append(planBySub[spID], MindmapPlan{ID: p.ID, Title: p.Title, Status: p.Status, Tasks: taskByPlan[p.ID]})
	}

	result := &MindmapResult{ProjectID: project.ID, ProjectName: project.Name, SubProjects: []MindmapSubProject{}}
	for _, sp := range subProjects {
		result.SubProjects = append(result.SubProjects, MindmapSubProject{ID: sp.ID, Name: sp.Name, Plans: planBySub[sp.ID]})
	}
	return result, nil
}
