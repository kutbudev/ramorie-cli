package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/terzigolu/josepshbrain-go/internal/api"
	"github.com/terzigolu/josepshbrain-go/internal/config"
)

type toolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func ToolDefinitions() []toolDef {
	return []toolDef{
		{
			Name:        "create_task",
			Description: "Yeni bir görev oluştur",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"description": map[string]interface{}{"type": "string"}, "priority": map[string]interface{}{"type": "string"}, "project": map[string]interface{}{"type": "string"}}, "required": []string{"description"}},
		},
		{
			Name:        "list_tasks",
			Description: "Görevleri listele",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"status": map[string]interface{}{"type": "string"}, "project": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}},
		},
		{
			Name:        "search_tasks",
			Description: "Görevlerde keyword arama yap",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"}, "project": map[string]interface{}{"type": "string"}, "tag": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}, "required": []string{"query"}},
		},
		{
			Name:        "get_next_tasks",
			Description: "Sıradaki görevleri öncelik sırasına göre getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"count": map[string]interface{}{"type": "number"}, "project": map[string]interface{}{"type": "string"}, "tag": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "get_task",
			Description: "Görev detaylarını getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "start_task",
			Description: "Görevi başlat (IN_PROGRESS)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "complete_task",
			Description: "Görevi tamamla (COMPLETED)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "stop_task",
			Description: "Görevi duraklat (aktif görevi temizler, IN_PROGRESS kalır)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "get_active_task",
			Description: "Şu anda aktif olan görevi getir (memory auto-link için)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "delete_task",
			Description: "Görevi sil",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "update_task_status",
			Description: "Görev durumunu güncelle",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"}}, "required": []string{"taskId", "status"}},
		},
		{
			Name:        "update_progress",
			Description: "Görev ilerleme durumunu güncelle (0-100)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "progress": map[string]interface{}{"type": "number"}}, "required": []string{"taskId", "progress"}},
		},
		{
			Name:        "add_task_note",
			Description: "Göreve not ekle (annotation)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "note": map[string]interface{}{"type": "string"}}, "required": []string{"taskId", "note"}},
		},
		{
			Name:        "create_subtask",
			Description: "Bir göreve alt görev ekle",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"parentTaskId": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"parentTaskId", "description"}},
		},
		{
			Name:        "bulk_start_tasks",
			Description: "Birden fazla görevi tek seferde başlat",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskIds": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"taskIds"}},
		},
		{
			Name:        "bulk_complete_tasks",
			Description: "Birden fazla görevi tek seferde tamamla",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskIds": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"taskIds"}},
		},
		{
			Name:        "list_projects",
			Description: "Projeleri listele",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "create_project",
			Description: "Yeni proje oluştur",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		{
			Name:        "set_active_project",
			Description: "Aktif projeyi değiştir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectName": map[string]interface{}{"type": "string"}}, "required": []string{"projectName"}},
		},
		{
			Name:        "get_project",
			Description: "Proje detaylarını getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectId": map[string]interface{}{"type": "string"}}, "required": []string{"projectId"}},
		},
		{
			Name:        "update_project",
			Description: "Projeyi güncelle",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectId": map[string]interface{}{"type": "string"}, "name": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"projectId"}},
		},
		{
			Name:        "delete_project",
			Description: "Projeyi sil",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"projectId": map[string]interface{}{"type": "string"}}, "required": []string{"projectId"}},
		},
		{
			Name:        "add_memory",
			Description: "Yeni bir hafıza/not ekle",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"content": map[string]interface{}{"type": "string"}, "project": map[string]interface{}{"type": "string"}}, "required": []string{"content"}},
		},
		{
			Name:        "list_memories",
			Description: "Hafızaları listele",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}, "term": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}},
		},
		{
			Name:        "get_task_memories",
			Description: "Bir görev ile ilişkili hafıza öğelerini getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "memory_tasks",
			Description: "Bir hafıza ile ilişkili görevleri getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "create_memory_task_link",
			Description: "Görev-hafıza linki oluştur (manual)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "memoryId": map[string]interface{}{"type": "string"}, "relationType": map[string]interface{}{"type": "string"}}, "required": []string{"taskId", "memoryId"}},
		},
		{
			Name:        "get_memory",
			Description: "Hafıza detaylarını getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "update_memory",
			Description: "Hafızayı güncelle",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}, "tags": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "delete_memory",
			Description: "Hafızayı sil",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"memoryId": map[string]interface{}{"type": "string"}}, "required": []string{"memoryId"}},
		},
		{
			Name:        "get_stats",
			Description: "Görev istatistiklerini getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "get_history",
			Description: "Son X günün görev aktivitesini getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"days": map[string]interface{}{"type": "number"}, "project": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "analyze_task_risks",
			Description: "Görev için risk analizi yap",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "analyze_task_dependencies",
			Description: "Görev için bağımlılık analizi yap",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "duplicate_task",
			Description: "Bir görevi kopyala (etiket ve notlarla; durum TODO, ilerleme 0)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskId": map[string]interface{}{"type": "string"}, "newDescription": map[string]interface{}{"type": "string"}}, "required": []string{"taskId"}},
		},
		{
			Name:        "move_tasks_to_project",
			Description: "Görevleri başka bir projeye taşı",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"taskIds": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "targetProject": map[string]interface{}{"type": "string"}}, "required": []string{"taskIds", "targetProject"}},
		},
		{
			Name:        "timeline",
			Description: "Son X günün aktivite zaman çizelgesi",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"days": map[string]interface{}{"type": "number"}, "project": map[string]interface{}{"type": "string"}}},
		},
		{
			Name:        "recall",
			Description: "Hafızalarda metin araması yap (keyword)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"term": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "number"}}, "required": []string{"term"}},
		},
		{
			Name:        "export_project",
			Description: "Proje raporunu markdown formatında dışa aktar",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"project": map[string]interface{}{"type": "string"}, "format": map[string]interface{}{"type": "string"}}, "required": []string{"project"}},
		},
		{
			Name:        "list_contexts",
			Description: "Mevcut tüm bağlamları (context) listele",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "create_context",
			Description: "Yeni bir bağlam (context) oluştur",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		{
			Name:        "set_active_context",
			Description: "Belirtilen bağlamı (context) etkinleştir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		// Context Pack tools
		{
			Name:        "list_context_packs",
			Description: "Context pack'leri listele (Active Context). Tip, durum ve arama ile filtrelenebilir.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"type": map[string]interface{}{"type": "string", "description": "Pack tipi: project, integration, decision, custom"}, "status": map[string]interface{}{"type": "string", "description": "Pack durumu: draft, published"}, "query": map[string]interface{}{"type": "string", "description": "İsim/açıklama araması"}, "limit": map[string]interface{}{"type": "number"}}},
		},
		{
			Name:        "get_context_pack",
			Description: "Belirli bir context pack'in detaylarını getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}}, "required": []string{"packId"}},
		},
		{
			Name:        "create_context_pack",
			Description: "Yeni bir context pack oluştur",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "type": map[string]interface{}{"type": "string", "description": "Pack tipi: project, integration, decision, custom"}, "description": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string", "description": "Pack durumu: draft, published"}, "tags": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"name", "type"}},
		},
		{
			Name:        "update_context_pack",
			Description: "Mevcut bir context pack'i güncelle",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}, "name": map[string]interface{}{"type": "string"}, "type": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"}, "tags": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"packId"}},
		},
		{
			Name:        "delete_context_pack",
			Description: "Bir context pack'i sil",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}}, "required": []string{"packId"}},
		},
		{
			Name:        "activate_context_pack",
			Description: "Bir context pack'i aktif (published) yap",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"packId": map[string]interface{}{"type": "string"}}, "required": []string{"packId"}},
		},
		{
			Name:        "get_active_context_pack",
			Description: "Aktif context pack'i getir (şu anki çalışma bağlamı)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		// Organization tools
		{
			Name:        "list_organizations",
			Description: "Organizasyonları listele",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "get_organization",
			Description: "Organizasyon detaylarını getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"orgId": map[string]interface{}{"type": "string"}}, "required": []string{"orgId"}},
		},
		{
			Name:        "create_organization",
			Description: "Yeni organizasyon oluştur",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}}, "required": []string{"name"}},
		},
		// Decision tools - for agents to record architectural decisions
		{
			Name:        "list_decisions",
			Description: "Kararları (ADR) listele. Agentlar aldıkları kararları burada kaydeder.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "description": "Durum filtresi: draft, proposed, approved, deprecated"}, "area": map[string]interface{}{"type": "string", "description": "Alan filtresi: Frontend, Backend, Architecture, etc."}, "limit": map[string]interface{}{"type": "number"}}},
		},
		{
			Name:        "get_decision",
			Description: "Belirli bir kararın detaylarını getir",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"decisionId": map[string]interface{}{"type": "string", "description": "Decision ID veya ADR numarası (örn: ADR-001)"}}, "required": []string{"decisionId"}},
		},
		{
			Name:        "create_decision",
			Description: "Yeni bir karar (ADR) oluştur. Agentlar önemli kararları burada kayıt altına alır.",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string", "description": "Karar başlığı"}, "description": map[string]interface{}{"type": "string", "description": "Kısa açıklama"}, "status": map[string]interface{}{"type": "string", "description": "Durum: draft, proposed, approved, deprecated"}, "area": map[string]interface{}{"type": "string", "description": "Alan: Frontend, Backend, Architecture, DevOps, etc."}, "context": map[string]interface{}{"type": "string", "description": "Kararın bağlamı ve nedeni"}, "consequences": map[string]interface{}{"type": "string", "description": "Kararın sonuçları ve etkileri"}}, "required": []string{"title"}},
		},
		{
			Name:        "update_decision",
			Description: "Mevcut bir kararı güncelle",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"decisionId": map[string]interface{}{"type": "string"}, "title": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"}, "area": map[string]interface{}{"type": "string"}, "context": map[string]interface{}{"type": "string"}, "consequences": map[string]interface{}{"type": "string"}}, "required": []string{"decisionId"}},
		},
		{
			Name:        "delete_decision",
			Description: "Bir kararı sil",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"decisionId": map[string]interface{}{"type": "string"}}, "required": []string{"decisionId"}},
		},
	}
}

