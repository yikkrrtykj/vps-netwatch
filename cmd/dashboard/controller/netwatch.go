package controller

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
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/pkg/tsdb"
	"github.com/nezhahq/nezha/service/singleton"
)

type netwatchLatencyResponse struct {
	Period      string                  `json:"period"`
	Start       int64                   `json:"start"`
	End         int64                   `json:"end"`
	GeneratedAt int64                   `json:"generated_at"`
	Servers     []netwatchLatencyServer `json:"servers"`
	Services    []netwatchLatencyService `json:"services"`
	Series      []netwatchLatencySeries `json:"series"`
	Peer        netwatchPeerState       `json:"peer"`
}

type netwatchLatencyServer struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	IP             string `json:"ip,omitempty"`
	IPv4           string `json:"ipv4,omitempty"`
	IPv6           string `json:"ipv6,omitempty"`
	CountryCode    string `json:"country_code,omitempty"`
	Platform       string `json:"platform,omitempty"`
	Online         bool   `json:"online"`
	BandwidthLabel string `json:"bandwidth_label,omitempty"`
	RemainingLabel string `json:"remaining_label,omitempty"`
	CPU            float64 `json:"cpu,omitempty"`
	MemUsed        uint64  `json:"mem_used,omitempty"`
	MemTotal       uint64  `json:"mem_total,omitempty"`
	DiskUsed       uint64  `json:"disk_used,omitempty"`
	DiskTotal      uint64  `json:"disk_total,omitempty"`
	Uptime         uint64  `json:"uptime,omitempty"`
	NetInSpeed     uint64 `json:"net_in_speed,omitempty"`
	NetOutSpeed    uint64 `json:"net_out_speed,omitempty"`
	NetInTransfer  uint64 `json:"net_in_transfer,omitempty"`
	NetOutTransfer uint64 `json:"net_out_transfer,omitempty"`
}

type netwatchLatencyService struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Target       string `json:"target"`
	Type         uint8  `json:"type"`
	TypeName     string `json:"type_name"`
	DisplayIndex int    `json:"display_index"`
	IsPeer       bool   `json:"is_peer"`
	PeerServerID uint64 `json:"peer_server_id,omitempty"`
}

type netwatchLatencySeries struct {
	Key          string            `json:"key"`
	ServiceID    uint64            `json:"service_id"`
	ServerID     uint64            `json:"server_id"`
	ServiceName  string            `json:"service_name"`
	ServerName   string            `json:"server_name"`
	Target       string            `json:"target"`
	Type         uint8             `json:"type"`
	TypeName     string            `json:"type_name"`
	DisplayIndex int               `json:"display_index"`
	IsPeer       bool              `json:"is_peer"`
	PeerServerID uint64            `json:"peer_server_id,omitempty"`
	AvgDelay     float64           `json:"avg_delay"`
	UpPercent    float32           `json:"up_percent"`
	TotalUp      uint64            `json:"total_up"`
	TotalDown    uint64            `json:"total_down"`
	DataPoints   []model.DataPoint `json:"data_points"`
}

type netwatchPeerState struct {
	Enabled        bool   `json:"enabled"`
	ServiceID      uint64 `json:"service_id,omitempty"`
	TargetServerID uint64 `json:"target_server_id,omitempty"`
	TargetName     string `json:"target_name,omitempty"`
	TargetIP       string `json:"target_ip,omitempty"`
}

type netwatchPeerTargetForm struct {
	TargetServerID uint64 `json:"target_server_id"`
}

type netwatchTargetForm struct {
	Target   string `json:"target"`
	Name     string `json:"name"`
	Duration uint64 `json:"duration"`
}

type netwatchTargetResponse struct {
	ServiceID uint64 `json:"service_id"`
	Name      string `json:"name"`
	Target    string `json:"target"`
	Type      uint8  `json:"type"`
	TypeName  string `json:"type_name"`
	Created   bool   `json:"created"`
}

type netwatchMihomoDiscoverForm struct {
	Controller string `json:"controller"`
	Secret     string `json:"secret"`
	Limit      int    `json:"limit"`
}

type netwatchMihomoDiscoverResponse struct {
	Targets []netwatchMihomoTarget `json:"targets"`
}

