package controller

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/pkg/tsdb"
	"github.com/nezhahq/nezha/service/singleton"
)

type netwatchLatencyResponse struct {
	Period      string                  `json:"period"`
	GeneratedAt int64                   `json:"generated_at"`
	Servers     []netwatchLatencyServer `json:"servers"`
	Services    []netwatchLatencyService `json:"services"`
	Series      []netwatchLatencySeries `json:"series"`
	Peer        netwatchPeerState       `json:"peer"`
}

type netwatchLatencyServer struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip,omitempty"`
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

	_, isMember := c.Get(model.CtxKeyAuthorizedUser)
	if !isMember && period != tsdb.Period1Day {
		return nil, singleton.Localizer.ErrorT("unauthorized: only 1d data available for guests")
	}

	serverMap := singleton.ServerShared.GetList()
	visibleServers := make(map[uint64]*model.Server)
	resp := &netwatchLatencyResponse{
		Period:      periodKey,
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
		resp.Servers = append(resp.Servers, netwatchLatencyServer{ID: id, Name: server.Name, IP: netwatchPeerTargetIP(server, serverMap)})
	}
	sort.Slice(resp.Servers, func(i, j int) bool { return resp.Servers[i].Name < resp.Servers[j].Name })

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

		history, err := netwatchLoadServiceHistory(service, period)
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

func netwatchLoadServiceHistory(service *model.Service, period tsdb.QueryPeriod) (*model.ServiceHistoryResponse, error) {
	response := &model.ServiceHistoryResponse{
		ServiceID:   service.ID,
		ServiceName: service.Name,
		Servers:     make([]model.ServerServiceStats, 0),
	}

	if !singleton.TSDBEnabled() {
		return queryServiceHistoryFromDB(service.ID, period, response)
	}

	result, err := singleton.TSDBShared.QueryServiceHistory(service.ID, period)
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
