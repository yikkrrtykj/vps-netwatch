package tasks

import (
	"time"

	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/utils"
	"gorm.io/gorm"
)

func AddPingTask(clients []string, name string, target, task_type string, interval, cover int) (uint, error) {
	db := dbcore.GetDBInstance()
	task := models.PingTask{
		Clients:  clients,
		Cover:    cover,
		Name:     name,
		Type:     task_type,
		Target:   target,
		Interval: interval,
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&task).Error; err != nil {
			return err
		}

		// Append by id to avoid races between concurrent create requests.
		result := tx.Model(&models.PingTask{}).Where("id = ?", task.Id).Update("weight", int(task.Id))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	ReloadPingSchedule()
	return task.Id, nil
}

func DeletePingTask(id []uint) error {
	db := dbcore.GetDBInstance()
	result := db.Where("id IN ?", id).Delete(&models.PingTask{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	ReloadPingSchedule()
	return result.Error
}

func EditPingTask(tasks []*models.PingTask) error {
	db := dbcore.GetDBInstance()
	for _, task := range tasks {
		// Select("*") 让 GORM 写入所有字段（包括 zero value 如 Cover=0），
		// 否则把 Cover 从 1 改回 0 会被默认的"忽略 zero value"逻辑吞掉。
		result := db.Model(&models.PingTask{}).Where("id = ?", task.Id).Select("*").Updates(task)
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	ReloadPingSchedule()
	return nil
}

func GetAllPingTasks() ([]models.PingTask, error) {
	db := dbcore.GetDBInstance()
	var tasks []models.PingTask
	if err := db.Order("weight ASC").Order("id ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetPingTasksByClient 返回该 client 实际参与的所有 ping 任务，考虑 Cover 字段：
//   Cover=0 (include) → 仅 Clients 列表里的节点
//   Cover=1 (all)     → 所有节点
//   Cover=2 (exclude) → 不在 Clients 列表里的节点
//
// 为了兼容老 agent（它会把 task.Clients 当白名单校验，未知 task_id 的 ping 命令直接丢弃），
// 对 cover=1/cover=2 命中的任务，返回时把 Clients 替换为 [自己的 uuid]。
// 这样 agent 视角下所有任务都长成 cover=0 的样子，不会再被白名单拒绝。
func GetPingTasksByClient(uuid string) []models.PingTask {
	db := dbcore.GetDBInstance()
	var allTasks []models.PingTask
	if err := db.Find(&allTasks).Error; err != nil {
		return nil
	}
	result := make([]models.PingTask, 0, len(allTasks))
	selfClients := models.StringArray([]string{uuid})
	for _, t := range allTasks {
		switch t.Cover {
		case 1:
			// 全部节点 → 给 agent 一份只含自己的 view
			view := t
			view.Clients = selfClients
			result = append(result, view)
		case 2:
			// 排除模式：uuid 不在 Clients 列表则包含
			excluded := false
			for _, c := range t.Clients {
				if c == uuid {
					excluded = true
					break
				}
			}
			if !excluded {
				view := t
				view.Clients = selfClients
				result = append(result, view)
			}
		default:
			// 仅指定列表：原样返回（Clients 已经合法）
			for _, c := range t.Clients {
				if c == uuid {
					result = append(result, t)
					break
				}
			}
		}
	}
	return result
}

func UpdatePingTaskOrder(order map[uint]int) error {
	db := dbcore.GetDBInstance()
	err := db.Transaction(func(tx *gorm.DB) error {
		for id, weight := range order {
			result := tx.Model(&models.PingTask{}).Where("id = ?", id).Update("weight", weight)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	ReloadPingSchedule()
	return nil
}

func SavePingRecord(record models.PingRecord) error {
	db := dbcore.GetDBInstance()
	return db.Create(&record).Error
}

func DeletePingRecordsBefore(time time.Time) error {
	db := dbcore.GetDBInstance()
	err := db.Where("time < ?", time).Delete(&models.PingRecord{}).Error
	return err
}

func DeletePingRecords(id []uint) error {
	db := dbcore.GetDBInstance()
	result := db.Where("task_id IN ?", id).Delete(&models.PingRecord{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func DeleteAllPingRecords() error {
	db := dbcore.GetDBInstance()
	result := db.Exec("DELETE FROM ping_records")
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
func ReloadPingSchedule() error {
	db := dbcore.GetDBInstance()
	var pingTasks []models.PingTask
	if err := db.Find(&pingTasks).Error; err != nil {
		return err
	}
	return utils.ReloadPingSchedule(pingTasks)
}

func GetPingRecords(uuid string, taskId int, start, end time.Time) ([]models.PingRecord, error) {
	db := dbcore.GetDBInstance()
	var records []models.PingRecord
	dbQuery := db.Model(&models.PingRecord{})
	if uuid != "" {
		dbQuery = dbQuery.Where("client = ?", uuid)
	}
	if taskId >= 0 {
		dbQuery = dbQuery.Where("task_id = ?", uint(taskId))
	}
	if err := dbQuery.Where("time >= ? AND time <= ?", start, end).Order("time DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
