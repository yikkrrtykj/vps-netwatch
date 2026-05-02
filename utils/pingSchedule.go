package utils

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/ws"
)

// PingTaskManager 管理定时器和任务
type PingTaskManager struct {
	mu         sync.Mutex
	cancelFunc context.CancelFunc
	tasks      map[int][]models.PingTask
}

var manager = &PingTaskManager{
	tasks: make(map[int][]models.PingTask),
}

// Reload 重载时间表
func (m *PingTaskManager) Reload(pingTasks []models.PingTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	m.tasks = make(map[int][]models.PingTask)

	// 按Interval分组任务
	taskGroups := make(map[int][]models.PingTask)
	for _, task := range pingTasks {
		if task.Interval <= 0 {
			continue
		}
		taskGroups[task.Interval] = append(taskGroups[task.Interval], task)
	}

	// 为每个唯一的Interval创建协程
	for interval, tasks := range taskGroups {
		m.tasks[interval] = tasks
		go m.runPreciseLoop(ctx, time.Duration(interval)*time.Second, tasks)
	}
	return nil
}

func (m *PingTaskManager) runPreciseLoop(ctx context.Context, interval time.Duration, tasks []models.PingTask) {
	// Start the first timer.
	timer := time.NewTimer(interval)

	// This will be the reference point for all future ticks.
	// By adding the interval to this time, we avoid accumulating execution delays.
	nextTick := time.Now().Add(interval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			onlineClients := ws.GetConnectedClients()
			for _, task := range tasks {
				go executePingTask(ctx, task, onlineClients)
			}

			nextTick = nextTick.Add(interval)
			timer.Reset(time.Until(nextTick))

		case <-ctx.Done():
			return
		}
	}
}

// executePingTask 执行单个PingTask
func executePingTask(ctx context.Context, task models.PingTask, onlineClients map[string]*ws.SafeConn) {
	var message struct {
		TaskID  uint   `json:"ping_task_id"`
		Message string `json:"message"`
		Type    string `json:"ping_type"`
		Target  string `json:"ping_target"`
	}

	message.Message = "ping"
	message.TaskID = task.Id
	message.Type = task.Type
	message.Target = task.Target

	// 根据 Cover 字段决定实际作用的 client 列表。
	// 这里以"当前在线 client"作为全集，避免向已离线节点发送。
	allOnlineUUIDs := make([]string, 0, len(onlineClients))
	for uuid := range onlineClients {
		allOnlineUUIDs = append(allOnlineUUIDs, uuid)
	}
	targetUUIDs := task.EffectiveClients(allOnlineUUIDs)

	sent, dropped := 0, 0
	for _, clientUUID := range targetUUIDs {
		select {
		case <-ctx.Done():
			// Context was canceled, stop sending pings.
			return
		default:
			// Context is still active, continue.
		}

		if conn, exists := onlineClients[clientUUID]; exists && conn != nil {
			if err := conn.WriteJSON(message); err != nil {
				dropped++
				continue
			}
			sent++
		} else {
			dropped++
		}
	}
	// 调度可视化日志：让用户能 grep 出来确认 cover=1/2 真的有发命令
	log.Printf("[ping-sched] task=%d cover=%d target=%s online=%d effective=%d sent=%d dropped=%d",
		task.Id, task.Cover, task.Target, len(allOnlineUUIDs), len(targetUUIDs), sent, dropped)
}

// ReloadPingSchedule 加载或重载时间表
func ReloadPingSchedule(pingTasks []models.PingTask) error {
	// 通知所有在线 agent 重新拉取任务清单
	// 这样 cover=1 / cover=2 这类任务（agent 之前没在 client 列表里、不知道 task ID 存在）
	// 才能立即被 agent 接受。否则 agent 会因为本地白名单校验丢弃未知 TaskID 的 ping 命令。
	go broadcastPingTasksReload()
	return manager.Reload(pingTasks)
}

// broadcastPingTasksReload 让所有在线 agent 重新拉 /api/clients/ping/tasks
func broadcastPingTasksReload() {
	notice := map[string]any{
		"message": "reload_ping_tasks",
	}
	for _, conn := range ws.GetConnectedClients() {
		if conn == nil {
			continue
		}
		_ = conn.WriteJSON(notice)
	}
}
