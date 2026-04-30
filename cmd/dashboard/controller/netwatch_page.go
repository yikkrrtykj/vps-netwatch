package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/service/singleton"
)

func netwatchShouldInjectUserIndex(filePath string) bool {
	return filePath == singleton.Conf.UserTemplate+"/index.html" || filePath == singleton.Conf.AdminTemplate+"/index.html"
}

func netwatchServeInjectedUserIndex(c *gin.Context, statusCode int, content []byte) {
	html := netwatchApplyBranding(string(content))
	scripts := ""
	if !strings.Contains(html, "vps-netwatch-home-button") {
		scripts += netwatchHomeButtonScript
	}
	if !strings.Contains(html, "vps-netwatch-server-board-script") {
		scripts += netwatchServerBoardScript
	}
	if scripts != "" && strings.Contains(html, "</body>") {
		html = strings.Replace(html, "</body>", scripts+"</body>", 1)
	} else if scripts != "" {
		html += scripts
	}
	c.Data(statusCode, "text/html; charset=utf-8", []byte(html))
}

func netwatchApplyBranding(content string) string {
	return strings.NewReplacer(
		"哪吒监控 Nezha Monitoring", "vps-netwatch",
		"Nezha Monitoring", "vps-netwatch",
		"哪吒监控", "vps-netwatch",
		"nezha-dash", "vps-netwatch",
		"Nezha", "vps-netwatch",
	).Replace(content)
}

