package controller

import (
	"fmt"
	"net/http"
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

func netwatchLatencyPage(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(netwatchLatencyHTMLV2))
}

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

const netwatchLatencyHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>vps-netwatch 延迟总览</title>
<style>
:root { color-scheme: light; --bg:#f6f7f9; --panel:#ffffff; --line:#d9dee8; --text:#1d2433; --muted:#687386; --brand:#2367d8; --green:#16a06b; --red:#e04f5f; --shadow:0 8px 24px rgba(31,41,55,.08); }
* { box-sizing: border-box; }
body { margin:0; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Arial,"Noto Sans SC",sans-serif; background:var(--bg); color:var(--text); }
.main { width:min(1180px, calc(100vw - 32px)); margin:24px auto; }
.nav { display:flex; gap:8px; align-items:center; padding:12px 16px; border:1px solid var(--line); border-radius:8px; background:var(--panel); box-shadow:var(--shadow); overflow:auto; }
.nav span { flex:0 0 auto; color:#4b5565; padding:8px 10px; border-radius:6px; font-size:14px; }
.nav .active { color:var(--brand); background:#eaf1ff; font-weight:700; }
.toolbar { display:flex; flex-wrap:wrap; gap:10px; align-items:center; margin:14px 0; }
.group { display:flex; align-items:center; gap:6px; min-height:38px; padding:5px; border:1px solid var(--line); border-radius:8px; background:var(--panel); box-shadow:0 4px 12px rgba(31,41,55,.05); }
.group label { padding:0 8px; color:var(--muted); font-size:13px; }
button { border:0; border-radius:6px; padding:7px 10px; background:transparent; color:var(--text); cursor:pointer; font:inherit; white-space:nowrap; }
button.active { background:#111827; color:#fff; }
button.ghost { color:var(--brand); }
.summary { display:grid; grid-template-columns:repeat(4, minmax(0,1fr)); gap:12px; margin:12px 0; }
.metric { min-height:74px; padding:14px 16px; border:1px solid var(--line); border-radius:8px; background:var(--panel); box-shadow:var(--shadow); }
.metric b { display:block; font-size:24px; line-height:1.1; margin-top:6px; }
.metric span { color:var(--muted); font-size:13px; }
.filters { display:grid; gap:10px; margin:12px 0; }
.filter-row { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
.filter-title { width:64px; color:var(--muted); font-size:13px; }
.chip { display:inline-flex; align-items:center; gap:6px; border:1px solid var(--line); background:var(--panel); color:#2f3747; }
.chip.active { border-color:#a7c3ff; background:#eef5ff; color:#123f91; }
.dot { width:9px; height:9px; border-radius:50%; display:inline-block; background:currentColor; }
.panel { border:1px solid var(--line); border-radius:8px; background:var(--panel); box-shadow:var(--shadow); overflow:hidden; }
.panel-head { display:flex; justify-content:space-between; gap:12px; align-items:center; padding:14px 16px; border-bottom:1px solid var(--line); }
.panel-title { font-size:16px; font-weight:700; }
.panel-subtitle { color:var(--muted); font-size:13px; }
.chart-wrap { position:relative; height:560px; padding:8px 12px 16px; }
canvas { width:100%; height:100%; display:block; }
.legend { display:flex; flex-wrap:wrap; gap:8px 14px; padding:0 16px 16px; color:#334155; font-size:13px; }
.legend-item { display:inline-flex; align-items:center; gap:6px; }
.empty { position:absolute; inset:0; display:none; align-items:center; justify-content:center; color:var(--muted); text-align:center; padding:24px; }
.error { display:none; margin:12px 0; padding:12px 14px; border:1px solid #f1b8bf; border-radius:8px; color:#8f1d2c; background:#fff1f3; }
@media (max-width: 760px) { .main { width:calc(100vw - 20px); margin:10px auto; } .summary { grid-template-columns:repeat(2, minmax(0,1fr)); } .chart-wrap { height:420px; } .filter-title { width:100%; } }
</style>
</head>
<body>
<main class="main">
  <nav class="nav" aria-label="netwatch views">
    <span>摘要</span><span>硬件</span><span>速率</span><span>IP 质量</span><span class="active">延迟</span><span>Ping</span><span>路由</span><span>BGP</span><span>状态</span>
  </nav>

  <section class="toolbar">
    <div class="group" id="periodGroup"><label>时间</label><button class="active" data-period="1d">1 天</button><button data-period="7d">7 天</button><button data-period="30d">30 天</button></div>
    <div class="group" id="protocolGroup"><label>协议</label><button class="active" data-protocol="all">全部</button><button data-protocol="ICMP">ICMP</button><button data-protocol="TCP">TCP</button></div>
    <button class="group ghost" id="refreshBtn">刷新</button>
  </section>

  <div class="error" id="errorBox"></div>
  <section class="summary" id="summary"></section>

  <section class="filters">
    <div class="filter-row"><div class="filter-title">目标</div><div id="serviceFilters" class="filter-row"></div></div>
    <div class="filter-row"><div class="filter-title">节点</div><div id="serverFilters" class="filter-row"></div></div>
  </section>

  <section class="panel">
    <div class="panel-head"><div><div class="panel-title">延迟总览</div><div class="panel-subtitle" id="subtitle">正在加载</div></div><div class="panel-subtitle" id="countInfo"></div></div>
    <div class="chart-wrap"><canvas id="chart"></canvas><div class="empty" id="emptyBox">暂无延迟数据。请在后台服务页添加 ICMP-Ping 或 TCP-Ping 监控任务，并等待一次采集。</div></div>
    <div class="legend" id="legend"></div>
  </section>
</main>
<script>
(function(){
  var colors = ["#5276d8", "#73bf69", "#f2c14e", "#e4576b", "#69bde7", "#3ea375", "#9b6cc3", "#ff8552", "#dd62c1", "#64748b", "#0ea5a6", "#f59e0b"];
  var state = { period: "1d", protocol: "all", hiddenServices: new Set(), hiddenServers: new Set(), data: null };
  var chart = document.getElementById("chart");
  var ctx = chart.getContext("2d");

  function tokenFromValue(value, depth) {
    if (!value || depth > 3) return "";
    if (typeof value === "string") {
      var text = value.trim();
      if (/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(text)) return text;
      if ((text.charAt(0) === "{" || text.charAt(0) === "[") && text.length < 20000) {
        try { return tokenFromValue(JSON.parse(text), depth + 1); } catch (_) { return ""; }
      }
      return "";
    }
    if (Array.isArray(value)) {
      for (var i = 0; i < value.length; i++) { var a = tokenFromValue(value[i], depth + 1); if (a) return a; }
      return "";
    }
    if (typeof value === "object") {
      var names = ["token", "access_token", "accessToken", "jwt", "Authorization"];
      for (var n = 0; n < names.length; n++) { if (value[names[n]]) { var t = tokenFromValue(value[names[n]], depth + 1); if (t) return t; } }
      for (var key in value) { var b = tokenFromValue(value[key], depth + 1); if (b) return b; }
    }
    return "";
  }

  function getToken() {
    var stores = [window.localStorage, window.sessionStorage];
    for (var s = 0; s < stores.length; s++) {
      try {
        for (var i = 0; i < stores[s].length; i++) {
          var token = tokenFromValue(stores[s].getItem(stores[s].key(i)), 0);
          if (token) return token;
        }
      } catch (_) {}
    }
    return "";
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value).replace(/[&<>\"']/g, function(ch){ return {"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[ch]; });
  }

  function fmtMs(value) {
    if (!isFinite(value)) return "-";
    return value < 100 ? value.toFixed(1) + "ms" : Math.round(value) + "ms";
  }

  function filteredSeries() {
    var data = state.data;
    if (!data) return [];
    return data.series.filter(function(s){
      if (state.protocol !== "all" && s.type_name !== state.protocol) return false;
      if (state.hiddenServices.has(String(s.service_id))) return false;
      if (state.hiddenServers.has(String(s.server_id))) return false;
      return s.data_points && s.data_points.length;
    });
  }

  function renderSummary() {
    var series = filteredSeries();
    var totalPoints = 0, sumDelay = 0, delayCount = 0, totalUp = 0, totalDown = 0;
    series.forEach(function(s){
      totalUp += s.total_up || 0;
      totalDown += s.total_down || 0;
      (s.data_points || []).forEach(function(p){ if (p.status !== 0 && p.delay > 0) { sumDelay += p.delay; delayCount++; } totalPoints++; });
    });
    var avg = delayCount ? sumDelay / delayCount : NaN;
    var up = (totalUp + totalDown) ? totalUp * 100 / (totalUp + totalDown) : NaN;
    document.getElementById("summary").innerHTML =
      "<div class='metric'><span>曲线</span><b>" + series.length + "</b></div>" +
      "<div class='metric'><span>平均延迟</span><b>" + fmtMs(avg) + "</b></div>" +
      "<div class='metric'><span>可用率</span><b>" + (isFinite(up) ? up.toFixed(2) + "%" : "-") + "</b></div>" +
      "<div class='metric'><span>采样点</span><b>" + totalPoints + "</b></div>";
  }

  function renderFilter(containerId, items, hiddenSet, labelFn) {
    var box = document.getElementById(containerId);
    box.innerHTML = "";
    var all = document.createElement("button");
    all.className = "chip active";
    all.textContent = "全部";
    all.onclick = function(){ hiddenSet.clear(); renderAll(); };
    box.appendChild(all);
    items.forEach(function(item, index){
      var id = String(item.id);
      var btn = document.createElement("button");
      btn.className = "chip" + (hiddenSet.has(id) ? "" : " active");
      btn.innerHTML = "<span class='dot' style='color:" + colors[index % colors.length] + "'></span>" + escapeHtml(labelFn(item));
      btn.onclick = function(){ hiddenSet.has(id) ? hiddenSet.delete(id) : hiddenSet.add(id); renderAll(); };
      box.appendChild(btn);
    });
  }

  function renderControls() {
    var data = state.data || {services:[], servers:[]};
    renderFilter("serviceFilters", data.services, state.hiddenServices, function(item){ return item.name; });
    renderFilter("serverFilters", data.servers, state.hiddenServers, function(item){ return item.name; });
  }

  function renderLegend(series) {
    var legend = document.getElementById("legend");
    legend.innerHTML = "";
    series.forEach(function(s, index){
      var item = document.createElement("span");
      item.className = "legend-item";
      item.innerHTML = "<span class='dot' style='background:" + colors[index % colors.length] + "'></span>" + escapeHtml(s.server_name + " / " + s.service_name + " " + fmtMs(s.avg_delay));
      legend.appendChild(item);
    });
  }

  function drawChart() {
    var rect = chart.getBoundingClientRect();
    var dpr = window.devicePixelRatio || 1;
    chart.width = Math.max(1, Math.floor(rect.width * dpr));
    chart.height = Math.max(1, Math.floor(rect.height * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, rect.width, rect.height);

    var series = filteredSeries();
    document.getElementById("countInfo").textContent = series.length + " 条曲线";
    document.getElementById("emptyBox").style.display = series.length ? "none" : "flex";
    renderLegend(series);
    renderSummary();
    if (!series.length) return;

    var points = [];
    series.forEach(function(s){ (s.data_points || []).forEach(function(p){ if (p.ts && p.delay > 0) points.push(p); }); });
    if (!points.length) { document.getElementById("emptyBox").style.display = "flex"; return; }

    var minX = Math.min.apply(null, points.map(function(p){ return p.ts; }));
    var maxX = Math.max.apply(null, points.map(function(p){ return p.ts; }));
    var maxY = Math.max.apply(null, points.map(function(p){ return p.delay; }));
    if (minX === maxX) maxX = minX + 60000;
    maxY = Math.max(10, Math.ceil(maxY * 1.2 / 10) * 10);

    var pad = {left:56, right:18, top:20, bottom:42};
    var w = rect.width - pad.left - pad.right;
    var h = rect.height - pad.top - pad.bottom;
    function x(ts){ return pad.left + (ts - minX) / (maxX - minX) * w; }
    function y(v){ return pad.top + h - v / maxY * h; }

    ctx.strokeStyle = "#e5e9f2";
    ctx.lineWidth = 1;
    ctx.fillStyle = "#64748b";
    ctx.font = "12px -apple-system, BlinkMacSystemFont, Segoe UI, Arial";
    for (var gy = 0; gy <= 5; gy++) {
      var value = maxY * gy / 5;
      var yy = y(value);
      ctx.beginPath(); ctx.moveTo(pad.left, yy); ctx.lineTo(pad.left + w, yy); ctx.stroke();
      ctx.fillText(Math.round(value) + "ms", 4, yy + 4);
    }
    for (var gx = 0; gx <= 4; gx++) {
      var ts = minX + (maxX - minX) * gx / 4;
      var dt = new Date(ts);
      var label = String(dt.getHours()).padStart(2,"0") + ":" + String(dt.getMinutes()).padStart(2,"0");
      ctx.fillText(label, pad.left + w * gx / 4 - 14, pad.top + h + 26);
    }

    series.forEach(function(s, index){
      var dps = (s.data_points || []).filter(function(p){ return p.ts && p.delay > 0; });
      if (!dps.length) return;
      ctx.strokeStyle = colors[index % colors.length];
      ctx.lineWidth = 1.8;
      ctx.beginPath();
      dps.forEach(function(p, i){ var xx = x(p.ts), yy = y(p.delay); if (i === 0) ctx.moveTo(xx, yy); else ctx.lineTo(xx, yy); });
      ctx.stroke();
    });
  }

  function renderAll() {
    renderControls();
    drawChart();
  }

  async function load() {
    document.getElementById("errorBox").style.display = "none";
    document.getElementById("subtitle").textContent = "正在加载";
    var headers = {};
    var token = getToken();
    if (token) headers.Authorization = "Bearer " + token;
    try {
      var res = await fetch("/api/v1/netwatch/latency?period=" + encodeURIComponent(state.period), { headers: headers, credentials: "same-origin" });
      var body = await res.json();
      if (!body.success) throw new Error(body.error || "请求失败");
      state.data = body.data;
      document.getElementById("subtitle").textContent = "更新时间 " + new Date(state.data.generated_at).toLocaleString();
      renderAll();
    } catch (err) {
      document.getElementById("errorBox").textContent = err.message || String(err);
      document.getElementById("errorBox").style.display = "block";
      document.getElementById("subtitle").textContent = "加载失败";
    }
  }

  document.getElementById("periodGroup").addEventListener("click", function(e){
    var period = e.target.getAttribute("data-period");
    if (!period) return;
    state.period = period;
    Array.prototype.forEach.call(this.querySelectorAll("button"), function(btn){ btn.classList.toggle("active", btn.getAttribute("data-period") === period); });
    load();
  });
  document.getElementById("protocolGroup").addEventListener("click", function(e){
    var protocol = e.target.getAttribute("data-protocol");
    if (!protocol) return;
    state.protocol = protocol;
    Array.prototype.forEach.call(this.querySelectorAll("button"), function(btn){ btn.classList.toggle("active", btn.getAttribute("data-protocol") === protocol); });
    renderAll();
  });
  document.getElementById("refreshBtn").onclick = load;
  window.addEventListener("resize", drawChart);
  load();
})();
</script>
</body>
</html>`