type netwatchMihomoTarget struct {
	Target   string `json:"target"`
	Type     uint8  `json:"type"`
	TypeName string `json:"type_name"`
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
	Network  string `json:"network,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Chain    string `json:"chain,omitempty"`
	Process  string `json:"process,omitempty"`
	Count    int    `json:"count"`
}

const (
	netwatchPeerServicePrefix = "[vps-netwatch-peer:"
	netwatchPeerServiceSuffix = "] "
)

func getNetwatchLatency(c *gin.Context) (*netwatchLatencyResponse, error) {
	periodKey := c.DefaultQuery("period", "1d")
	period, err := tsdb.ParseQueryPeriod(periodKey)
	if err != nil {
		return nil, err
	}
	start, end, err := netwatchLatencyRange(c, period)
	if err != nil {
		return nil, err
	}

	_, isMember := c.Get(model.CtxKeyAuthorizedUser)
	if !isMember && end.Sub(start) > 24*time.Hour+time.Minute {
		return nil, singleton.Localizer.ErrorT("unauthorized: only 1d data available for guests")
	}

	serverMap := singleton.ServerShared.GetList()
	visibleServers := make(map[uint64]*model.Server)
	resp := &netwatchLatencyResponse{
		Period:      periodKey,
		Start:       start.UnixMilli(),
		End:         end.UnixMilli(),
		GeneratedAt: time.Now().UnixMilli(),
		Servers:     make([]netwatchLatencyServer, 0, len(serverMap)),
		Services:    make([]netwatchLatencyService, 0),
		Series:      make([]netwatchLatencySeries, 0),
		Peer:        netwatchCurrentPeerState(serverMap),
	}

	for id, server := range serverMap {
		if server == nil || (server.HideForGuest && !isMember) {
			continue
		}
		visibleServers[id] = server
		ipv4, ipv6 := netwatchServerIPVersions(server)
		serverInfo := netwatchLatencyServer{
			ID:             id,
			Name:           server.Name,
			IP:             netwatchPeerTargetIP(server, serverMap),
			IPv4:           ipv4,
			IPv6:           ipv6,
			Online:         server.TaskStream != nil,
			BandwidthLabel: netwatchServerBandwidthLabel(server),
			RemainingLabel: netwatchServerRemainingLabel(server),
		}
		if server.GeoIP != nil {
			serverInfo.CountryCode = server.GeoIP.CountryCode
		}
		if server.Host != nil {
			serverInfo.Platform = server.Host.Platform
			serverInfo.MemTotal = server.Host.MemTotal
			serverInfo.DiskTotal = server.Host.DiskTotal
		}
		if server.State != nil {
			serverInfo.CPU = server.State.CPU
			serverInfo.MemUsed = server.State.MemUsed
			serverInfo.DiskUsed = server.State.DiskUsed
			serverInfo.Uptime = server.State.Uptime
			serverInfo.NetInSpeed = server.State.NetInSpeed
			serverInfo.NetOutSpeed = server.State.NetOutSpeed
			serverInfo.NetInTransfer = server.State.NetInTransfer
			serverInfo.NetOutTransfer = server.State.NetOutTransfer
		}
		resp.Servers = append(resp.Servers, serverInfo)
	}
	sort.Slice(resp.Servers, func(i, j int) bool { return resp.Servers[i].Name < resp.Servers[j].Name })
	if netwatchTruthy(c.Query("servers_only")) {
		return resp, nil
	}

	for _, service := range singleton.ServiceSentinelShared.GetSortedList() {
		if service == nil || !netwatchIsLatencyService(service.Type) {
			continue
		}
		peerServerID := netwatchPeerServerID(service, serverMap)
		if !netwatchShouldExposeLatencyService(service, peerServerID) {
			continue
		}
		serviceName := netwatchDisplayServiceName(service, serverMap)

		serviceInfo := netwatchLatencyService{
			ID:           service.ID,
			Name:         serviceName,
			Target:       service.Target,
			Type:         service.Type,
			TypeName:     netwatchServiceTypeName(service.Type),
			DisplayIndex: service.DisplayIndex,
			IsPeer:       peerServerID > 0,
			PeerServerID: peerServerID,
		}
		resp.Services = append(resp.Services, serviceInfo)

		history, err := netwatchLoadServiceHistoryRange(service, start, end)
		if err != nil {
			return nil, err
		}

		for _, serverStats := range history.Servers {
			server, ok := visibleServers[serverStats.ServerID]
			if !ok || server == nil || len(serverStats.Stats.DataPoints) == 0 {
				continue
			}
			resp.Series = append(resp.Series, netwatchLatencySeries{
				Key:          netwatchSeriesKey(service.ID, serverStats.ServerID),
				ServiceID:    service.ID,
				ServerID:     serverStats.ServerID,
				ServiceName:  serviceName,
				ServerName:   server.Name,
				Target:       service.Target,
				Type:         service.Type,
				TypeName:     netwatchServiceTypeName(service.Type),
				DisplayIndex: service.DisplayIndex,
				IsPeer:       peerServerID > 0,
				PeerServerID: peerServerID,
				AvgDelay:     serverStats.Stats.AvgDelay,
				UpPercent:    serverStats.Stats.UpPercent,
				TotalUp:      serverStats.Stats.TotalUp,
				TotalDown:    serverStats.Stats.TotalDown,
				DataPoints:   serverStats.Stats.DataPoints,
			})
		}
	}

	return resp, nil
}

func netwatchShouldExposeLatencyService(service *model.Service, peerServerID uint64) bool {
	if service == nil {
		return false
	}
	if service.Cover == model.ServiceCoverIgnoreAll && len(service.SkipServers) == 0 {
		return false
	}
	if peerServerID > 0 && service.Cover != model.ServiceCoverAll {
		return false
	}
	return true
}

func netwatchLatencyRange(c *gin.Context, period tsdb.QueryPeriod) (time.Time, time.Time, error) {
	now := time.Now()
	dateText := strings.TrimSpace(c.Query("date"))
	if dateText != "" {
		start, err := time.ParseInLocation("2006-01-02", dateText, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid date: %s", dateText)
		}
		return start, start.Add(24 * time.Hour), nil
	}

	startText := strings.TrimSpace(c.Query("start"))
	endText := strings.TrimSpace(c.Query("end"))
	if startText == "" && endText == "" {
		return now.Add(-period.Duration()), now, nil
	}
	if startText == "" || endText == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("start and end must be provided together")
	}

	start, err := netwatchParseRangeTime(startText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start: %s", startText)
	}
	end, err := netwatchParseRangeTime(endText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end: %s", endText)
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time range")
	}
	if end.Sub(start) > 31*24*time.Hour {
		return time.Time{}, time.Time{}, fmt.Errorf("time range cannot exceed 31 days")
	}
	return start, end, nil
}

func netwatchParseRangeTime(value string) (time.Time, error) {
	if millis, err := strconv.ParseInt(value, 10, 64); err == nil {
		if millis < 100000000000 {
			return time.Unix(millis, 0), nil
		}
		return time.UnixMilli(millis), nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format")
}

func netwatchNormalizeMonitorTarget(input string) (string, uint8, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return "", 0, fmt.Errorf("target is required")
	}
	if strings.ContainsAny(text, "\r\n\t ") {
		return "", 0, fmt.Errorf("target cannot contain whitespace")
	}
	if strings.Contains(text, "://") {
		parsed, err := url.Parse(text)
		if err != nil {
			return "", 0, fmt.Errorf("invalid target URL")
		}
		if parsed.Host == "" {
			return "", 0, fmt.Errorf("target URL must include host")
		}
		text = parsed.Host
	}
	text = strings.TrimSuffix(text, "/")

	host, port, hasPort, err := netwatchSplitMonitorHostPort(text)
	if err != nil {
		return "", 0, err
	}
	if hasPort {
		return netwatchJoinHostPort(host, port), model.TaskTypeTCPPing, nil
	}
	if strings.ContainsAny(text, "/?#") {
		return "", 0, fmt.Errorf("ICMP target must be a host or IP without path")
	}
	host = strings.Trim(text, "[]")
	if host == "" {
		return "", 0, fmt.Errorf("target host is required")
	}
	return host, model.TaskTypeICMPPing, nil
}

func netwatchSplitMonitorHostPort(text string) (string, string, bool, error) {
	host, port, err := net.SplitHostPort(text)
	if err == nil {
		if err := netwatchValidatePort(port); err != nil {
			return "", "", false, err
		}
		host = strings.Trim(host, "[]")
		if host == "" {
			return "", "", false, fmt.Errorf("target host is required")
		}
		return host, port, true, nil
	}
	if strings.Count(text, ":") == 1 {
		parts := strings.SplitN(text, ":", 2)
		if parts[0] != "" && netwatchValidatePort(parts[1]) == nil {
			return parts[0], parts[1], true, nil
		}
		if parts[0] != "" && parts[1] != "" {
			return "", "", false, fmt.Errorf("invalid TCP port")
		}
	}
	if strings.HasPrefix(text, "[") && strings.Contains(text, "]") {
		return "", "", false, fmt.Errorf("invalid host:port target")
	}
	return "", "", false, nil
}

func netwatchValidatePort(port string) error {
	n, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("invalid TCP port")
	}
	return nil
}

func netwatchJoinHostPort(host, port string) string {
	host = strings.Trim(host, "[]")
	if strings.Contains(host, ":") {
		return net.JoinHostPort(host, port)
	}
	return host + ":" + port
}

func netwatchDefaultTargetName(target string, taskType uint8) string {
	switch taskType {
	case model.TaskTypeTCPPing:
		return "TCP " + target
	default:
		return "Ping " + target
	}
}

func netwatchFindExistingTarget(taskType uint8, target string) *model.Service {
	for _, service := range singleton.ServiceSentinelShared.GetSortedList() {
		if service == nil || strings.HasPrefix(service.Name, netwatchPeerServicePrefix) {
			continue
		}
		if service.Type == taskType && strings.EqualFold(strings.TrimSpace(service.Target), target) {
			return service
		}
	}
	return nil
}

func netwatchNormalizeControllerURL(input string) (string, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return "", fmt.Errorf("mihomo controller URL is required")
	}
	if !strings.Contains(text, "://") {
		text = "http://" + text
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid mihomo controller URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("mihomo controller URL must use http or https")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func netwatchMihomoConnectionTarget(conn map[string]any) (netwatchMihomoTarget, bool) {
	metadata, _ := conn["metadata"].(map[string]any)
	if metadata == nil {
		return netwatchMihomoTarget{}, false
	}
	network := strings.ToLower(netwatchAnyString(metadata["network"]))
	host := netwatchAnyString(metadata["destinationIP"])
	if host == "" {
		host = netwatchAnyString(metadata["host"])
	}
	port := netwatchAnyString(metadata["destinationPort"])
	if host == "" {
		return netwatchMihomoTarget{}, false
	}
	target := host
	taskType := uint8(model.TaskTypeICMPPing)
	if network == "tcp" && port != "" && netwatchValidatePort(port) == nil {
		target = netwatchJoinHostPort(host, port)
		taskType = model.TaskTypeTCPPing
	}
	chains := netwatchAnyStringSlice(conn["chains"])
	return netwatchMihomoTarget{
		Target:   target,
		Type:     taskType,
		TypeName: netwatchServiceTypeName(taskType),
		Host:     host,
		Port:     port,
		Network:  network,
		Rule:     netwatchAnyString(conn["rule"]),
		Chain:    strings.Join(chains, " / "),
		Process:  netwatchAnyString(metadata["process"]),
	}, true
}

func netwatchAnyString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	default:
		return ""
	}
}

func netwatchAnyStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text := netwatchAnyString(item); text != "" {
			result = append(result, text)
		}
	}
	return result
}

func updateNetwatchPeerTarget(c *gin.Context) (*netwatchPeerState, error) {
	var form netwatchPeerTargetForm
	if err := c.ShouldBindJSON(&form); err != nil {
		return nil, err
	}

	serverMap := singleton.ServerShared.GetList()
	autoServices := netwatchAutoPeerServices()

	if form.TargetServerID == 0 {
		for _, service := range autoServices {
			netwatchDisablePeerService(service)
			if err := netwatchSavePeerService(service); err != nil {
				return nil, err
			}
		}
		return &netwatchPeerState{}, nil
	}

	targetServer, ok := serverMap[form.TargetServerID]
	if !ok || targetServer == nil {
		return nil, singleton.Localizer.ErrorT("server id %d does not exist", form.TargetServerID)
	}
	if !targetServer.HasPermission(c) {
		return nil, singleton.Localizer.ErrorT("permission denied")
	}

	targetIP := netwatchPeerTargetIP(targetServer, serverMap)
	if targetIP == "" {
		return nil, singleton.Localizer.ErrorT("server %s has no public IP yet", targetServer.Name)
	}

	var selected *model.Service
	for _, service := range autoServices {
		peerID := netwatchAutoPeerServerID(service)
		if peerID == form.TargetServerID {
			selected = service
			continue
		}
		netwatchDisablePeerService(service)
		if err := netwatchSavePeerService(service); err != nil {
			return nil, err
		}
	}

	if selected == nil {
		selected = &model.Service{
			Common: model.Common{UserID: getUid(c)},
		}
	}

	selected.Name = netwatchPeerServiceName(targetServer.ID, targetServer.Name)
	selected.Target = targetIP
	selected.Type = model.TaskTypeICMPPing
	selected.SkipServers = map[uint64]bool{targetServer.ID: true}
	selected.Cover = model.ServiceCoverAll
	selected.DisplayIndex = 0
	selected.Notify = false
	selected.NotificationGroupID = 0
	selected.Duration = 30
	selected.LatencyNotify = false
	selected.MinLatency = 0
	selected.MaxLatency = 0
	selected.EnableShowInService = false
	selected.EnableTriggerTask = false
	selected.RecoverTriggerTasks = nil
	selected.FailTriggerTasks = nil

	if err := netwatchSavePeerService(selected); err != nil {
		return nil, err
	}

	return &netwatchPeerState{
		Enabled:        true,
		ServiceID:      selected.ID,
		TargetServerID: targetServer.ID,
		TargetName:     targetServer.Name,
		TargetIP:       targetIP,
	}, nil
}

func createNetwatchTarget(c *gin.Context) (*netwatchTargetResponse, error) {
	var form netwatchTargetForm
	if err := c.ShouldBindJSON(&form); err != nil {
		return nil, err
	}

	target, taskType, err := netwatchNormalizeMonitorTarget(form.Target)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(form.Name)
	if name == "" {
		name = netwatchDefaultTargetName(target, taskType)
	}
	duration := form.Duration
	if duration == 0 {
		duration = 30
	}
	if duration < 5 {
		duration = 5
	}

	if existing := netwatchFindExistingTarget(taskType, target); existing != nil {
		changed := false
		if existing.Cover == model.ServiceCoverIgnoreAll && len(existing.SkipServers) == 0 {
			existing.Cover = model.ServiceCoverAll
			changed = true
		}
		if existing.Duration == 0 {
			existing.Duration = duration
			changed = true
		}
		if changed {
			if err := netwatchSaveService(existing); err != nil {
				return nil, err
			}
		}
		return &netwatchTargetResponse{
			ServiceID: existing.ID,
			Name:      existing.Name,
			Target:    existing.Target,
			Type:      existing.Type,
			TypeName:  netwatchServiceTypeName(existing.Type),
			Created:   false,
		}, nil
	}

	service := &model.Service{
		Common:              model.Common{UserID: getUid(c)},
		Name:                name,
		Target:              target,
		Type:                taskType,
		SkipServers:         map[uint64]bool{},
		Cover:               model.ServiceCoverAll,
		DisplayIndex:        0,
		Notify:              false,
		NotificationGroupID: 0,
		Duration:            duration,
		LatencyNotify:       false,
		MinLatency:          0,
		MaxLatency:          0,
		EnableShowInService: true,
		EnableTriggerTask:   false,
		RecoverTriggerTasks: []uint64{},
		FailTriggerTasks:    []uint64{},
	}

	if err := netwatchSaveService(service); err != nil {
		return nil, err
	}

	return &netwatchTargetResponse{
		ServiceID: service.ID,
		Name:      service.Name,
		Target:    service.Target,
		Type:      service.Type,
		TypeName:  netwatchServiceTypeName(service.Type),
		Created:   true,
	}, nil
}

func discoverNetwatchMihomoTargets(c *gin.Context) (*netwatchMihomoDiscoverResponse, error) {
	var form netwatchMihomoDiscoverForm
	if err := c.ShouldBindJSON(&form); err != nil {
		return nil, err
	}
	controller, err := netwatchNormalizeControllerURL(form.Controller)
	if err != nil {
		return nil, err
	}
	limit := form.Limit
	if limit <= 0 || limit > 100 {
		limit = 60
	}

	req, err := http.NewRequest(http.MethodGet, controller+"/connections", nil)
	if err != nil {
		return nil, err
	}
	secret := strings.TrimSpace(form.Secret)
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	client := http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mihomo controller returned %s", resp.Status)
	}

	var payload struct {
		Connections []map[string]any `json:"connections"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, err
	}

	targets := make([]netwatchMihomoTarget, 0)
	seen := make(map[string]int)
	for _, conn := range payload.Connections {
		target, ok := netwatchMihomoConnectionTarget(conn)
		if !ok {
			continue
		}
		key := strconvFormatUint(uint64(target.Type)) + "|" + strings.ToLower(target.Target)
		if idx, ok := seen[key]; ok {
			targets[idx].Count++
			continue
		}
		target.Count = 1
		seen[key] = len(targets)
		targets = append(targets, target)
		if len(targets) >= limit {
			break
		}
	}

	return &netwatchMihomoDiscoverResponse{Targets: targets}, nil
}