const netwatchHomeButtonScript = `<script id="vps-netwatch-home-button">
(function () {
  var marker = "data-vps-netwatch-latency";
  var buttonClass = "inset-shadow-2xs inset-shadow-white/20 flex cursor-pointer flex-col items-center gap-0 rounded-[50px] bg-blue-100 p-2.5 text-blue-600 transition-all dark:bg-blue-900 dark:text-blue-100";
  var activeClass = "inset-shadow-black/20 bg-blue-600 text-white dark:bg-blue-100 dark:text-blue-600";
  var colors = ["#5276d8", "#f2c14e", "#73bf69", "#e4576b", "#69bde7", "#8b5cf6", "#14b8a6", "#f97316", "#ec4899", "#64748b"];
  var state = { visible: false, loaded: false, loading: false, data: null, domain: null, view: null, hover: null, selectedServerId: "", peerTargetServerId: "", peerSaving: false, refreshTimer: 0, peerTimer: 0, peerWaitUntil: 0 };
  var panel;
  var canvas;
  var ctx;
  var tooltip;
  var lastPlot;

  function injectStyle() {
    if (document.getElementById("vps-netwatch-inline-style")) return;
    var style = document.createElement("style");
    style.id = "vps-netwatch-inline-style";
    style.textContent =
      "#vps-netwatch-latency-panel{margin-top:14px;border:1px solid rgba(148,163,184,.26);border-radius:16px;background:rgba(255,255,255,.86);box-shadow:0 10px 28px rgba(15,23,42,.08);overflow:hidden;color:#111827}" +
      ".dark #vps-netwatch-latency-panel{background:rgba(0,0,0,.48);border-color:rgba(255,255,255,.12);color:#f8fafc}" +
      "#vps-netwatch-latency-panel[hidden]{display:none!important}" +
      ".vpsnw-head{display:grid;grid-template-columns:max-content minmax(0,1fr) auto;align-items:center;gap:8px 10px;padding:12px 16px;border-bottom:1px solid rgba(148,163,184,.22)}" +
      ".vpsnw-title{grid-column:1;grid-row:1;font-weight:800;font-size:14px;white-space:nowrap}.vpsnw-sub{color:#64748b;font-size:12px}.dark .vpsnw-sub{color:#94a3b8}" +
      ".vpsnw-server-tabs{grid-column:2;grid-row:1;display:flex;flex-wrap:wrap;justify-content:flex-start;gap:6px;min-width:0}.vpsnw-server-btn{border:1px solid rgba(148,163,184,.35);border-radius:999px;background:rgba(248,250,252,.9);color:#334155;cursor:pointer;font-size:12px;line-height:1;padding:5px 10px;white-space:nowrap}.vpsnw-server-btn.active{border-color:#2563eb;background:#2563eb;color:#fff}.dark .vpsnw-server-btn{background:rgba(255,255,255,.08);border-color:rgba(255,255,255,.16);color:#e2e8f0}.dark .vpsnw-server-btn.active{background:#dbeafe;border-color:#dbeafe;color:#1d4ed8}" +
      ".vpsnw-range{grid-column:3;grid-row:1;justify-self:end;text-align:right;max-width:440px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.vpsnw-peer-row{grid-column:2/4;grid-row:2;display:flex;flex-wrap:wrap;align-items:center;justify-content:flex-start;gap:6px;min-width:0;font-size:12px;color:#64748b}.dark .vpsnw-peer-row{color:#94a3b8}.vpsnw-peer-label{grid-column:1;grid-row:2;font-size:12px;white-space:nowrap;color:#64748b}.dark .vpsnw-peer-label{color:#94a3b8}.vpsnw-peer-tabs{display:flex;flex-wrap:wrap;justify-content:flex-start;gap:6px;min-width:0}.vpsnw-peer-note{min-width:48px;color:#2563eb}.dark .vpsnw-peer-note{color:#93c5fd}.vpsnw-peer-btn{border:1px solid rgba(148,163,184,.35);border-radius:7px;background:rgba(248,250,252,.9);color:#334155;cursor:pointer;font-size:12px;line-height:1;padding:5px 9px;white-space:nowrap}.vpsnw-peer-btn.active{border-color:#0f172a;background:#0f172a;color:#fff}.vpsnw-peer-btn:disabled{cursor:not-allowed;opacity:.58}.dark .vpsnw-peer-btn{background:rgba(255,255,255,.08);border-color:rgba(255,255,255,.16);color:#e2e8f0}.dark .vpsnw-peer-btn.active{background:#f8fafc;border-color:#f8fafc;color:#0f172a}" +
      ".vpsnw-legend{display:flex;flex-wrap:wrap;justify-content:center;gap:10px 18px;padding:10px 14px 2px;font-size:13px}.vpsnw-legend span{display:inline-flex;align-items:center;gap:6px}.vpsnw-dot{width:10px;height:10px;border-radius:50%;display:inline-block}" +
      ".vpsnw-chart{position:relative;height:360px;padding:6px 14px 14px}.vpsnw-chart canvas{width:100%;height:100%;display:block}" +
      ".vpsnw-tip{display:none;position:absolute;z-index:20;min-width:160px;max-width:260px;padding:10px 12px;border:1px solid rgba(148,163,184,.35);border-radius:8px;background:rgba(255,255,255,.96);box-shadow:0 12px 28px rgba(15,23,42,.2);font-size:13px;color:#111827;pointer-events:none}.dark .vpsnw-tip{background:rgba(15,15,15,.96);color:#f8fafc}" +
      ".vpsnw-tip-time{color:#475569;margin-bottom:6px}.dark .vpsnw-tip-time{color:#cbd5e1}.vpsnw-tip-row{display:flex;align-items:center;justify-content:space-between;gap:14px;line-height:1.7}.vpsnw-tip-name{display:flex;align-items:center;gap:6px}" +
      ".vpsnw-empty{display:none;padding:22px 16px;color:#94a3b8;text-align:center;font-size:13px}" +
      "@media(max-width:760px){.vpsnw-chart{height:300px;padding-left:8px;padding-right:8px}.vpsnw-head{grid-template-columns:1fr}.vpsnw-title,.vpsnw-server-tabs,.vpsnw-range,.vpsnw-peer-label,.vpsnw-peer-row{grid-column:1;grid-row:auto}.vpsnw-range{justify-self:start;max-width:100%;text-align:left;white-space:normal}}";
    document.head.appendChild(style);
  }

  function getToken() {
    function scan(value, depth) {
      if (!value || depth > 3) return "";
      if (typeof value === "string") {
        var text = value.trim();
        if (/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(text)) return text;
        if ((text.charAt(0) === "{" || text.charAt(0) === "[") && text.length < 20000) {
          try { return scan(JSON.parse(text), depth + 1); } catch (_) { return ""; }
        }
        return "";
      }
      if (Array.isArray(value)) {
        for (var i = 0; i < value.length; i++) { var a = scan(value[i], depth + 1); if (a) return a; }
      } else if (typeof value === "object") {
        for (var key in value) { var b = scan(value[key], depth + 1); if (b) return b; }
      }
      return "";
    }
    var stores = [window.localStorage, window.sessionStorage];
    for (var s = 0; s < stores.length; s++) {
      try {
        for (var i = 0; i < stores[s].length; i++) {
          var token = scan(stores[s].getItem(stores[s].key(i)), 0);
          if (token) return token;
        }
      } catch (_) {}
    }
    return "";
  }

  function createPanel(host) {
    injectStyle();
    if (!panel) {
      panel = document.createElement("section");
      panel.id = "vps-netwatch-latency-panel";
      panel.hidden = true;
      panel.innerHTML =
        '<div class="vpsnw-head"><div class="vpsnw-title">延迟</div><div class="vpsnw-server-tabs" id="vpsnw-server-tabs"></div><div class="vpsnw-sub vpsnw-range" id="vpsnw-range">加载中</div><span class="vpsnw-peer-label">互 ping 目标</span><div class="vpsnw-peer-row"><div class="vpsnw-peer-tabs" id="vpsnw-peer-tabs"></div><span class="vpsnw-peer-note" id="vpsnw-peer-note"></span></div></div>' +
        '<div class="vpsnw-empty" id="vpsnw-empty">暂无延迟数据</div>' +
        '<div class="vpsnw-legend" id="vpsnw-legend"></div>' +
        '<div class="vpsnw-chart"><canvas id="vpsnw-canvas"></canvas><div class="vpsnw-tip" id="vpsnw-tip"></div></div>';
      canvas = panel.querySelector("#vpsnw-canvas");
      ctx = canvas.getContext("2d");
      tooltip = panel.querySelector("#vpsnw-tip");
      canvas.addEventListener("wheel", onWheel, { passive: false });
      canvas.addEventListener("mousemove", onMove);
      canvas.addEventListener("mouseleave", function () { state.hover = null; tooltip.style.display = "none"; draw(); });
      canvas.addEventListener("dblclick", function () { if (state.domain) { state.view = { start: state.domain.start, end: state.domain.end }; draw(); } });
      window.addEventListener("resize", draw);
    }
    if (!panel.isConnected || panel.previousElementSibling !== host) {
      host.insertAdjacentElement("afterend", panel);
    }
    return panel;
  }

  function pad(n) { return String(n).padStart(2, "0"); }
  function fmtTime(ts) { var d = new Date(ts); return d.getFullYear() + "-" + pad(d.getMonth()+1) + "-" + pad(d.getDate()) + " " + pad(d.getHours()) + ":" + pad(d.getMinutes()) + ":" + pad(d.getSeconds()); }
  function fmtAxis(ts) { var d = new Date(ts); return pad(d.getHours()) + ":" + pad(d.getMinutes()); }
  function fmtMs(v) { return !isFinite(v) ? "-" : (v < 100 ? v.toFixed(2) : Math.round(v)) + "ms"; }
  function escapeHtml(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return {"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[ch];
    });
  }

  function ensureSelectedServer() {
    var servers = (state.data && state.data.servers) || [];
    var saved = "";
    try { saved = window.localStorage.getItem("vpsnw-selected-server") || ""; } catch (_) {}
    function exists(id) {
      return servers.some(function (server) { return String(server.id) === String(id); });
    }
    if (state.selectedServerId && exists(state.selectedServerId)) return;
    if (saved && exists(saved)) {
      state.selectedServerId = saved;
      return;
    }
    state.selectedServerId = servers.length ? String(servers[0].id) : "";
  }

  function selectedServerName() {
    var servers = (state.data && state.data.servers) || [];
    for (var i = 0; i < servers.length; i++) {
      if (String(servers[i].id) === String(state.selectedServerId)) return servers[i].name || ("VPS " + servers[i].id);
    }
    return "未选择 VPS";
  }

  function renderServerTabs() {
    var box = panel && panel.querySelector("#vpsnw-server-tabs");
    if (!box || !state.data) return;
    ensureSelectedServer();
    box.innerHTML = "";
    (state.data.servers || []).forEach(function (server) {
      var id = String(server.id);
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "vpsnw-server-btn" + (id === String(state.selectedServerId) ? " active" : "");
      btn.textContent = server.name || ("VPS " + id);
      btn.onclick = function () {
        state.selectedServerId = id;
        state.hover = null;
        try { window.localStorage.setItem("vpsnw-selected-server", id); } catch (_) {}
        if (state.peerTargetServerId && state.peerTargetServerId !== id) startPeerWait();
        draw();
      };
      box.appendChild(btn);
    });
  }

  function syncPeerState() {
    state.peerTargetServerId = state.data && state.data.peer && state.data.peer.enabled ? String(state.data.peer.target_server_id || "") : "";
  }

  function renderPeerTargets() {
    var box = panel && panel.querySelector("#vpsnw-peer-tabs");
    if (!box || !state.data) return;
    var servers = state.data.servers || [];
    box.innerHTML = "";

    function addButton(label, id, active, disabled, title) {
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "vpsnw-peer-btn" + (active ? " active" : "");
      btn.textContent = label;
      btn.disabled = disabled || state.peerSaving;
      if (title) btn.title = title;
      btn.onclick = function () { setPeerTarget(id); };
      box.appendChild(btn);
    }

    addButton("关闭", "0", !state.peerTargetServerId, false, "默认只看上海电信和上海联通");
    servers.forEach(function (server) {
      var id = String(server.id);
      var label = server.name || ("VPS " + id);
      var sameAsSource = id === String(state.selectedServerId);
      var disabled = !server.ip || sameAsSource;
      var title = sameAsSource ? "源 VPS 和目标 VPS 不能相同" : (disabled ? "这台 VPS 还没有上报公网 IP" : "让当前 VPS ping " + label);
      addButton(label, id, id === String(state.peerTargetServerId), disabled, title);
    });
  }

  async function setPeerTarget(id) {
    if (state.peerSaving) return;
    state.peerSaving = true;
    renderPeerTargets();
    try {
      var headers = { "Content-Type": "application/json" };
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      var res = await fetch("/api/v1/netwatch/peer-target", {
        method: "POST",
        headers: headers,
        credentials: "same-origin",
        body: JSON.stringify({ target_server_id: id === "0" ? 0 : Number(id) })
      });
      var body = await res.json();
      if (!body.success) throw new Error(body.error || "设置失败");
      state.peerTargetServerId = body.data && body.data.enabled ? String(body.data.target_server_id || "") : "";
      state.loaded = false;
      if (state.peerTargetServerId) {
        startPeerWait();
      } else {
        stopPeerWait();
      }
      setEmptyText(state.peerTargetServerId ? "采集中" : "暂无延迟数据");
      await load({ resetView: false });
    } catch (err) {
      setEmptyText(err.message || String(err));
      panel.querySelector("#vpsnw-empty").style.display = "block";
    } finally {
      state.peerSaving = false;
      renderPeerTargets();
    }
  }

  function aggregate() {
    if (!state.data) return [];
    ensureSelectedServer();
    var grouped = {};
    (state.data.series || []).forEach(function (raw) {
      if (state.selectedServerId && String(raw.server_id) !== String(state.selectedServerId)) return;
      var isPeer = !!raw.is_peer;
      if (isPeer) {
        if (!state.peerTargetServerId || String(raw.peer_server_id) !== String(state.peerTargetServerId)) return;
        if (String(raw.peer_server_id) === String(state.selectedServerId)) return;
      }
      var key = isPeer ? ("peer:" + String(raw.peer_server_id)) : String(raw.service_id);
      if (!grouped[key]) grouped[key] = { name: raw.service_name, target: raw.target || "", type_name: raw.type_name || "", display_index: raw.display_index || 0, peer_server_id: raw.peer_server_id || 0, is_peer: isPeer, points: {}, total: 0, count: 0 };
      (raw.data_points || []).forEach(function (p) {
        if (!p || p.status === 0 || !(p.delay > 0)) return;
        var minute = Math.floor(p.ts / 60000) * 60000;
        if (!grouped[key].points[minute]) grouped[key].points[minute] = [];
        grouped[key].points[minute].push(p.delay);
        grouped[key].total += p.delay;
        grouped[key].count += 1;
      });
    });
    return Object.keys(grouped).map(function (key) {
      var g = grouped[key];
      var points = Object.keys(g.points).map(function (ts) {
        var values = g.points[ts];
        var sum = values.reduce(function (a, b) { return a + b; }, 0);
        return { ts: Number(ts), delay: sum / values.length };
      }).sort(function (a, b) { return a.ts - b.ts; });
      return { name: g.name, target: g.target, type_name: g.type_name, display_index: g.display_index, is_peer: g.is_peer, peer_server_id: g.peer_server_id, avg: g.count ? g.total / g.count : 0, points: points };
    }).filter(function (s) { return s.points.length; }).sort(function (a, b) {
      if (a.display_index !== b.display_index) return b.display_index - a.display_index;
      return a.name.localeCompare(b.name);
    });
  }

  function setEmptyText(text) {
    var empty = panel && panel.querySelector("#vpsnw-empty");
    if (empty) empty.textContent = text || "暂无延迟数据";
  }

  function hasActivePeerSeries(series) {
    if (!state.peerTargetServerId || String(state.peerTargetServerId) === String(state.selectedServerId)) return false;
    return series.some(function (item) { return item.is_peer && String(item.peer_server_id) === String(state.peerTargetServerId) && item.points.length; });
  }

  function updatePeerNote(series) {
    var note = panel && panel.querySelector("#vpsnw-peer-note");
    if (!note) return;
    note.textContent = "";
    if (!state.peerTargetServerId) return;
    if (String(state.peerTargetServerId) === String(state.selectedServerId)) {
      note.textContent = "请选择其它目标";
      stopPeerWait();
      return;
    }
    if (hasActivePeerSeries(series)) {
      stopPeerWait();
      return;
    }
    note.textContent = state.peerWaitUntil && Date.now() <= state.peerWaitUntil ? "采集中" : "暂无数据";
  }

  function updateDomain(resetView) {
    var end = (state.data && state.data.generated_at) || Date.now();
    var start = end - 86400000;
    var oldView = state.view;
    state.domain = { start: start, end: end };
    if (resetView || !oldView) {
      state.view = { start: start, end: end };
    } else {
      clampView(oldView.start, oldView.end);
    }
  }

  function clampView(start, end) {
    var minRange = 5 * 60000;
    var maxRange = state.domain.end - state.domain.start;
    var range = Math.max(minRange, Math.min(maxRange, end - start));
    var center = (start + end) / 2;
    start = center - range / 2;
    end = center + range / 2;
    if (start < state.domain.start) { end += state.domain.start - start; start = state.domain.start; }
    if (end > state.domain.end) { start -= end - state.domain.end; end = state.domain.end; }
    state.view = { start: Math.max(state.domain.start, start), end: Math.min(state.domain.end, end) };
  }

  async function load(options) {
    options = options || {};
    if (state.loading) return;
    state.loading = true;
    var headers = {};
    try {
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      var res = await fetch("/api/v1/netwatch/latency?period=1d", { headers: headers, credentials: "same-origin" });
      var body = await res.json();
      if (!body.success) throw new Error(body.error || "请求失败");
      state.data = body.data;
      state.loaded = true;
      syncPeerState();
      ensureSelectedServer();
      updateDomain(options.resetView !== false);
      draw();
    } finally {
      state.loading = false;
    }
  }

  function draw() {
    if (!panel || panel.hidden || !canvas || !ctx || !state.data) return;
    renderServerTabs();
    renderPeerTargets();
    var series = aggregate();
    updatePeerNote(series);
    if (!series.length) setEmptyText(state.peerTargetServerId && state.peerTargetServerId !== state.selectedServerId && state.peerWaitUntil && Date.now() <= state.peerWaitUntil ? "采集中" : "暂无延迟数据");
    panel.querySelector("#vpsnw-empty").style.display = series.length ? "none" : "block";
    panel.querySelector(".vpsnw-chart").style.display = series.length ? "block" : "none";
    renderLegend(series);
    panel.querySelector("#vpsnw-range").textContent = selectedServerName() + " | " + fmtTime(state.view.start) + " - " + fmtTime(state.view.end);
    if (!series.length) return;

    var rect = canvas.getBoundingClientRect();
    var dpr = window.devicePixelRatio || 1;
    canvas.width = Math.max(1, Math.floor(rect.width * dpr));
    canvas.height = Math.max(1, Math.floor(rect.height * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, rect.width, rect.height);

    var pad = { left: 44, right: 18, top: 18, bottom: 54 };
    var plot = { left: pad.left, top: pad.top, width: rect.width - pad.left - pad.right, height: rect.height - pad.top - pad.bottom };
    lastPlot = plot;
    var visible = [];
    series.forEach(function (s) { s.points.forEach(function (p) { if (p.ts >= state.view.start && p.ts <= state.view.end) visible.push(p); }); });
    var maxY = Math.max.apply(null, (visible.length ? visible : [{ delay: 10 }]).map(function (p) { return p.delay; }));
    maxY = Math.max(10, Math.ceil(maxY * 1.18 / 10) * 10);
    function x(ts) { return plot.left + (ts - state.view.start) / (state.view.end - state.view.start) * plot.width; }
    function y(v) { return plot.top + plot.height - v / maxY * plot.height; }

    var isDark = document.documentElement.classList.contains("dark") || document.body.classList.contains("dark");
    ctx.strokeStyle = isDark ? "rgba(255,255,255,.12)" : "#dbe2ee";
    ctx.lineWidth = 1;
    ctx.fillStyle = isDark ? "#cbd5e1" : "#334155";
    ctx.font = "12px -apple-system,BlinkMacSystemFont,Segoe UI,Arial";
    for (var gy = 0; gy <= 4; gy++) {
      var value = maxY * gy / 4;
      var yy = y(value);
      ctx.beginPath(); ctx.moveTo(plot.left, yy); ctx.lineTo(plot.left + plot.width, yy); ctx.stroke();
      ctx.fillText(Math.round(value) + "ms", 0, yy + 4);
    }
    for (var gx = 0; gx <= 6; gx++) {
      var ts = state.view.start + (state.view.end - state.view.start) * gx / 6;
      ctx.fillText(fmtAxis(ts), plot.left + plot.width * gx / 6 - 14, plot.top + plot.height + 22);
    }

    series.forEach(function (s, index) {
      var pts = s.points.filter(function (p) { return p.ts >= state.view.start && p.ts <= state.view.end; });
      if (!pts.length) return;
      ctx.strokeStyle = colors[index % colors.length];
      ctx.lineWidth = 1.8;
      ctx.beginPath();
      pts.forEach(function (p, i) { if (i === 0) ctx.moveTo(x(p.ts), y(p.delay)); else ctx.lineTo(x(p.ts), y(p.delay)); });
      ctx.stroke();
    });

    if (state.hover) drawHover(series, plot, x, y);
  }

  function renderLegend(series) {
    var legend = panel.querySelector("#vpsnw-legend");
    legend.innerHTML = "";
    series.forEach(function (s, index) {
      var pts = s.points.filter(function (p) { return p.ts >= state.view.start && p.ts <= state.view.end; });
      var avg = pts.length ? pts.reduce(function (sum, p) { return sum + p.delay; }, 0) / pts.length : s.avg;
      var item = document.createElement("span");
      item.innerHTML = '<i class="vpsnw-dot" style="background:' + colors[index % colors.length] + '"></i>' + escapeHtml(s.name) + " " + fmtMs(avg);
      legend.appendChild(item);
    });
  }

  function drawHover(series, plot, x, y) {
    if (state.hover.x < plot.left || state.hover.x > plot.left + plot.width || state.hover.y < plot.top || state.hover.y > plot.top + plot.height) {
      tooltip.style.display = "none";
      return;
    }
    var hoverTs = state.view.start + (state.hover.x - plot.left) / plot.width * (state.view.end - state.view.start);
    var items = [];
    series.forEach(function (s, index) {
      var pts = s.points.filter(function (p) { return p.ts >= state.view.start && p.ts <= state.view.end; });
      if (!pts.length) return;
      var nearest = pts[0];
      for (var i = 1; i < pts.length; i++) if (Math.abs(pts[i].ts - hoverTs) < Math.abs(nearest.ts - hoverTs)) nearest = pts[i];
      items.push({ s: s, p: nearest, color: colors[index % colors.length] });
    });
    if (!items.length) return;
    var lineX = x(items[0].p.ts);
    ctx.strokeStyle = "#9fb1cf";
    ctx.setLineDash([4, 3]);
    ctx.beginPath(); ctx.moveTo(lineX, plot.top); ctx.lineTo(lineX, plot.top + plot.height); ctx.stroke();
    ctx.setLineDash([]);
    items.forEach(function (item) {
      ctx.fillStyle = item.color;
      ctx.beginPath(); ctx.arc(x(item.p.ts), y(item.p.delay), 4, 0, Math.PI * 2); ctx.fill();
      ctx.strokeStyle = "#fff"; ctx.lineWidth = 1.4; ctx.stroke();
    });
    tooltip.innerHTML = '<div class="vpsnw-tip-time">' + fmtTime(items[0].p.ts) + '</div>' + items.map(function (item) {
      return '<div class="vpsnw-tip-row"><span class="vpsnw-tip-name"><i class="vpsnw-dot" style="background:' + item.color + '"></i>' + escapeHtml(item.s.name) + '</span><strong>' + fmtMs(item.p.delay) + '</strong></div>';
    }).join("");
    tooltip.style.display = "block";
    tooltip.style.left = Math.min(panel.clientWidth - tooltip.offsetWidth - 20, Math.max(10, state.hover.x + 18)) + "px";
    tooltip.style.top = Math.min(canvas.parentElement.clientHeight - tooltip.offsetHeight - 12, Math.max(10, state.hover.y - 18)) + "px";
  }

  function onWheel(event) {
    if (!state.domain || !state.view || !lastPlot) return;
    event.preventDefault();
    var rect = canvas.getBoundingClientRect();
    var x = event.clientX - rect.left;
    var center = state.view.start + Math.max(0, Math.min(1, (x - lastPlot.left) / lastPlot.width)) * (state.view.end - state.view.start);
    var range = (state.view.end - state.view.start) * (event.deltaY > 0 ? 1.22 : 0.82);
    clampView(center - range / 2, center + range / 2);
    draw();
  }

  function onMove(event) {
    var rect = canvas.getBoundingClientRect();
    state.hover = { x: event.clientX - rect.left, y: event.clientY - rect.top };
    draw();
  }

  function toggle(btn, host) {
    createPanel(host);
    state.visible = !state.visible;
    panel.hidden = !state.visible;
    btn.className = buttonClass + (state.visible ? " " + activeClass : "");
    if (state.visible) {
      startRefresh();
    } else {
      stopRefresh();
      stopPeerWait();
    }
    if (state.visible && !state.loaded) {
      load({ resetView: true }).catch(function (err) {
        setEmptyText(err.message || String(err));
        panel.querySelector("#vpsnw-empty").style.display = "block";
      });
    } else if (state.visible) {
      draw();
    }
  }

  function startRefresh() {
    if (state.refreshTimer) return;
    state.refreshTimer = window.setInterval(function () {
      if (state.visible) load({ resetView: false }).catch(function () {});
    }, 30000);
  }

  function stopRefresh() {
    if (state.refreshTimer) window.clearInterval(state.refreshTimer);
    state.refreshTimer = 0;
  }

  function startPeerWait() {
    state.peerWaitUntil = Date.now() + 120000;
    if (state.peerTimer) return;
    state.peerTimer = window.setInterval(function () {
      if (!state.visible || !state.peerWaitUntil || Date.now() > state.peerWaitUntil) {
        stopPeerWait();
        draw();
        return;
      }
      load({ resetView: false }).catch(function () {});
    }, 10000);
  }

  function stopPeerWait() {
    if (state.peerTimer) window.clearInterval(state.peerTimer);
    state.peerTimer = 0;
    state.peerWaitUntil = 0;
  }

  function ensureButton() {
    var controlsRow = document.querySelector(".server-overview-controls");
    var controls = document.querySelector(".server-overview-controls section");
    if (!controlsRow || !controls) return false;

    var btn = document.querySelector("[" + marker + "]");
    if (btn && !controls.contains(btn)) btn.remove();
    btn = controls.querySelector("[" + marker + "]");

    if (!btn) {
      btn = document.createElement("button");
      btn.setAttribute(marker, "1");
      btn.type = "button";
      btn.title = "延迟";
      btn.setAttribute("aria-label", "延迟");
      btn.innerHTML = '<svg viewBox="0 0 20 20" width="13" height="13" fill="currentColor" aria-hidden="true"><path d="M3 12.6a1 1 0 0 1 1-1h1.9l2.2-5.7a1 1 0 0 1 1.86 0l2.63 6.84 1.55-3.1a1 1 0 0 1 .9-.55H17a1 1 0 1 1 0 2h-1.34l-2.36 4.72a1 1 0 0 1-1.83-.08L9.03 9.37l-1.5 3.88a1 1 0 0 1-.93.64H4a1 1 0 0 1-1-1.29Z"/></svg>';
    }
    btn.className = buttonClass + (state.visible ? " " + activeClass : "");
    btn.onclick = function () { toggle(btn, controlsRow); };

    var buttons = Array.prototype.filter.call(controls.querySelectorAll(":scope > button"), function (item) {
      return !item.hasAttribute(marker);
    });
    if (buttons.length < 3) return !!btn.isConnected;
    if (btn.parentNode !== controls || btn.previousElementSibling !== buttons[2]) {
      buttons[2].after(btn);
    }
    return true;
  }

  ensureButton();
  window.setInterval(ensureButton, 1000);
  var observer = new MutationObserver(ensureButton);
  observer.observe(document.documentElement, { childList: true, subtree: true });
  document.addEventListener("visibilitychange", ensureButton);
})();
</script>`

