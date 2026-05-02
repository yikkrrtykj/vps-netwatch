package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/komari-monitor/komari/api"
	"github.com/komari-monitor/komari/database/clients"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/database/tasks"
)

type probeTargetRequest struct {
	Target   string   `json:"target"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Interval int      `json:"interval"`
	Clients  []string `json:"clients"`
	// Cover 决定 Clients 字段如何解释：0=仅指定, 1=全部, 2=排除指定
	Cover int `json:"cover"`
}

type clashDiscoverRequest struct {
	Controller string `json:"controller"`
	Secret     string `json:"secret"`
	Limit      int    `json:"limit"`
}

type clashTarget struct {
	Target  string   `json:"target"`
	Type    string   `json:"type"`
	Host    string   `json:"host"`
	Port    int      `json:"port,omitempty"`
	Network string   `json:"network,omitempty"`
	Rule    string   `json:"rule,omitempty"`
	Process string   `json:"process,omitempty"`
	Chains  []string `json:"chains,omitempty"`
	Count   int      `json:"count"`
}

// AddProbeTarget 接收前端 ProbeManual 提交的探针目标，自动识别协议（icmp/tcp/http），
// 合并到现有 ping 任务（如果同 type+target 已存在则只补充 clients），否则新建。
func AddProbeTarget(c *gin.Context) {
	var req probeTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	target, detectedType, err := normalizeProbeTarget(req.Target)
	if err != nil {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	taskType := detectedType
	if strings.TrimSpace(req.Type) != "" {
		taskType = strings.ToLower(strings.TrimSpace(req.Type))
		if !validPingTaskType(taskType) {
			api.RespondError(c, http.StatusBadRequest, "type must be icmp, tcp, or http")
			return
		}
	}

	interval := req.Interval
	if interval <= 0 {
		interval = 60
	}
	if interval < 5 {
		interval = 5
	}

	clientIDs, err := normalizeProbeClients(req.Clients, req.Cover)
	if err != nil {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(clientIDs) == 0 && req.Cover == 0 {
		api.RespondError(c, http.StatusBadRequest, "no clients available")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("%s %s", strings.ToUpper(taskType), target)
	}

	existingTasks, err := tasks.GetAllPingTasks()
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	for i := range existingTasks {
		task := &existingTasks[i]
		if !strings.EqualFold(task.Type, taskType) || task.Target != target {
			continue
		}
		// 同 target+type 已存在 → 合并 clients；如果新请求给了非默认 Cover，覆盖；
		// 如果给了非空 Name，也覆盖（用户明确传名 = 用户期望改名）
		mergedClients := mergeStringLists([]string(task.Clients), clientIDs)
		needSave := len(mergedClients) != len(task.Clients)
		if req.Cover != task.Cover {
			task.Cover = req.Cover
			needSave = true
		}
		newName := strings.TrimSpace(req.Name)
		if newName != "" && newName != task.Name {
			task.Name = newName
			needSave = true
		}
		if needSave {
			task.Clients = models.StringArray(mergedClients)
			if err := tasks.EditPingTask([]*models.PingTask{task}); err != nil {
				api.RespondError(c, http.StatusInternalServerError, err.Error())
				return
			}
		}
		api.RespondSuccess(c, gin.H{
			"task_id": task.Id,
			"created": false,
			"type":    task.Type,
			"target":  task.Target,
			"clients": mergedClients,
			"cover":   task.Cover,
			"name":    task.Name,
		})
		return
	}

	taskID, err := tasks.AddPingTask(clientIDs, name, target, taskType, interval, req.Cover)
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	api.RespondSuccess(c, gin.H{
		"task_id": taskID,
		"created": true,
		"type":    taskType,
		"target":  target,
		"clients": clientIDs,
		"cover":   req.Cover,
	})
}

// DiscoverClashTargets 从 Clash/mihomo controller 拉取活跃连接，去重 + 按出现频率排序，
// 返回给前端 ClashHelper 让用户多选添加为监控目标。
func DiscoverClashTargets(c *gin.Context) {
	var req clashDiscoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	controller := strings.TrimSpace(req.Controller)
	if controller == "" {
		controller = "http://127.0.0.1:9090"
	}
	if !strings.Contains(controller, "://") {
		controller = "http://" + controller
	}
	baseURL, err := url.Parse(controller)
	if err != nil || baseURL.Host == "" {
		api.RespondError(c, http.StatusBadRequest, "invalid clash controller address")
		return
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/connections"
	baseURL.RawQuery = ""

	httpReq, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if secret := strings.TrimSpace(req.Secret); secret != "" {
		httpReq.Header.Set("Authorization", "Bearer "+secret)
	}

	httpClient := &http.Client{Timeout: 6 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		api.RespondError(c, http.StatusBadGateway, "failed to query clash: "+err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		api.RespondError(c, http.StatusBadGateway, "failed to read clash response: "+err.Error())
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		api.RespondError(c, http.StatusBadGateway, fmt.Sprintf("clash returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
		return
	}

	var payload struct {
		Connections []struct {
			Metadata map[string]any `json:"metadata"`
			Chains   []string       `json:"chains"`
			Rule     string         `json:"rule"`
			Process  string         `json:"process"`
		} `json:"connections"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		api.RespondError(c, http.StatusBadGateway, "invalid clash response: "+err.Error())
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 80
	}

	discovered := make(map[string]*clashTarget)
	for _, conn := range payload.Connections {
		host := firstNonEmptyString(
			anyString(conn.Metadata["destinationIP"]),
			anyString(conn.Metadata["host"]),
			anyString(conn.Metadata["remoteDestination"]),
			anyString(conn.Metadata["destinationHost"]),
		)
		host = strings.TrimSpace(strings.Trim(host, "[]"))
		if host == "" {
			continue
		}

		port := anyInt(conn.Metadata["destinationPort"])
		network := strings.ToLower(firstNonEmptyString(anyString(conn.Metadata["network"]), anyString(conn.Metadata["type"])))
		targetType := "icmp"
		target := host
		if port > 0 && port <= 65535 && network != "udp" {
			targetType = "tcp"
			target = net.JoinHostPort(host, strconv.Itoa(port))
		}

		key := targetType + "|" + target
		if item, ok := discovered[key]; ok {
			item.Count++
			continue
		}
		discovered[key] = &clashTarget{
			Target:  target,
			Type:    targetType,
			Host:    host,
			Port:    port,
			Network: network,
			Rule:    conn.Rule,
			Process: conn.Process,
			Chains:  conn.Chains,
			Count:   1,
		}
	}

	targets := make([]clashTarget, 0, len(discovered))
	for _, item := range discovered {
		targets = append(targets, *item)
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Count == targets[j].Count {
			return targets[i].Target < targets[j].Target
		}
		return targets[i].Count > targets[j].Count
	})
	if len(targets) > limit {
		targets = targets[:limit]
	}

	api.RespondSuccess(c, gin.H{"targets": targets})
}