func netwatchLoadServiceHistoryRange(service *model.Service, start, end time.Time) (*model.ServiceHistoryResponse, error) {
	response := &model.ServiceHistoryResponse{
		ServiceID:   service.ID,
		ServiceName: service.Name,
		Servers:     make([]model.ServerServiceStats, 0),
	}

	if !singleton.TSDBEnabled() {
		return queryServiceHistoryFromDBRange(service.ID, start, end, response)
	}

	result, err := singleton.TSDBShared.QueryServiceHistoryRange(service.ID, start, end)
	if err != nil {
		return nil, err
	}

	serverMap := singleton.ServerShared.GetList()
	for i := range result.Servers {
		if server, ok := serverMap[result.Servers[i].ServerID]; ok && server != nil {
			result.Servers[i].ServerName = server.Name
		}
	}
	response.Servers = result.Servers

	return response, nil
}

func netwatchIsLatencyService(t uint8) bool {
	return t == model.TaskTypeICMPPing || t == model.TaskTypeTCPPing
}

func netwatchCurrentPeerState(serverMap map[uint64]*model.Server) netwatchPeerState {
	for _, service := range singleton.ServiceSentinelShared.GetSortedList() {
		if service == nil || !strings.HasPrefix(service.Name, netwatchPeerServicePrefix) {
			continue
		}
		peerID := netwatchAutoPeerServerID(service)
		if peerID == 0 || service.Cover != model.ServiceCoverAll {
			continue
		}
		server := serverMap[peerID]
		if server == nil {
			continue
		}
		return netwatchPeerState{
			Enabled:        true,
			ServiceID:      service.ID,
			TargetServerID: peerID,
			TargetName:     server.Name,
			TargetIP:       service.Target,
		}
	}
	return netwatchPeerState{}
}