const netwatchServerBoardScript = `<script id="vps-netwatch-server-board-script">
(function () {
  if (window.__vpsNetwatchServerBoardInstalled) return;
  window.__vpsNetwatchServerBoardInstalled = true;

  var board;
  var refreshTimer = 0;
  var state = {
    data: null,
    view: readNativeView(),
    loading: false,
    error: "",
    note: "",
    discovered: [],
    mihomoUrl: "",
    mihomoSecret: ""
  };
  var serviceEmptyPattern = /No Service|no service|暂无服务|没有服务|服务数据|暂无数据/i;

  function readNativeView() {
    try { return window.localStorage.getItem("inline") === "1" ? "table" : "grid"; } catch (_) { return "grid"; }
  }

  function getToken() {
    function scan(value, depth) {
      if (!value || depth > 3) return "";
      if (typeof value === "string") {
        var text = value.trim();
        if (/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(text)) return text;
        if ((text.charAt(0) === "{" || text.charAt(0) === "[") && text.length < 20000) {
          try { return scan(JSON.parse(text), depth + 1); } catch (_) { return ""; }
        }
        return "";
      }
      if (Array.isArray(value)) {
        for (var i = 0; i < value.length; i++) { var a = scan(value[i], depth + 1); if (a) return a; }
      } else if (typeof value === "object") {
        for (var key in value) { var b = scan(value[key], depth + 1); if (b) return b; }
      }
      return "";
    }
    var stores = [window.localStorage, window.sessionStorage];
    for (var s = 0; s < stores.length; s++) {
      try {
        for (var i = 0; i < stores[s].length; i++) {
          var token = scan(stores[s].getItem(stores[s].key(i)), 0);
          if (token) return token;
        }
      } catch (_) {}
    }
    return "";
  }

  function injectStyle() {
    if (document.getElementById("vps-netwatch-server-board-style")) return;
    var style = document.createElement("style");
    style.id = "vps-netwatch-server-board-style";
    style.textContent =
      ".vpsnw-server-board-ready .server-overview-controls section>button:nth-of-type(2){display:none!important}" +
      ".vpsnw-server-board-ready .server-card-list,.vpsnw-server-board-ready .server-inline-list{display:none!important}" +
      "#vps-netwatch-server-board{margin-top:14px;color:#111827}" +
      ".dark #vps-netwatch-server-board{color:#f8fafc}" +
      ".vpsnw-board-top{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:10px 14px;align-items:center;margin-bottom:10px}" +
      ".vpsnw-board-title{display:flex;align-items:center;gap:8px;font-size:13px;font-weight:800;color:#0f172a}.dark .vpsnw-board-title{color:#f8fafc}" +
      ".vpsnw-board-title span{width:8px;height:8px;border-radius:999px;background:#22c55e;box-shadow:0 0 0 3px rgba(34,197,94,.15)}" +
      ".vpsnw-board-tools{display:flex;flex-wrap:wrap;justify-content:flex-end;gap:6px}.vpsnw-board-tabs{display:flex;gap:4px;border:1px solid rgba(148,163,184,.28);border-radius:999px;background:rgba(255,255,255,.75);padding:3px}.dark .vpsnw-board-tabs{background:rgba(15,23,42,.62);border-color:rgba(255,255,255,.14)}" +
      ".vpsnw-board-tabs button{border:0;border-radius:999px;background:transparent;color:#64748b;cursor:pointer;font-size:12px;font-weight:700;line-height:1;padding:7px 10px}.dark .vpsnw-board-tabs button{color:#cbd5e1}.vpsnw-board-tabs button.active{background:#2563eb;color:#fff}" +
      ".vpsnw-wizard{grid-column:1/-1;display:grid;gap:7px;border:1px solid rgba(148,163,184,.22);border-radius:8px;background:rgba(255,255,255,.66);padding:9px}.dark .vpsnw-wizard{background:rgba(15,23,42,.58);border-color:rgba(255,255,255,.12)}" +
      ".vpsnw-wizard-row{display:flex;flex-wrap:wrap;align-items:center;gap:7px}.vpsnw-wizard label{display:inline-flex;align-items:center;gap:6px;min-height:30px;border:1px solid rgba(148,163,184,.28);border-radius:7px;background:rgba(248,250,252,.9);color:#64748b;font-size:12px;padding:4px 8px}.dark .vpsnw-wizard label{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14);color:#94a3b8}" +
      ".vpsnw-wizard input{min-width:150px;border:0;outline:0;background:transparent;color:#0f172a;font:inherit}.dark .vpsnw-wizard input{color:#f8fafc}.vpsnw-wizard .wide input{min-width:230px}.vpsnw-board-btn{border:1px solid rgba(37,99,235,.35);border-radius:7px;background:#2563eb;color:#fff;cursor:pointer;font-size:12px;font-weight:700;line-height:1;padding:8px 10px}.vpsnw-board-btn.secondary{background:rgba(248,250,252,.9);color:#1d4ed8}.dark .vpsnw-board-btn.secondary{background:rgba(255,255,255,.08);color:#bfdbfe}" +
      ".vpsnw-board-note{font-size:12px;color:#2563eb}.vpsnw-board-note.err{color:#dc2626}.dark .vpsnw-board-note{color:#93c5fd}.dark .vpsnw-board-note.err{color:#fca5a5}.vpsnw-discover summary{cursor:pointer;font-size:12px;color:#64748b;user-select:none}.dark .vpsnw-discover summary{color:#94a3b8}.vpsnw-discover-list{display:flex;flex-wrap:wrap;gap:6px;margin-top:8px}.vpsnw-discover-item{display:inline-flex;align-items:center;gap:6px;border:1px solid rgba(148,163,184,.26);border-radius:999px;background:rgba(248,250,252,.86);font-size:12px;padding:5px 7px}.dark .vpsnw-discover-item{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14)}.vpsnw-mini-btn{border:0;border-radius:999px;background:#0f172a;color:#fff;cursor:pointer;font-size:11px;line-height:1;padding:5px 7px}.dark .vpsnw-mini-btn{background:#f8fafc;color:#0f172a}" +
      ".vpsnw-server-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:10px}.vpsnw-node-card{border:1px solid rgba(148,163,184,.24);border-radius:8px;background:rgba(255,255,255,.84);box-shadow:0 10px 24px rgba(15,23,42,.07);padding:12px;cursor:pointer;transition:transform .16s ease,border-color .16s ease,box-shadow .16s ease}.vpsnw-node-card:hover{transform:translateY(-1px);border-color:rgba(37,99,235,.38);box-shadow:0 14px 30px rgba(15,23,42,.12)}.dark .vpsnw-node-card{background:rgba(15,23,42,.76);border-color:rgba(255,255,255,.12);box-shadow:none}" +
      ".vpsnw-node-top{display:flex;align-items:flex-start;justify-content:space-between;gap:10px;margin-bottom:10px}.vpsnw-node-name{display:flex;align-items:center;gap:8px;min-width:0}.vpsnw-node-name strong{font-size:13px;line-height:1.2;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.vpsnw-node-sub{display:flex;align-items:center;gap:5px;margin-top:4px;flex-wrap:wrap}" +
      ".vpsnw-status{width:9px;height:9px;border-radius:999px;background:#ef4444;box-shadow:0 0 0 3px rgba(239,68,68,.13);flex:0 0 auto}.vpsnw-status.online{background:#22c55e;box-shadow:0 0 0 3px rgba(34,197,94,.16)}" +
      ".vpsnw-pill{display:inline-flex;align-items:center;border-radius:999px;font-size:11px;font-weight:700;line-height:1;padding:4px 7px;white-space:nowrap}.vpsnw-pill.ip{background:rgba(37,99,235,.11);color:#1d4ed8}.vpsnw-pill.band{background:rgba(20,184,166,.12);color:#0f766e}.vpsnw-pill.left{background:rgba(99,102,241,.12);color:#4f46e5}.vpsnw-pill.muted{background:rgba(100,116,139,.12);color:#64748b}.vpsnw-pill.bad{background:rgba(239,68,68,.13);color:#b91c1c}.vpsnw-pill.warn{background:rgba(245,158,11,.16);color:#b45309}.vpsnw-pill.jitter{background:rgba(168,85,247,.13);color:#7e22ce}.dark .vpsnw-pill.ip{background:rgba(96,165,250,.16);color:#bfdbfe}.dark .vpsnw-pill.band{background:rgba(45,212,191,.16);color:#99f6e4}.dark .vpsnw-pill.left{background:rgba(129,140,248,.18);color:#c7d2fe}.dark .vpsnw-pill.muted{background:rgba(148,163,184,.16);color:#cbd5e1}.dark .vpsnw-pill.bad{color:#fecaca}.dark .vpsnw-pill.warn{color:#fde68a}.dark .vpsnw-pill.jitter{color:#e9d5ff}" +
      ".vpsnw-plan-row{display:flex;flex-wrap:wrap;gap:6px;margin-bottom:10px}.vpsnw-meter-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;margin-bottom:10px}.vpsnw-meter label{display:flex;justify-content:space-between;gap:6px;color:#64748b;font-size:11px}.dark .vpsnw-meter label{color:#94a3b8}.vpsnw-meter b{color:#0f172a;font-size:11px}.dark .vpsnw-meter b{color:#f8fafc}.vpsnw-bar{height:4px;border-radius:999px;background:rgba(148,163,184,.18);overflow:hidden;margin-top:5px}.vpsnw-bar span{display:block;height:100%;border-radius:999px;background:#60a5fa}.vpsnw-bar.mem span{background:#a78bfa}.vpsnw-bar.disk span{background:#f59e0b}" +
      ".vpsnw-speed-row{display:grid;grid-template-columns:1fr 1fr;gap:8px;border-top:1px solid rgba(148,163,184,.18);padding-top:9px}.vpsnw-speed-row div{display:flex;flex-direction:column;gap:3px}.vpsnw-speed-row span{color:#64748b;font-size:11px}.dark .vpsnw-speed-row span{color:#94a3b8}.vpsnw-speed-row b{font-size:12px;color:#0f172a}.dark .vpsnw-speed-row b{color:#f8fafc}.vpsnw-up{color:#2563eb!important}.vpsnw-down{color:#16a34a!important}.vpsnw-latency-row{display:flex;flex-wrap:wrap;gap:5px;margin:9px 0 10px;padding-top:8px;border-top:1px dashed rgba(148,163,184,.24)}.vpsnw-latency-title{color:#64748b;font-size:11px;padding:4px 0}.dark .vpsnw-latency-title{color:#94a3b8}" +
      ".vpsnw-server-table-wrap{overflow:auto;border:1px solid rgba(148,163,184,.22);border-radius:8px;background:rgba(255,255,255,.82);box-shadow:0 10px 24px rgba(15,23,42,.06)}.dark .vpsnw-server-table-wrap{background:rgba(15,23,42,.72);border-color:rgba(255,255,255,.12);box-shadow:none}.vpsnw-server-table{width:100%;min-width:1040px;border-collapse:separate;border-spacing:0}.vpsnw-server-table th{position:sticky;top:0;background:rgba(248,250,252,.94);color:#64748b;font-size:12px;font-weight:800;text-align:left;padding:11px 12px;white-space:nowrap}.dark .vpsnw-server-table th{background:rgba(15,23,42,.96);color:#cbd5e1}.vpsnw-server-table td{border-top:1px solid rgba(148,163,184,.18);font-size:12px;padding:11px 12px;vertical-align:middle}.vpsnw-server-table tr{cursor:pointer}.vpsnw-server-table tr:hover td{background:rgba(37,99,235,.06)}.dark .vpsnw-server-table tr:hover td{background:rgba(96,165,250,.08)}.vpsnw-cell-node{display:flex;align-items:center;gap:8px;min-width:170px}.vpsnw-cell-node strong{display:block;max-width:210px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.vpsnw-cell-node small{display:block;color:#64748b;margin-top:3px}.dark .vpsnw-cell-node small{color:#94a3b8}.vpsnw-table-meter{display:grid;gap:4px;min-width:92px}.vpsnw-table-meter span{display:flex;justify-content:space-between;color:#64748b;font-size:11px}.dark .vpsnw-table-meter span{color:#94a3b8}" +
      ".vpsnw-server-empty{border:1px dashed rgba(148,163,184,.36);border-radius:8px;background:rgba(255,255,255,.66);color:#64748b;font-size:13px;padding:22px;text-align:center}.dark .vpsnw-server-empty{background:rgba(15,23,42,.5);color:#94a3b8;border-color:rgba(255,255,255,.16)}" +
      "@media(max-width:760px){.vpsnw-board-top{grid-template-columns:1fr}.vpsnw-board-tools{justify-content:flex-start}.vpsnw-wizard .wide input,.vpsnw-wizard input{min-width:0;width:100%}.vpsnw-wizard label{width:100%}.vpsnw-meter-grid{grid-template-columns:1fr}.vpsnw-server-grid{grid-template-columns:1fr}.vpsnw-speed-row{grid-template-columns:1fr}}";
    document.head.appendChild(style);
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return {"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[ch];
    });
  }
  function fmtBytes(value) {
    var n = Math.max(0, Number(value) || 0);
    var units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
    var i = 0;
    while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
    var text = i === 0 ? String(Math.round(n)) : (n < 10 ? n.toFixed(2) : n < 100 ? n.toFixed(1) : String(Math.round(n)));
    return text + " " + units[i];
  }
  function fmtRate(value) { return fmtBytes(value) + "/s"; }
  function fmtMs(value) { return !isFinite(value) || value <= 0 ? "-" : (value < 100 ? value.toFixed(1) : Math.round(value)) + "ms"; }
  function pct(used, total) {
    total = Number(total) || 0;
    if (!total) return 0;
    return Math.max(0, Math.min(100, (Number(used) || 0) / total * 100));
  }

  function cleanupBranding() {
    if (!document.body) return;
    if (/哪吒|Nezha|nezha/i.test(document.title || "")) document.title = "vps-netwatch";
    Array.prototype.forEach.call(document.querySelectorAll('meta[name="apple-mobile-web-app-title"],meta[property="og:title"],meta[name="application-name"]'), function (meta) {
      meta.setAttribute("content", "vps-netwatch");
    });
    Array.prototype.forEach.call(document.querySelectorAll('a[href*="github.com/naiba/nezha"],a[href*="github.com/nezhahq/nezha"]'), function (link) {
      link.href = "https://github.com/yikkrrtykj/vps-netwatch";
      link.textContent = "vps-netwatch";
    });
    var walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, {
      acceptNode: function (node) {
        var parent = node.parentElement;
        if (!parent || /SCRIPT|STYLE|TEXTAREA|INPUT/.test(parent.tagName)) return NodeFilter.FILTER_REJECT;
        return /(哪吒监控|Nezha Monitoring|nezha-dash|Nezha)/.test(node.nodeValue || "") ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP;
      }
    });
    var nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(function (node) {
      node.nodeValue = node.nodeValue
        .replace(/哪吒监控 Nezha Monitoring/g, "vps-netwatch")
        .replace(/Nezha Monitoring/g, "vps-netwatch")
        .replace(/哪吒监控/g, "vps-netwatch")
        .replace(/nezha-dash/g, "vps-netwatch")
        .replace(/Nezha/g, "vps-netwatch");
    });
  }

  function protocolTags(server) {
    var tags = [];
    if (server.ipv4) tags.push("IPv4");
    if (server.ipv6) tags.push("IPv6");
    if (!tags.length && server.ip) tags.push(String(server.ip).indexOf(":") >= 0 ? "IPv6" : "IPv4");
    return tags.length ? tags : ["IP"];
  }
  function planPills(server) {
    var remaining = server.remaining_label || "未设置剩余时间";
    var bandwidth = server.bandwidth_label || "未设置带宽";
    return '<span class="vpsnw-pill left">' + escapeHtml(remaining) + '</span><span class="vpsnw-pill band">' + escapeHtml(bandwidth) + '</span>';
  }
  function ipPills(server) {
    return protocolTags(server).map(function (tag) { return '<span class="vpsnw-pill ip">' + tag + '</span>'; }).join("");
  }
  function meter(label, value, kind) {
    var n = Math.max(0, Math.min(100, Number(value) || 0));
    return '<div class="vpsnw-meter"><label><span>' + label + '</span><b>' + n.toFixed(1) + '%</b></label><div class="vpsnw-bar ' + kind + '"><span style="width:' + n.toFixed(2) + '%"></span></div></div>';
  }
  function tableMeter(label, value, kind) {
    var n = Math.max(0, Math.min(100, Number(value) || 0));
    return '<div class="vpsnw-table-meter"><span><em>' + label + '</em><b>' + n.toFixed(1) + '%</b></span><div class="vpsnw-bar ' + kind + '"><span style="width:' + n.toFixed(2) + '%"></span></div></div>';
  }

  function seriesPoints(raw) {
    return (raw.data_points || []).filter(function (p) { return p && p.status !== 0 && p.delay > 0; });
  }
  function percentile(values, ratio) {
    if (!values.length) return 0;
    var sorted = values.slice().sort(function (a, b) { return a - b; });
    return sorted[Math.min(sorted.length - 1, Math.max(0, Math.floor((sorted.length - 1) * ratio)))];
  }
  function analyzeSeries(raw) {
    var pts = seriesPoints(raw);
    var values = pts.map(function (p) { return Number(p.delay) || 0; }).filter(function (v) { return v > 0; });
    var last = values.length ? values[values.length - 1] : 0;
    var avg = values.length ? values.reduce(function (a, b) { return a + b; }, 0) / values.length : Number(raw.avg_delay) || 0;
    var max = values.length ? Math.max.apply(null, values) : avg;
    var p50 = percentile(values, 0.5);
    var p95 = percentile(values, 0.95);
    var anomalies = [];
    if (max >= Math.max(180, avg * 2.2, avg + 80)) anomalies.push({ type: "warn", text: "峰值 " + fmtMs(max) });
    if (values.length >= 6 && p95 - p50 >= 60) anomalies.push({ type: "jitter", text: "持续抖动" });
    if (Number(raw.up_percent) > 0 && Number(raw.up_percent) < 98) anomalies.push({ type: "bad", text: "丢包 " + Math.max(0, 100 - Number(raw.up_percent)).toFixed(1) + "%" });
    return { name: raw.service_name || raw.target || "延迟", type: raw.type_name || "", isPeer: !!raw.is_peer, avg: avg, last: last || avg, max: max, up: Number(raw.up_percent) || 0, anomalies: anomalies };
  }
  function latencyByServer() {
    var map = {};
    var data = state.data || {};
    (data.servers || []).forEach(function (server) { map[String(server.id)] = { items: [], anomalies: [] }; });
    (data.series || []).forEach(function (raw) {
      var key = String(raw.server_id);
      if (!map[key]) map[key] = { items: [], anomalies: [] };
      var item = analyzeSeries(raw);
      if (!item.last && !item.avg) return;
      map[key].items.push(item);
      item.anomalies.forEach(function (a) { map[key].anomalies.push({ service: item.name, type: a.type, text: a.text }); });
    });
    Object.keys(map).forEach(function (key) {
      map[key].items.sort(function (a, b) { return (b.isPeer ? 1 : 0) - (a.isPeer ? 1 : 0) || a.name.localeCompare(b.name); });
    });
    return map;
  }
  function latencyPills(summary, limit) {
    var items = (summary && summary.items) || [];
    if (!items.length) return '<span class="vpsnw-latency-title">延迟 暂无数据</span>';
    return '<span class="vpsnw-latency-title">延迟</span>' + items.slice(0, limit || 4).map(function (item) {
      return '<span class="vpsnw-pill ' + (item.isPeer ? "warn" : "ip") + '">' + escapeHtml(item.name) + ' <b>' + fmtMs(item.last || item.avg) + '</b></span>';
    }).join("") + (items.length > (limit || 4) ? '<span class="vpsnw-pill muted">+' + (items.length - (limit || 4)) + '</span>' : '');
  }
  function anomalyPills(summary) {
    var items = (summary && summary.anomalies) || [];
    if (!items.length) return "";
    return items.slice(0, 3).map(function (item) {
      return '<span class="vpsnw-pill ' + item.type + '">' + escapeHtml(item.service) + ' ' + escapeHtml(item.text) + '</span>';
    }).join("");
  }

  function renderGrid(servers, latencyMap) {
    return '<div class="vpsnw-server-grid">' + servers.map(function (server) {
      var summary = latencyMap[String(server.id)] || {};
      var cpu = Number(server.cpu) || 0;
      var mem = pct(server.mem_used, server.mem_total);
      var disk = pct(server.disk_used, server.disk_total);
      var ip = server.ip || server.ipv4 || server.ipv6 || "";
      return '<article class="vpsnw-node-card" data-vpsnw-server-id="' + server.id + '">' +
        '<div class="vpsnw-node-top"><div class="vpsnw-node-name"><span class="vpsnw-status ' + (server.online ? "online" : "") + '"></span><div><strong>' + escapeHtml(server.name) + '</strong><div class="vpsnw-node-sub">' + ipPills(server) + (ip ? '<span class="vpsnw-pill muted">' + escapeHtml(ip) + '</span>' : '') + '</div></div></div></div>' +
        '<div class="vpsnw-plan-row">' + planPills(server) + '</div>' +
        '<div class="vpsnw-meter-grid">' + meter("CPU", cpu, "cpu") + meter("内存", mem, "mem") + meter("硬盘", disk, "disk") + '</div>' +
        '<div class="vpsnw-latency-row">' + latencyPills(summary, 4) + anomalyPills(summary) + '</div>' +
        '<div class="vpsnw-speed-row"><div><span>实时上传</span><b class="vpsnw-up">↑ ' + fmtRate(server.net_out_speed) + '</b><span>累计 ' + fmtBytes(server.net_out_transfer) + '</span></div><div><span>实时下载</span><b class="vpsnw-down">↓ ' + fmtRate(server.net_in_speed) + '</b><span>累计 ' + fmtBytes(server.net_in_transfer) + '</span></div></div>' +
      '</article>';
    }).join("") + '</div>';
  }
  function renderTable(servers, latencyMap) {
    return '<div class="vpsnw-server-table-wrap"><table class="vpsnw-server-table"><thead><tr><th>节点</th><th>状态</th><th>CPU</th><th>内存</th><th>硬盘</th><th>剩余时间</th><th>最大带宽</th><th>实时网络速率</th><th>协议</th><th>延迟</th><th>异常</th><th>总传输</th></tr></thead><tbody>' +
      servers.map(function (server) {
        var summary = latencyMap[String(server.id)] || {};
        var cpu = Number(server.cpu) || 0;
        var mem = pct(server.mem_used, server.mem_total);
        var disk = pct(server.disk_used, server.disk_total);
        var ip = server.ip || server.ipv4 || server.ipv6 || "";
        return '<tr data-vpsnw-server-id="' + server.id + '">' +
          '<td><div class="vpsnw-cell-node"><span class="vpsnw-status ' + (server.online ? "online" : "") + '"></span><div><strong>' + escapeHtml(server.name) + '</strong>' + (ip ? '<small>' + escapeHtml(ip) + '</small>' : '') + '</div></div></td>' +
          '<td>' + (server.online ? '<span class="vpsnw-pill left">在线</span>' : '<span class="vpsnw-pill muted">离线</span>') + '</td>' +
          '<td>' + tableMeter("CPU", cpu, "cpu") + '</td>' +
          '<td>' + tableMeter("内存", mem, "mem") + '</td>' +
          '<td>' + tableMeter("硬盘", disk, "disk") + '</td>' +
          '<td><span class="vpsnw-pill left">' + escapeHtml(server.remaining_label || "未设置") + '</span></td>' +
          '<td><span class="vpsnw-pill band">' + escapeHtml(server.bandwidth_label || "未设置") + '</span></td>' +
          '<td><b class="vpsnw-up">↑ ' + fmtRate(server.net_out_speed) + '</b><br><b class="vpsnw-down">↓ ' + fmtRate(server.net_in_speed) + '</b></td>' +
          '<td>' + ipPills(server) + '</td>' +
          '<td>' + latencyPills(summary, 2) + '</td>' +
          '<td>' + (anomalyPills(summary) || '<span class="vpsnw-pill muted">正常</span>') + '</td>' +
          '<td>↑ ' + fmtBytes(server.net_out_transfer) + '<br>↓ ' + fmtBytes(server.net_in_transfer) + '</td>' +
        '</tr>';
      }).join("") + '</tbody></table></div>';
  }

  function renderDiscoverList() {
    if (!state.discovered.length) return "";
    return '<div class="vpsnw-discover-list">' + state.discovered.map(function (item) {
      var meta = [item.type_name, item.network, item.process, item.chain].filter(Boolean).join(" · ");
      return '<span class="vpsnw-discover-item"><b>' + escapeHtml(item.target) + '</b>' + (meta ? '<small>' + escapeHtml(meta) + '</small>' : '') + '<button class="vpsnw-mini-btn" data-vpsnw-add-target="' + escapeHtml(item.target) + '" data-vpsnw-target-name="' + escapeHtml(item.host || item.target) + '">加入</button></span>';
    }).join("") + '</div>';
  }
  function renderWizard() {
    var discoverOpen = state.discovered.length || state.mihomoUrl ? " open" : "";
    return '<div class="vpsnw-wizard"><div class="vpsnw-wizard-row"><label class="wide"><span>目标向导</span><input id="vpsnw-board-target" placeholder="IP / 域名 / IP:端口"></label><label><span>名称</span><input id="vpsnw-board-target-name" placeholder="可留空"></label><button class="vpsnw-board-btn" data-vpsnw-add-target-form="1">加入监控</button><span class="vpsnw-board-note ' + (state.error ? "err" : "") + '">' + escapeHtml(state.note || "IP/域名创建 ICMP，IP:端口创建 TCP") + '</span></div><details class="vpsnw-discover"' + discoverOpen + '><summary>mihomo / Clash 连接发现</summary><div class="vpsnw-wizard-row"><label class="wide"><span>控制器</span><input id="vpsnw-board-mihomo-url" value="' + escapeHtml(state.mihomoUrl) + '" placeholder="http://127.0.0.1:9090"></label><label><span>密钥</span><input id="vpsnw-board-mihomo-secret" type="password" value="' + escapeHtml(state.mihomoSecret) + '" placeholder="可留空"></label><button class="vpsnw-board-btn secondary" data-vpsnw-discover="1">读取连接</button></div>' + renderDiscoverList() + '</details></div>';
  }
  function render() {
    if (!board) return;
    var data = state.data || {};
    var servers = (data.servers || []).slice().sort(function (a, b) { return String(a.name || "").localeCompare(String(b.name || "")); });
    var latencyMap = latencyByServer();
    var body = "";
    if (state.error && !servers.length) {
      body = '<div class="vpsnw-server-empty">' + escapeHtml(state.error) + '</div>';
    } else if (!servers.length) {
      body = '<div class="vpsnw-server-empty">' + (state.loading ? "正在读取服务器数据" : "暂无服务器数据") + '</div>';
    } else {
      body = state.view === "table" ? renderTable(servers, latencyMap) : renderGrid(servers, latencyMap);
    }
    board.innerHTML =
      '<div class="vpsnw-board-top"><div class="vpsnw-board-title"><span></span>服务器速览</div><div class="vpsnw-board-tools"><div class="vpsnw-board-tabs"><button type="button" data-vpsnw-view="grid" class="' + (state.view === "grid" ? "active" : "") + '">卡片</button><button type="button" data-vpsnw-view="table" class="' + (state.view === "table" ? "active" : "") + '">表格</button></div></div>' + renderWizard() + '</div>' + body;
  }

  async function loadServers() {
    if (state.loading) return;
    state.loading = true;
    render();
    try {
      var headers = {};
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      var res = await fetch("/api/v1/netwatch/latency?period=1d&_=" + Date.now(), { headers: headers, credentials: "same-origin" });
      var json = await res.json();
      if (!json.success) throw new Error(json.error || "服务器数据读取失败");
      state.data = json.data || {};
      state.error = "";
    } catch (err) {
      state.error = (err && err.message) || "服务器数据读取失败";
    } finally {
      state.loading = false;
      render();
    }
  }

  async function addTarget(target, name) {
    target = String(target || "").trim();
    name = String(name || "").trim();
    if (!target) {
      state.note = "请输入目标";
      state.error = "missing target";
      render();
      return;
    }
    state.note = "正在加入监控";
    state.error = "";
    render();
    try {
      var headers = { "Content-Type": "application/json" };
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      var res = await fetch("/api/v1/netwatch/target", {
        method: "POST",
        headers: headers,
        credentials: "same-origin",
        body: JSON.stringify({ target: target, name: name, duration: 30 })
      });
      var json = await res.json();
      if (!json.success) throw new Error(json.error || "加入失败");
      state.note = (json.data && json.data.created ? "已创建 " : "已存在 ") + (json.data ? json.data.type_name + " " + json.data.target : target);
      state.discovered = [];
      await loadServers();
    } catch (err) {
      state.note = (err && err.message) || "加入失败";
      state.error = state.note;
      render();
    }
  }

  async function discoverTargets() {
    var urlInput = board && board.querySelector("#vpsnw-board-mihomo-url");
    var secretInput = board && board.querySelector("#vpsnw-board-mihomo-secret");
    state.mihomoUrl = urlInput ? urlInput.value : state.mihomoUrl;
    state.mihomoSecret = secretInput ? secretInput.value : state.mihomoSecret;
    state.note = "正在读取连接";
    state.error = "";
    render();
    try {
      var headers = { "Content-Type": "application/json" };
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      var res = await fetch("/api/v1/netwatch/mihomo/discover", {
        method: "POST",
        headers: headers,
        credentials: "same-origin",
        body: JSON.stringify({ controller: state.mihomoUrl, secret: state.mihomoSecret, limit: 60 })
      });
      var json = await res.json();
      if (!json.success) throw new Error(json.error || "读取失败");
      state.discovered = (json.data && json.data.targets) || [];
      state.note = state.discovered.length ? "发现 " + state.discovered.length + " 个连接目标" : "没有发现可加入的连接目标";
      render();
    } catch (err) {
      state.note = (err && err.message) || "读取失败";
      state.error = state.note;
      render();
    }
  }

  function hideNativeServiceEmpty(controls) {
    try { window.localStorage.setItem("showServices", "0"); } catch (_) {}
    if (!controls || !controls.parentElement) return;
    Array.prototype.forEach.call(controls.parentElement.children, function (el) {
      if (el === controls || el.id === "vps-netwatch-server-board" || el.classList.contains("server-overview") || el.classList.contains("server-card-list") || el.classList.contains("server-inline-list")) return;
      if (serviceEmptyPattern.test(el.textContent || "")) el.style.display = "none";
    });
  }
  function ensureBoard() {
    injectStyle();
    cleanupBranding();
    var controls = document.querySelector(".server-overview-controls");
    if (!controls) return false;
    document.body.classList.add("vpsnw-server-board-ready");
    hideNativeServiceEmpty(controls);
    var nativeView = readNativeView();
    if (nativeView !== state.view && !board) state.view = nativeView;
    if (!board) {
      board = document.createElement("section");
      board.id = "vps-netwatch-server-board";
      render();
      loadServers();
      refreshTimer = window.setInterval(loadServers, 30000);
    }
    if (!board.isConnected || board.previousElementSibling !== controls) {
      controls.insertAdjacentElement("afterend", board);
    }
    return true;
  }

  document.addEventListener("click", function (event) {
    var target = event.target && event.target.closest ? event.target : event.target && event.target.parentElement;
    if (!target || !board || !board.contains(target)) return;
    var viewButton = target.closest("[data-vpsnw-view]");
    if (viewButton) {
      state.view = viewButton.getAttribute("data-vpsnw-view") || "grid";
      try { window.localStorage.setItem("inline", state.view === "table" ? "1" : "0"); } catch (_) {}
      render();
      return;
    }
    var formButton = target.closest("[data-vpsnw-add-target-form]");
    if (formButton) {
      var targetInput = board.querySelector("#vpsnw-board-target");
      var nameInput = board.querySelector("#vpsnw-board-target-name");
      addTarget(targetInput && targetInput.value, nameInput && nameInput.value);
      return;
    }
    var addButton = target.closest("[data-vpsnw-add-target]");
    if (addButton) {
      addTarget(addButton.getAttribute("data-vpsnw-add-target"), addButton.getAttribute("data-vpsnw-target-name"));
      return;
    }
    if (target.closest("[data-vpsnw-discover]")) {
      discoverTargets();
      return;
    }
    var card = target.closest("[data-vpsnw-server-id]");
    if (card && !target.closest("button,input,summary,details")) {
      var id = card.getAttribute("data-vpsnw-server-id");
      if (id) {
        try { window.sessionStorage.setItem("fromMainPage", "true"); } catch (_) {}
        window.location.href = "/server/" + encodeURIComponent(id);
      }
    }
  });

  ensureBoard();
  var observer = new MutationObserver(ensureBoard);
  observer.observe(document.documentElement, { childList: true, subtree: true });
  window.setInterval(function () {
    if (!board) ensureBoard();
    var nativeView = readNativeView();
    if (board && nativeView !== state.view) {
      state.view = nativeView;
      render();
    }
    cleanupBranding();
  }, 1500);
  document.addEventListener("visibilitychange", function () {
    if (!document.hidden) {
      ensureBoard();
      loadServers();
    }
  });
  window.addEventListener("beforeunload", function () {
    if (refreshTimer) window.clearInterval(refreshTimer);
  });
})();
</script>`