func normalizeProbeTarget(raw string) (string, string, error) {
	input := strings.TrimSpace(raw)
	if input == "" {
		return "", "", fmt.Errorf("target is required")
	}
	if strings.ContainsAny(input, " \t\r\n") {
		return "", "", fmt.Errorf("target cannot contain spaces")
	}

	if strings.Contains(input, "://") {
		parsed, err := url.Parse(input)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return "", "", fmt.Errorf("invalid URL target")
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme == "http" || scheme == "https" {
			return input, "http", nil
		}
		input = parsed.Host
	}

	if host, port, ok, err := splitProbeHostPort(input); err != nil {
		return "", "", err
	} else if ok {
		portNum, err := strconv.Atoi(port)
		if err != nil || portNum <= 0 || portNum > 65535 {
			return "", "", fmt.Errorf("invalid target port")
		}
		return net.JoinHostPort(host, port), "tcp", nil
	}

	return strings.Trim(input, "[]"), "icmp", nil
}

func splitProbeHostPort(input string) (string, string, bool, error) {
	if strings.HasPrefix(input, "[") {
		host, port, err := net.SplitHostPort(input)
		if err != nil {
			return "", "", false, fmt.Errorf("invalid host:port target")
		}
		return strings.Trim(host, "[]"), port, true, nil
	}
	if strings.Count(input, ":") == 1 {
		parts := strings.SplitN(input, ":", 2)
		if parts[0] == "" || parts[1] == "" {
			return "", "", false, fmt.Errorf("invalid host:port target")
		}
		return parts[0], parts[1], true, nil
	}
	if strings.Count(input, ":") > 1 {
		if ip := net.ParseIP(input); ip != nil {
			return "", "", false, nil
		}
		host, port, err := net.SplitHostPort(input)
		if err != nil {
			return "", "", false, nil
		}
		return strings.Trim(host, "[]"), port, true, nil
	}
	return "", "", false, nil
}

func normalizeProbeClients(raw []string, cover int) ([]string, error) {
	// Cover=1（全部）时 clients 可以为空——调度时会自动取所有节点
	// Cover=2（排除）时 clients 是排除列表，可以为空（等价于 Cover=1）
	if cover == 1 {
		// 不需要存 clients 列表
		return mergeStringLists(nil, raw), nil
	}
	if len(raw) > 0 {
		return mergeStringLists(nil, raw), nil
	}

	// Cover=0 且未指定 clients → 默认全部节点
	allClients, err := clients.GetAllClientBasicInfo()
	if err != nil {
		return nil, err
	}
	clientIDs := make([]string, 0, len(allClients))
	for _, client := range allClients {
		clientIDs = append(clientIDs, client.UUID)
	}
	return mergeStringLists(nil, clientIDs), nil
}

func mergeStringLists(base []string, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	merged := make([]string, 0, len(base)+len(extra))
	for _, list := range [][]string{base, extra} {
		for _, item := range list {
			value := strings.TrimSpace(item)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			merged = append(merged, value)
		}
	}
	return merged
}

func validPingTaskType(taskType string) bool {
	return taskType == "icmp" || taskType == "tcp" || taskType == "http"
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func anyString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func anyInt(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	case json.Number:
		n, _ := strconv.Atoi(v.String())
		return n
	default:
		return 0
	}
}