func netwatchAutoPeerServices() []*model.Service {
	var services []*model.Service
	for _, service := range singleton.ServiceSentinelShared.GetSortedList() {
		if service != nil && strings.HasPrefix(service.Name, netwatchPeerServicePrefix) {
			services = append(services, service)
		}
	}
	return services
}

func netwatchSavePeerService(service *model.Service) error {
	return netwatchSaveService(service)
}

func netwatchSaveService(service *model.Service) error {
	if service.ID == 0 {
		if err := singleton.DB.Create(service).Error; err != nil {
			return newGormError("%v", err)
		}
	} else if err := singleton.DB.Save(service).Error; err != nil {
		return newGormError("%v", err)
	}
	if err := singleton.ServiceSentinelShared.Update(service); err != nil {
		return err
	}
	singleton.ServiceSentinelShared.UpdateServiceList()
	return nil
}

func netwatchDisablePeerService(service *model.Service) {
	service.Cover = model.ServiceCoverIgnoreAll
	service.SkipServers = map[uint64]bool{}
	service.Notify = false
	service.EnableShowInService = false
	service.EnableTriggerTask = false
}

func netwatchPeerServiceName(serverID uint64, serverName string) string {
	return fmt.Sprintf("%s%d%sVPS %s", netwatchPeerServicePrefix, serverID, netwatchPeerServiceSuffix, serverName)
}