func CallTool(client *api.Client, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "create_task":
		description, _ := args["description"].(string)
		description = strings.TrimSpace(description)
		if description == "" {
			return nil, errors.New("description is required")
		}
		priority, _ := args["priority"].(string)
		priority = normalizePriority(priority)
		projectIdentifier, _ := args["project"].(string)
		projectID, err := resolveProjectID(client, projectIdentifier)
		if err != nil {
			return nil, err
		}
		task, err := client.CreateTask(projectID, description, "", priority)
		if err != nil {
			return nil, err
		}
		return task, nil

	case "list_tasks":
		status, _ := args["status"].(string)
		projectIdentifier, _ := args["project"].(string)
		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}
		tasks, err := client.ListTasks(projectID, strings.TrimSpace(status))
		if err != nil {
			return nil, err
		}
		limit := toInt(args["limit"])
		if limit > 0 && limit < len(tasks) {
			tasks = tasks[:limit]
		}
		return tasks, nil

	case "search_tasks":
		query, _ := args["query"].(string)
		query = strings.TrimSpace(query)
		if query == "" {
			return nil, errors.New("query is required")
		}
		status, _ := args["status"].(string)
		projectIdentifier, _ := args["project"].(string)
		tag, _ := args["tag"].(string)
		limit := toInt(args["limit"])

		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}

		tags := []string{}
		if strings.TrimSpace(tag) != "" {
			tags = append(tags, strings.TrimSpace(tag))
		}

		tasks, err := client.ListTasksQuery(projectID, strings.TrimSpace(status), query, nil, tags)
		if err != nil {
			return nil, err
		}
		if limit > 0 && limit < len(tasks) {
			tasks = tasks[:limit]
		}
		return tasks, nil

	case "get_next_tasks":
		count := toInt(args["count"])
		if count <= 0 {
			count = 5
		}
		projectIdentifier, _ := args["project"].(string)
		tag, _ := args["tag"].(string)

		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}

		tags := []string{}
		if strings.TrimSpace(tag) != "" {
			tags = append(tags, strings.TrimSpace(tag))
		}

		// Default to TODO tasks
		tasks, err := client.ListTasksQuery(projectID, "TODO", "", nil, tags)
		if err != nil {
			return nil, err
		}

		sort.Slice(tasks, func(i, j int) bool {
			pi := priorityRank(tasks[i].Priority)
			pj := priorityRank(tasks[j].Priority)
			if pi != pj {
				return pi > pj
			}
			return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		})

		if count < len(tasks) {
			tasks = tasks[:count]
		}
		return tasks, nil

	case "get_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.GetTask(taskID)

	case "start_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.StartTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true}, nil

	case "complete_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.CompleteTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true}, nil

	case "stop_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.StopTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true}, nil

	case "get_active_task":
		return client.GetActiveTask()

	case "delete_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if err := client.DeleteTask(taskID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": taskID}, nil

	case "update_task_status":
		taskID, _ := args["taskId"].(string)
		status, _ := args["status"].(string)
		taskID = strings.TrimSpace(taskID)
		status = strings.TrimSpace(status)
		if taskID == "" || status == "" {
			return nil, errors.New("taskId and status are required")
		}
		return client.UpdateTask(taskID, map[string]interface{}{"status": status})

	case "update_progress":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		progress := toInt(args["progress"])
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		if progress < 0 || progress > 100 {
			return nil, errors.New("progress must be between 0 and 100")
		}
		return client.UpdateTask(taskID, map[string]interface{}{"progress": progress})

	case "add_task_note":
		taskID, _ := args["taskId"].(string)
		note, _ := args["note"].(string)
		taskID = strings.TrimSpace(taskID)
		note = strings.TrimSpace(note)
		if taskID == "" || note == "" {
			return nil, errors.New("taskId and note are required")
		}
		return client.CreateAnnotation(taskID, note)

	case "create_subtask":
		parentTaskID, _ := args["parentTaskId"].(string)
		description, _ := args["description"].(string)
		parentTaskID = strings.TrimSpace(parentTaskID)
		description = strings.TrimSpace(description)
		if parentTaskID == "" || description == "" {
			return nil, errors.New("parentTaskId and description are required")
		}
		return client.CreateSubtask(parentTaskID, description)

	case "bulk_start_tasks":
		ids, err := resolveTaskIDList(client, args["taskIds"])
		if err != nil {
			return nil, err
		}
		status := "IN_PROGRESS"
		if err := client.BulkUpdateTasks(ids, &status, nil, nil); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "count": len(ids)}, nil

	case "bulk_complete_tasks":
		ids, err := resolveTaskIDList(client, args["taskIds"])
		if err != nil {
			return nil, err
		}
		status := "COMPLETED"
		if err := client.BulkUpdateTasks(ids, &status, nil, nil); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "count": len(ids)}, nil

	case "list_projects":
		return client.ListProjects()

	case "create_project":
		name, _ := args["name"].(string)
		description, _ := args["description"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.CreateProject(name, strings.TrimSpace(description))

	case "set_active_project":
		projectName, _ := args["projectName"].(string)
		projectName = strings.TrimSpace(projectName)
		if projectName == "" {
			return nil, errors.New("projectName is required")
		}
		projects, err := client.ListProjects()
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.Name == projectName || strings.HasPrefix(p.ID.String(), projectName) {
				if err := client.SetProjectActive(p.ID.String()); err != nil {
					return nil, err
				}
				cfg, _ := config.LoadConfig()
				if cfg == nil {
					cfg = &config.Config{}
				}
				cfg.ActiveProjectID = p.ID.String()
				_ = config.SaveConfig(cfg)
				return map[string]interface{}{"ok": true, "project_id": p.ID.String(), "name": p.Name}, nil
			}
		}
		return nil, errors.New("project not found")

	case "get_project":
		projectID, _ := args["projectId"].(string)
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return nil, errors.New("projectId is required")
		}
		return client.GetProject(projectID)

	case "update_project":
		projectID, _ := args["projectId"].(string)
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return nil, errors.New("projectId is required")
		}
		updates := make(map[string]interface{})
		if name, ok := args["name"].(string); ok && strings.TrimSpace(name) != "" {
			updates["name"] = strings.TrimSpace(name)
		}
		if description, ok := args["description"].(string); ok {
			updates["description"] = strings.TrimSpace(description)
		}
		return client.UpdateProject(projectID, updates)

	case "delete_project":
		projectID, _ := args["projectId"].(string)
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return nil, errors.New("projectId is required")
		}
		if err := client.DeleteProject(projectID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": projectID}, nil

	case "add_memory":
		content, _ := args["content"].(string)
		content = strings.TrimSpace(content)
		if content == "" {
			return nil, errors.New("content is required")
		}
		projectIdentifier, _ := args["project"].(string)
		projectID, err := resolveProjectID(client, projectIdentifier)
		if err != nil {
			return nil, err
		}
		return client.CreateMemory(projectID, content)

	case "list_memories":
		projectIdentifier, _ := args["project"].(string)
		term, _ := args["term"].(string)
		projectID := ""
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err != nil {
				return nil, err
			}
			projectID = pid
		}
		memories, err := client.ListMemories(projectID, "")
		if err != nil {
			return nil, err
		}
		term = strings.TrimSpace(term)
		if term != "" {
			filtered := memories[:0]
			for _, m := range memories {
				if strings.Contains(strings.ToLower(m.Content), strings.ToLower(term)) {
					filtered = append(filtered, m)
				}
			}
			memories = filtered
		}
		limit := toInt(args["limit"])
		if limit > 0 && limit < len(memories) {
			memories = memories[:limit]
		}
		return memories, nil

	case "get_memory":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		return client.GetMemory(memoryID)

	case "update_memory":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		updates := make(map[string]interface{})
		if content, ok := args["content"].(string); ok && strings.TrimSpace(content) != "" {
			updates["content"] = strings.TrimSpace(content)
		}
		if tagsRaw, ok := args["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, strings.TrimSpace(s))
				}
			}
			updates["tags"] = tags
		}
		return client.UpdateMemory(memoryID, updates)

	case "delete_memory":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		if err := client.DeleteMemory(memoryID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": memoryID}, nil

	case "get_task_memories":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.ListTaskMemories(taskID)

	case "memory_tasks":
		memoryID, _ := args["memoryId"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return nil, errors.New("memoryId is required")
		}
		return client.ListMemoryTasks(memoryID)

	case "create_memory_task_link":
		taskID, _ := args["taskId"].(string)
		memoryID, _ := args["memoryId"].(string)
		relationType, _ := args["relationType"].(string)
		taskID = strings.TrimSpace(taskID)
		memoryID = strings.TrimSpace(memoryID)
		relationType = strings.TrimSpace(relationType)
		if taskID == "" || memoryID == "" {
			return nil, errors.New("taskId and memoryId are required")
		}
		b, err := client.CreateMemoryTaskLink(taskID, memoryID, relationType)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return map[string]interface{}{"ok": true}, nil
		}
		return out, nil

	case "get_stats":
		b, err := client.Request("GET", "/reports/stats", nil)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("invalid stats response")
		}
		return out, nil

	case "get_history":
		days := toInt(args["days"])
		if days == 0 {
			days = 7
		}
		endpoint := "/reports/history"
		if days > 0 {
			endpoint = fmt.Sprintf("/reports/history?days=%d", days)
		}
		b, err := client.Request("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("invalid history response")
		}
		return out, nil

	case "analyze_task_risks":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.AIRisks(taskID)

	case "analyze_task_dependencies":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		return client.AIDependencies(taskID)

	case "duplicate_task":
		taskID, _ := args["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return nil, errors.New("taskId is required")
		}
		newDescription, _ := args["newDescription"].(string)

		// Get original task
		original, err := client.GetTask(taskID)
		if err != nil {
			return nil, err
		}

		// Create new task with same properties
		title := original.Title
		if newDescription != "" {
			title = newDescription
		} else {
			title = title + " (kopya)"
		}

		newTask, err := client.CreateTask(
			original.ProjectID.String(),
			title,
			original.Description,
			original.Priority,
		)
		if err != nil {
			return nil, err
		}

		// Copy annotations
		for _, ann := range original.Annotations {
			_, _ = client.CreateAnnotation(newTask.ID.String(), ann.Content)
		}

		return map[string]interface{}{
			"ok":          true,
			"original_id": original.ID.String(),
			"new_id":      newTask.ID.String(),
			"title":       newTask.Title,
		}, nil

	case "move_tasks_to_project":
		taskIdsRaw := args["taskIds"]
		targetProject, _ := args["targetProject"].(string)
		targetProject = strings.TrimSpace(targetProject)
		if targetProject == "" {
			return nil, errors.New("targetProject is required")
		}

		taskIds, err := resolveTaskIDList(client, taskIdsRaw)
		if err != nil {
			return nil, err
		}

		// Resolve project
		projectID, err := resolveProjectID(client, targetProject)
		if err != nil {
			return nil, err
		}

		// Move each task
		movedCount := 0
		for _, id := range taskIds {
			_, err := client.UpdateTask(id, map[string]interface{}{"project_id": projectID})
			if err == nil {
				movedCount++
			}
		}

		return map[string]interface{}{
			"ok":         true,
			"moved":      movedCount,
			"total":      len(taskIds),
			"project_id": projectID,
		}, nil

	case "timeline":
		days := toInt(args["days"])
		if days == 0 {
			days = 7
		}
		projectIdentifier, _ := args["project"].(string)

		endpoint := fmt.Sprintf("/reports/history?days=%d", days)
		if strings.TrimSpace(projectIdentifier) != "" {
			pid, err := resolveProjectID(client, projectIdentifier)
			if err == nil {
				endpoint = fmt.Sprintf("/reports/history?days=%d&project_id=%s", days, pid)
			}
		}

		b, err := client.Request("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("invalid timeline response")
		}
		return out, nil

	case "recall":
		term, _ := args["term"].(string)
		term = strings.TrimSpace(term)
		if term == "" {
			return nil, errors.New("term is required")
		}
		limit := toInt(args["limit"])
		if limit == 0 {
			limit = 10
		}

		// Get all memories and filter by term
		memories, err := client.ListMemories("", "")
		if err != nil {
			return nil, err
		}

		var filtered []interface{}
		for _, m := range memories {
			if strings.Contains(strings.ToLower(m.Content), strings.ToLower(term)) {
				filtered = append(filtered, map[string]interface{}{
					"id":         m.ID.String(),
					"content":    m.Content,
					"created_at": m.CreatedAt,
				})
				if len(filtered) >= limit {
					break
				}
			}
		}

		return map[string]interface{}{
			"term":    term,
			"count":   len(filtered),
			"results": filtered,
		}, nil

	case "export_project":
		projectIdentifier, _ := args["project"].(string)
		format, _ := args["format"].(string)
		if format == "" {
			format = "markdown"
		}

		projectID, err := resolveProjectID(client, projectIdentifier)
		if err != nil {
			return nil, err
		}

		// Get project details
		projects, err := client.ListProjects()
		if err != nil {
			return nil, err
		}

		var project *struct {
			Name        string
			Description string
		}
		for _, p := range projects {
			if p.ID.String() == projectID {
				project = &struct {
					Name        string
					Description string
				}{p.Name, p.Description}
				break
			}
		}

		if project == nil {
			return nil, errors.New("project not found")
		}

		// Get tasks
		tasks, err := client.ListTasks(projectID, "")
		if err != nil {
			return nil, err
		}

		// Build markdown report
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", project.Name))
		if project.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", project.Description))
		}

		// Stats
		total := len(tasks)
		completed := 0
		inProgress := 0
		pending := 0
		for _, t := range tasks {
			switch t.Status {
			case "COMPLETED":
				completed++
			case "IN_PROGRESS":
				inProgress++
			default:
				pending++
			}
		}

		sb.WriteString("## İstatistikler\n\n")
		sb.WriteString(fmt.Sprintf("- **Toplam:** %d\n", total))
		sb.WriteString(fmt.Sprintf("- **Tamamlanan:** %d\n", completed))
		sb.WriteString(fmt.Sprintf("- **Devam Eden:** %d\n", inProgress))
		sb.WriteString(fmt.Sprintf("- **Bekleyen:** %d\n\n", pending))

		// Task list
		sb.WriteString("## Görevler\n\n")
		for _, t := range tasks {
			status := "⏳"
			if t.Status == "COMPLETED" {
				status = "✅"
			} else if t.Status == "IN_PROGRESS" {
				status = "🔄"
			}
			sb.WriteString(fmt.Sprintf("- %s **%s** [%s]\n", status, t.Title, t.Priority))
		}

		return map[string]interface{}{
			"project":  project.Name,
			"format":   format,
			"markdown": sb.String(),
		}, nil

	case "list_contexts":
		contexts, err := client.ListContexts()
		if err != nil {
			return nil, err
		}
		return contexts, nil

	case "create_context":
		name, _ := args["name"].(string)
		description, _ := args["description"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.CreateContext(name, strings.TrimSpace(description))

	case "set_active_context":
		name, _ := args["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.UseContext(name)

	// Context Pack tools
	case "list_context_packs":
		packType, _ := args["type"].(string)
		status, _ := args["status"].(string)
		query, _ := args["query"].(string)
		limit := toInt(args["limit"])

		response, err := client.ListContextPacks(
			strings.TrimSpace(packType),
			strings.TrimSpace(status),
			strings.TrimSpace(query),
			limit,
			0,
		)
		if err != nil {
			return nil, err
		}
		return response, nil

	case "get_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}
		return client.GetContextPack(packID)

	case "create_context_pack":
		name, _ := args["name"].(string)
		packType, _ := args["type"].(string)
		description, _ := args["description"].(string)
		status, _ := args["status"].(string)

		name = strings.TrimSpace(name)
		packType = strings.TrimSpace(packType)
		if name == "" {
			return nil, errors.New("name is required")
		}
		if packType == "" {
			packType = "custom"
		}

		// Parse tags
		var tags []string
		if tagsRaw, ok := args["tags"].([]interface{}); ok {
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, strings.TrimSpace(s))
				}
			}
		}

		return client.CreateContextPack(name, packType, strings.TrimSpace(description), strings.TrimSpace(status), tags)

	case "update_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}

		updates := make(map[string]interface{})
		if name, ok := args["name"].(string); ok && strings.TrimSpace(name) != "" {
			updates["name"] = strings.TrimSpace(name)
		}
		if packType, ok := args["type"].(string); ok && strings.TrimSpace(packType) != "" {
			updates["type"] = strings.TrimSpace(packType)
		}
		if description, ok := args["description"].(string); ok {
			updates["description"] = strings.TrimSpace(description)
		}
		if status, ok := args["status"].(string); ok && strings.TrimSpace(status) != "" {
			updates["status"] = strings.TrimSpace(status)
		}
		if tagsRaw, ok := args["tags"].([]interface{}); ok {
			var tags []string
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, strings.TrimSpace(s))
				}
			}
			updates["tags"] = tags
		}

		return client.UpdateContextPack(packID, updates)

	case "delete_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}
		if err := client.DeleteContextPack(packID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": packID}, nil

	case "activate_context_pack":
		packID, _ := args["packId"].(string)
		packID = strings.TrimSpace(packID)
		if packID == "" {
			return nil, errors.New("packId is required")
		}
		pack, err := client.SetActiveContextPack(packID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "pack": pack}, nil

	case "get_active_context_pack":
		return client.GetActiveContextPack()

	// Organization tools
	case "list_organizations":
		return client.ListOrganizations()

	case "get_organization":
		orgID, _ := args["orgId"].(string)
		orgID = strings.TrimSpace(orgID)
		if orgID == "" {
			return nil, errors.New("orgId is required")
		}
		return client.GetOrganization(orgID)

	case "create_organization":
		name, _ := args["name"].(string)
		description, _ := args["description"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		return client.CreateOrganization(name, strings.TrimSpace(description))

	// Decision tools - for agents to record architectural decisions
	case "list_decisions":
		status, _ := args["status"].(string)
		area, _ := args["area"].(string)
		limit := toInt(args["limit"])
		decisions, err := client.ListDecisions(strings.TrimSpace(status), strings.TrimSpace(area), limit)
		if err != nil {
			return nil, err
		}
		return decisions, nil

	case "get_decision":
		decisionID, _ := args["decisionId"].(string)
		decisionID = strings.TrimSpace(decisionID)
		if decisionID == "" {
			return nil, errors.New("decisionId is required")
		}
		return client.GetDecision(decisionID)

	case "create_decision":
		title, _ := args["title"].(string)
		title = strings.TrimSpace(title)
		if title == "" {
			return nil, errors.New("title is required")
		}
		description, _ := args["description"].(string)
		status, _ := args["status"].(string)
		area, _ := args["area"].(string)
		context, _ := args["context"].(string)
		consequences, _ := args["consequences"].(string)

		return client.CreateDecision(
			title,
			strings.TrimSpace(description),
			strings.TrimSpace(status),
			strings.TrimSpace(area),
			strings.TrimSpace(context),
			strings.TrimSpace(consequences),
		)

	case "update_decision":
		decisionID, _ := args["decisionId"].(string)
		decisionID = strings.TrimSpace(decisionID)
		if decisionID == "" {
			return nil, errors.New("decisionId is required")
		}

		updates := make(map[string]interface{})
		if title, ok := args["title"].(string); ok && strings.TrimSpace(title) != "" {
			updates["title"] = strings.TrimSpace(title)
		}
		if description, ok := args["description"].(string); ok {
			updates["description"] = strings.TrimSpace(description)
		}
		if status, ok := args["status"].(string); ok && strings.TrimSpace(status) != "" {
			updates["status"] = strings.TrimSpace(status)
		}
		if area, ok := args["area"].(string); ok && strings.TrimSpace(area) != "" {
			updates["area"] = strings.TrimSpace(area)
		}
		if context, ok := args["context"].(string); ok {
			updates["context"] = strings.TrimSpace(context)
		}
		if consequences, ok := args["consequences"].(string); ok {
			updates["consequences"] = strings.TrimSpace(consequences)
		}

		return client.UpdateDecision(decisionID, updates)

	case "delete_decision":
		decisionID, _ := args["decisionId"].(string)
		decisionID = strings.TrimSpace(decisionID)
		if decisionID == "" {
			return nil, errors.New("decisionId is required")
		}
		if err := client.DeleteDecision(decisionID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ok": true, "deleted": decisionID}, nil

	default:
		return nil, errors.New("tool not implemented")
	}
}

func priorityRank(p string) int {
	switch strings.ToUpper(strings.TrimSpace(p)) {
	case "H", "HIGH":
		return 3
	case "M", "MEDIUM":
		return 2
	case "L", "LOW":
		return 1
	default:
		return 0
	}
}

func resolveTaskIDList(client *api.Client, v interface{}) ([]string, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("taskIds must be an array")
	}
	if len(arr) == 0 {
		return nil, errors.New("taskIds cannot be empty")
	}
	ids := make([]string, 0, len(arr))
	for _, raw := range arr {
		s, _ := raw.(string)
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// backend supports short identifier in GET /tasks/:id, but bulk endpoints require full UUID
		t, err := client.GetTask(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, t.ID.String())
	}
	if len(ids) == 0 {
		return nil, errors.New("no valid task ids")
	}
	return ids, nil
}

func resolveProjectID(client *api.Client, projectIdentifier string) (string, error) {
	projectIdentifier = strings.TrimSpace(projectIdentifier)
	if projectIdentifier == "" {
		cfg, err := config.LoadConfig()
		if err == nil && cfg.ActiveProjectID != "" {
			return cfg.ActiveProjectID, nil
		}
		projects, err := client.ListProjects()
		if err != nil {
			return "", err
		}
		for _, p := range projects {
			if p.IsActive {
				return p.ID.String(), nil
			}
		}
		return "", errors.New("no active project")
	}

	if _, err := uuid.Parse(projectIdentifier); err == nil {
		return projectIdentifier, nil
	}

	projects, err := client.ListProjects()
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.Name == projectIdentifier || strings.HasPrefix(p.ID.String(), projectIdentifier) {
			return p.ID.String(), nil
		}
	}

	return "", errors.New("project not found")
}

func normalizePriority(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return "M"
	}
	switch s {
	case "H", "HIGH":
		return "H"
	case "M", "MEDIUM":
		return "M"
	case "L", "LOW":
		return "L"
	default:
		return "M"
	}
}

func toInt(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		var x int
		_, _ = fmt.Sscanf(t, "%d", &x)
		return x
	default:
		return 0
	}
}
