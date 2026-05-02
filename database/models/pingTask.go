package models

type PingRecord struct {
	Client     string    `json:"client" gorm:"type:varchar(36);not null;index"`
	ClientInfo Client    `json:"client_info" gorm:"foreignKey:Client;references:UUID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
	TaskId     uint      `json:"task_id" gorm:"not null;index"`
	Task       PingTask  `json:"task" gorm:"foreignKey:TaskId;references:Id;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;"`
	Time       LocalTime `json:"time" gorm:"index;not null"`
	Value      int       `json:"value" gorm:"type:int;not null"` // Ping 值，单位毫秒
}

type PingTask struct {
	Id       uint        `json:"id,omitempty" gorm:"primaryKey;autoIncrement"`
	Weight   int         `json:"weight" gorm:"type:int;not null;default:0;index"`
	Name     string      `json:"name" gorm:"type:varchar(255);not null;index"`
	Clients  StringArray `json:"clients" gorm:"type:longtext"`
	// Cover 决定 Clients 字段如何解释：
	//   0 = 仅包含 Clients 列表里的节点（默认，向后兼容）
	//   1 = 包含所有节点（Clients 字段被忽略；新增节点自动加入）
	//   2 = 排除 Clients 列表里的节点（其余所有节点）
	Cover    int         `json:"cover" gorm:"type:int;not null;default:0"`
	Type     string      `json:"type" gorm:"type:varchar(12);not null;default:'icmp'"` // icmp tcp http
	Target   string      `json:"target" gorm:"type:varchar(255);not null"`
	Interval int         `json:"interval" gorm:"type:int;not null;default:60"` // 间隔时间
}

// EffectiveClients 根据 Cover 字段计算任务实际作用的 client UUID 列表。
// allClientUUIDs 由调用方提供（一次性查询）以避免 N+1。
func (t *PingTask) EffectiveClients(allClientUUIDs []string) []string {
	switch t.Cover {
	case 1:
		// 全部节点
		out := make([]string, len(allClientUUIDs))
		copy(out, allClientUUIDs)
		return out
	case 2:
		// 排除 Clients 列表
		excluded := make(map[string]bool, len(t.Clients))
		for _, c := range t.Clients {
			excluded[c] = true
		}
		out := make([]string, 0, len(allClientUUIDs))
		for _, c := range allClientUUIDs {
			if !excluded[c] {
				out = append(out, c)
			}
		}
		return out
	default:
		// 仅 Clients 列表
		out := make([]string, len(t.Clients))
		copy(out, []string(t.Clients))
		return out
	}
}