func netwatchServerIP(server *model.Server) string {
	if server == nil || server.GeoIP == nil {
		return ""
	}
	if server.GeoIP.IP.IPv4Addr != "" {
		return server.GeoIP.IP.IPv4Addr
	}
	return server.GeoIP.IP.IPv6Addr
}

func netwatchServerBandwidthLabel(server *model.Server) string {
	if server == nil {
		return ""
	}
	for _, text := range []string{server.PublicNote, server.Name} {
		if label := netwatchServerBandwidthFromText(text); label != "" {
			return label
		}
	}
	return ""
}

func netwatchServerRemainingLabel(server *model.Server) string {
	if server == nil {
		return ""
	}
	now := time.Now()
	for _, text := range []string{server.PublicNote, server.Name} {
		if label := netwatchServerRemainingFromText(text, now); label != "" {
			return label
		}
	}
	return ""
}

func netwatchServerBandwidthFromText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lowerText := strings.ToLower(text)
	for _, marker := range []string{"bandwidth=", "bandwidth:", "带宽=", "带宽:"} {
		idx := strings.Index(lowerText, strings.ToLower(marker))
		if idx < 0 {
			continue
		}
		return netwatchCleanBandwidthLabel(text[idx+len(marker):], false)
	}
	if idx := strings.LastIndex(text, "@"); idx >= 0 {
		return netwatchCleanBandwidthLabel(text[idx+1:], true)
	}
	return ""
}

func netwatchCleanBandwidthLabel(text string, stopAtSpace bool) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	end := strings.IndexFunc(text, func(r rune) bool {
		if r == ',' || r == '，' || r == ';' || r == '；' || r == '|' || r == '/' || r == '[' || r == ']' || r == '(' || r == ')' || r == '（' || r == '）' {
			return true
		}
		return stopAtSpace && unicode.IsSpace(r)
	})
	if end >= 0 {
		text = text[:end]
	}
	text = strings.TrimSpace(text)
	if len([]rune(text)) > 32 {
		return ""
	}
	return text
}

func netwatchServerRemainingFromText(text string, now time.Time) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	type marker struct {
		text       string
		isDuration bool
	}
	markers := []marker{
		{text: "remaining=", isDuration: true},
		{text: "remaining:", isDuration: true},
		{text: "remain=", isDuration: true},
		{text: "remain:", isDuration: true},
		{text: "days=", isDuration: true},
		{text: "days:", isDuration: true},
		{text: "剩余=", isDuration: true},
		{text: "剩余:", isDuration: true},
		{text: "expire="},
		{text: "expire:"},
		{text: "expires="},
		{text: "expires:"},
		{text: "expiry="},
		{text: "expiry:"},
		{text: "到期="},
		{text: "到期:"},
		{text: "有效期="},
		{text: "有效期:"},
	}
	lowerText := strings.ToLower(text)
	for _, marker := range markers {
		idx := strings.Index(lowerText, strings.ToLower(marker.text))
		if idx < 0 {
			continue
		}
		token := netwatchCleanMetadataToken(text[idx+len(marker.text):])
		if token == "" {
			continue
		}
		if label := netwatchRemainingTokenLabel(token, marker.isDuration, now); label != "" {
			return label
		}
	}
	return ""
}

func netwatchCleanMetadataToken(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	end := strings.IndexFunc(text, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '|' || r == '[' || r == ']' || r == '(' || r == ')' || r == '（' || r == '）' || unicode.IsSpace(r)
	})
	if end >= 0 {
		text = text[:end]
	}
	return strings.TrimSpace(text)
}

func netwatchRemainingTokenLabel(token string, durationToken bool, now time.Time) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	lowerToken := strings.ToLower(token)
	if strings.Contains(lowerToken, "永久") || strings.Contains(lowerToken, "permanent") || strings.Contains(lowerToken, "forever") {
		return "永久"
	}
	if days, ok := netwatchRemainingDaysFromToken(lowerToken, durationToken); ok {
		return netwatchFormatRemainingDays(days)
	}
	normalized := strings.NewReplacer("/", "-", ".", "-").Replace(token)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
		"2006-1-2",
	} {
		expireAt, err := time.ParseInLocation(layout, normalized, time.Local)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" || layout == "2006-1-2" {
			expireAt = expireAt.Add(24*time.Hour - time.Nanosecond)
		}
		remaining := expireAt.Sub(now)
		if remaining < 0 {
			return "已到期"
		}
		days := int(remaining / (24 * time.Hour))
		if remaining%(24*time.Hour) > 0 {
			days++
		}
		return netwatchFormatRemainingDays(days)
	}
	return ""
}

func netwatchRemainingDaysFromToken(token string, durationToken bool) (int, bool) {
	n, ok := netwatchLeadingInt(token)
	if !ok {
		return 0, false
	}
	if durationToken || strings.Contains(token, "天") || strings.Contains(token, "day") || strings.HasSuffix(token, "d") {
		return n, true
	}
	return 0, false
}

func netwatchLeadingInt(text string) (int, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	end := 0
	for _, r := range text {
		if !unicode.IsDigit(r) {
			break
		}
		end += len(string(r))
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(text[:end])
	if err != nil {
		return 0, false
	}
	return n, true
}

func netwatchFormatRemainingDays(days int) string {
	if days < 0 {
		return "已到期"
	}
	if days == 0 {
		return "今天到期"
	}
	return fmt.Sprintf("余 %d 天", days)
}

func netwatchPeerTargetIP(server *model.Server, serverMap map[uint64]*model.Server) string {
	if ip := netwatchServerIP(server); ip != "" {
		return ip
	}
	if server == nil {
		return ""
	}
	for _, service := range singleton.ServiceSentinelShared.GetSortedList() {
		if service == nil || service.Target == "" {
			continue
		}
		if netwatchAutoPeerServerID(service) == server.ID {
			return service.Target
		}
		if strings.HasPrefix(service.Name, "VPS ") && strings.TrimSpace(strings.TrimPrefix(service.Name, "VPS ")) == server.Name {
			return service.Target
		}
		if netwatchPeerServerID(service, serverMap) == server.ID {
			return service.Target
		}
	}
	return ""
}

func netwatchServerIPVersions(server *model.Server) (string, string) {
	if server == nil || server.GeoIP == nil {
		return "", ""
	}
	return server.GeoIP.IP.IPv4Addr, server.GeoIP.IP.IPv6Addr
}

func netwatchTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func netwatchDisplayServiceName(service *model.Service, serverMap map[uint64]*model.Server) string {
	peerID := netwatchPeerServerID(service, serverMap)
	if peerID > 0 {
		if server := serverMap[peerID]; server != nil {
			return "VPS " + server.Name
		}
	}
	if strings.HasPrefix(service.Name, netwatchPeerServicePrefix) {
		if _, rest, ok := strings.Cut(service.Name, netwatchPeerServiceSuffix); ok {
			return rest
		}
	}
	return service.Name
}

func netwatchPeerServerID(service *model.Service, serverMap map[uint64]*model.Server) uint64 {
	if service == nil {
		return 0
	}
	if id := netwatchAutoPeerServerID(service); id > 0 {
		return id
	}
	if strings.HasPrefix(service.Name, "VPS ") {
		name := strings.TrimSpace(strings.TrimPrefix(service.Name, "VPS "))
		for id, server := range serverMap {
			if server == nil {
				continue
			}
			if server.Name == name || (service.Target != "" && service.Target == netwatchServerIP(server)) {
				return id
			}
		}
	}
	return 0
}

func netwatchAutoPeerServerID(service *model.Service) uint64 {
	if service == nil || !strings.HasPrefix(service.Name, netwatchPeerServicePrefix) {
		return 0
	}
	rest := strings.TrimPrefix(service.Name, netwatchPeerServicePrefix)
	idText, _, ok := strings.Cut(rest, netwatchPeerServiceSuffix)
	if !ok {
		return 0
	}
	id, err := strconv.ParseUint(idText, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func netwatchServiceTypeName(t uint8) string {
	switch t {
	case model.TaskTypeICMPPing:
		return "ICMP"
	case model.TaskTypeTCPPing:
		return "TCP"
	default:
		return "OTHER"
	}
}

func netwatchSeriesKey(serviceID, serverID uint64) string {
	return strconvFormatUint(serverID) + ":" + strconvFormatUint(serviceID)
}

func strconvFormatUint(v uint64) string {
	return strconv.FormatUint(v, 10)
}
