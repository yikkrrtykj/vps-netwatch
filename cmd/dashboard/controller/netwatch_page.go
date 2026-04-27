package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/service/singleton"
)

func netwatchShouldInjectUserIndex(filePath string) bool {
	return filePath == singleton.Conf.UserTemplate+"/index.html"
}

func netwatchServeInjectedUserIndex(c *gin.Context, statusCode int, content []byte) {
	html := string(content)
	if !strings.Contains(html, "vps-netwatch-home-button") {
		if strings.Contains(html, "</body>") {
			html = strings.Replace(html, "</body>", netwatchHomeButtonScript+"</body>", 1)
		} else {
			html += netwatchHomeButtonScript
		}
	}
	c.Data(statusCode, "text/html; charset=utf-8", []byte(html))
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
    btn.className = buttonClass;
    btn.onclick = function () { window.location.href = "/dashboard/netwatch/latency"; };

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

const netwatchLatencyHTMLV2 = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>延迟 - vps-netwatch</title>
<style>
:root { color-scheme: light; --bg:#f7f8fb; --panel:#fff; --line:#d9dee8; --text:#111827; --muted:#687386; --brand:#2563eb; --brand-soft:#eaf1ff; --shadow:0 1px 2px rgba(15,23,42,.06),0 8px 20px rgba(15,23,42,.04); }
* { box-sizing: border-box; }
html, body { min-height: 100%; }
body { margin:0; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Arial,"Noto Sans SC",sans-serif; background:var(--bg); color:var(--text); }
button { font:inherit; }
.page { width:min(1220px, calc(100vw - 8px)); margin:10px auto; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:8px; box-shadow:var(--shadow); overflow:hidden; }
.tabs { display:flex; align-items:center; gap:6px; padding:14px 18px 0; border-bottom:1px solid var(--line); overflow:auto; }
.tab { display:inline-flex; align-items:center; gap:7px; min-height:42px; padding:0 10px; color:#263244; text-decoration:none; white-space:nowrap; border-bottom:2px solid transparent; font-size:14px; }
.tab svg { width:17px; height:17px; color:#6b7280; }
.tab.active { color:var(--brand); border-bottom-color:var(--brand); font-weight:700; }
.tab.active svg { color:var(--brand); }
.tab.more { margin-left:auto; }
.toolbar { display:flex; flex-wrap:wrap; align-items:center; gap:8px; padding:12px 10px 4px; }
.icon-btn, .seg, .filter { display:inline-flex; align-items:center; min-height:36px; border:1px solid var(--line); background:#fff; border-radius:7px; box-shadow:0 2px 8px rgba(15,23,42,.06); }
.icon-btn { width:36px; justify-content:center; color:#334155; cursor:pointer; }
.icon-btn svg { width:16px; height:16px; }
.date-pill { display:inline-flex; align-items:center; gap:9px; min-height:36px; padding:0 12px; border:1px solid var(--line); border-radius:7px; background:#fff; box-shadow:0 2px 8px rgba(15,23,42,.06); font-weight:700; white-space:nowrap; }
.date-pill svg { width:16px; height:16px; color:#334155; }
.seg { gap:8px; padding:5px 9px; }
.seg-label, .filter-label { color:#64748b; font-size:13px; }
.switch { position:relative; width:34px; height:20px; border:0; border-radius:999px; background:#e5e7eb; padding:0; cursor:pointer; }
.switch::after { content:""; position:absolute; top:2px; left:2px; width:16px; height:16px; border-radius:50%; background:#fff; box-shadow:0 1px 4px rgba(15,23,42,.2); transition:left .16s ease; }
.switch.tcp::after { left:16px; }
.protocol-name { min-width:38px; font-weight:700; }
.filter { gap:9px; padding:5px 9px; }
.chip { display:inline-flex; align-items:center; gap:6px; min-height:24px; border:0; background:transparent; color:#1f2937; cursor:pointer; padding:2px 4px; white-space:nowrap; }
.chip .box { width:16px; height:16px; border-radius:4px; border:1px solid #cbd5e1; background:#fff; display:inline-flex; align-items:center; justify-content:center; }
.chip.active .box { background:#111827; border-color:#111827; }
.chip.active .box::after { content:""; width:8px; height:5px; border-left:2px solid #fff; border-bottom:2px solid #fff; transform:rotate(-45deg) translate(1px,-1px); }
.clear { color:var(--brand); border:0; background:#eef5ff; border-radius:6px; min-height:24px; padding:0 8px; cursor:pointer; }
.legend { display:flex; flex-wrap:wrap; justify-content:center; gap:10px 18px; min-height:38px; padding:10px 18px 4px; font-size:13px; color:#1f2937; }
.legend-item { display:inline-flex; align-items:center; gap:6px; white-space:nowrap; }
.dot { width:11px; height:11px; border-radius:50%; display:inline-block; }
.chart-wrap { position:relative; height:480px; padding:6px 18px 14px; }
canvas { display:block; width:100%; height:100%; }
.tooltip { position:absolute; display:none; min-width:160px; max-width:260px; padding:10px 12px; border:1px solid #d7deea; border-radius:6px; background:rgba(255,255,255,.96); box-shadow:0 8px 24px rgba(15,23,42,.18); pointer-events:none; font-size:13px; color:#1f2937; z-index:4; }
.tooltip-time { margin-bottom:6px; color:#334155; }
.tooltip-row { display:flex; align-items:center; justify-content:space-between; gap:14px; line-height:1.7; }
.tooltip-name { display:flex; align-items:center; gap:6px; }
.empty, .error { display:none; margin:12px 16px; padding:14px; border-radius:8px; font-size:14px; }
.empty { color:#64748b; background:#f8fafc; border:1px dashed #cbd5e1; }
.error { color:#991b1b; background:#fff1f2; border:1px solid #fecdd3; }
@media (max-width: 780px) { .page { width:calc(100vw - 16px); margin:8px auto; } .tabs { padding-left:10px; } .tab.more { margin-left:0; } .toolbar { align-items:stretch; } .filter { max-width:100%; overflow:auto; } .date-pill { flex:1; justify-content:center; } .chart-wrap { height:390px; padding-left:8px; padding-right:8px; } }
</style>
</head>
<body>
<main class="page">
  <section class="panel">
    <nav class="tabs" aria-label="vps-netwatch">
      <a class="tab" href="/"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M3 4.5A2.5 2.5 0 0 1 5.5 2h9A2.5 2.5 0 0 1 17 4.5v11A2.5 2.5 0 0 1 14.5 18h-9A2.5 2.5 0 0 1 3 15.5v-11Zm2.5-.8a.8.8 0 0 0-.8.8v11c0 .44.36.8.8.8h9a.8.8 0 0 0 .8-.8v-11a.8.8 0 0 0-.8-.8h-9Z"/><path d="M6 6h8v1.5H6V6Zm0 3.25h8v1.5H6v-1.5Zm0 3.25h5v1.5H6v-1.5Z"/></svg>摘要</a>
      <span class="tab"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M7 2h6v2h2a1 1 0 0 1 1 1v2h2v6h-2v2a1 1 0 0 1-1 1h-2v2H7v-2H5a1 1 0 0 1-1-1v-2H2V7h2V5a1 1 0 0 1 1-1h2V2Zm-1 4v8h8V6H6Z"/></svg>硬件</span>
      <span class="tab"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M10 3a7 7 0 1 0 6.64 4.78l-1.67.9a5.1 5.1 0 1 1-3.84-3.65L10 9.7l1.82.44 1.5-6.16A6.9 6.9 0 0 0 10 3Z"/></svg>速率</span>
      <span class="tab"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M4 3h12a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Zm2 4v6h2V7H6Zm3 0v6h2V7H9Zm3 0v6h2V7h-2Z"/></svg>IP 质量</span>
      <a class="tab active" href="/dashboard/netwatch/latency"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M3 12.6a1 1 0 0 1 1-1h1.9l2.2-5.7a1 1 0 0 1 1.86 0l2.63 6.84 1.55-3.1a1 1 0 0 1 .9-.55H17a1 1 0 1 1 0 2h-1.34l-2.36 4.72a1 1 0 0 1-1.83-.08L9.03 9.37l-1.5 3.88a1 1 0 0 1-.93.64H4a1 1 0 0 1-1-1.29Z"/></svg>延迟</a>
      <span class="tab"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M10 2a8 8 0 0 1 7.93 7H16a6 6 0 1 0-2.02 4.49l-1.36-1.36A4 4 0 1 1 14 9h-2.4l3.2 3.2L18 9h-2.06A8 8 0 1 1 10 2Z"/></svg>Ping</span>
      <span class="tab"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M5 4a3 3 0 1 1 2.83 4H7v3h6v-1.17a3 3 0 1 1 2 0V12a1 1 0 0 1-1 1H7v1.17a3 3 0 1 1-2 0V7.83A3 3 0 0 1 5 4Z"/></svg>路由</span>
      <span class="tab"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M4 4a2 2 0 0 1 2-2h2v16H6a2 2 0 0 1-2-2V4Zm8-2h2a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-2V2Z"/></svg>BGP</span>
      <span class="tab"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M3 10h3l2-6 4 12 2-6h3v2h-1.56l-2.49 7.48a1 1 0 0 1-1.9 0L8.02 10.4l-.07.2A1 1 0 0 1 7 11.3H3V10Z"/></svg>状态</span>
      <span class="tab more">更多 <svg viewBox="0 0 20 20" fill="currentColor"><path d="M7.2 4.2a1 1 0 0 1 1.4 0l5.1 5.1a1 1 0 0 1 0 1.4l-5.1 5.1a1 1 0 1 1-1.4-1.4l4.4-4.4-4.4-4.4a1 1 0 0 1 0-1.4Z"/></svg></span>
    </nav>

    <div class="toolbar">
      <button class="icon-btn" id="prevBtn" title="上一段"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M12.8 4.2a1 1 0 0 1 0 1.4L8.4 10l4.4 4.4a1 1 0 1 1-1.4 1.4l-5.1-5.1a1 1 0 0 1 0-1.4l5.1-5.1a1 1 0 0 1 1.4 0Z"/></svg></button>
      <div class="date-pill"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M6 2a1 1 0 0 1 1 1v1h6V3a1 1 0 1 1 2 0v1h1a2 2 0 0 1 2 2v9.5A2.5 2.5 0 0 1 15.5 18h-11A2.5 2.5 0 0 1 2 15.5V6a2 2 0 0 1 2-2h1V3a1 1 0 0 1 1-1Zm10 7H4v6.5c0 .28.22.5.5.5h11a.5.5 0 0 0 .5-.5V9Z"/></svg><span id="rangeLabel">加载中</span></div>
      <button class="icon-btn" id="nextBtn" title="下一段"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M7.2 15.8a1 1 0 0 1 0-1.4l4.4-4.4-4.4-4.4a1 1 0 0 1 1.4-1.4l5.1 5.1a1 1 0 0 1 0 1.4l-5.1 5.1a1 1 0 0 1-1.4 0Z"/></svg></button>

      <div class="seg"><span class="seg-label">协议</span><span class="protocol-name" id="leftProtocol">ICMP</span><button class="switch" id="protocolSwitch" title="切换 ICMP/TCP"></button><span class="protocol-name">TCP</span></div>
      <div class="filter"><button class="clear" id="carrierAll">全选</button><div id="carrierFilters"></div></div>
      <div class="filter"><button class="clear" id="cityAll">清空</button><div id="cityFilters"></div></div>
    </div>

    <div class="error" id="errorBox"></div>
    <div class="empty" id="emptyBox">暂无延迟数据。请在后台服务页添加上海电信、上海联通的 ICMP-Ping 或 TCP-Ping 监控任务，并等待一次采集。</div>
    <div class="legend" id="legend"></div>
    <div class="chart-wrap" id="chartWrap">
      <canvas id="chart"></canvas>
      <div class="tooltip" id="tooltip"></div>
    </div>
  </section>
</main>
<script>
(function () {
  var colors = ["#5276d8", "#f2c14e", "#ef5b5b", "#43a772", "#8b5fb7", "#e16ac5", "#69bde7", "#64748b"];
  var knownCities = ["上海", "北京", "广州", "深圳", "香港", "东京", "新加坡", "洛杉矶"];
  var periodMs = { "1d": 86400000, "7d": 604800000, "30d": 2592000000 };
  var state = { period: "1d", protocol: "ICMP", carriers: new Set(["电信", "移动", "联通"]), cities: new Set(["上海", "北京", "广州"]), data: null, domain: null, view: null, hover: null };
  var canvas = document.getElementById("chart");
  var wrap = document.getElementById("chartWrap");
  var ctx = canvas.getContext("2d");
  var tooltip = document.getElementById("tooltip");
  var lastPlot = null;

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
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return {"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[ch];
    });
  }

  function pad(n) { return String(n).padStart(2, "0"); }
  function fmtDate(ts) { var d = new Date(ts); return d.getFullYear() + "年" + (d.getMonth()+1) + "月" + d.getDate() + "日"; }
  function fmtTime(ts) { var d = new Date(ts); return d.getFullYear() + "-" + pad(d.getMonth()+1) + "-" + pad(d.getDate()) + " " + pad(d.getHours()) + ":" + pad(d.getMinutes()) + ":" + pad(d.getSeconds()); }
  function fmtAxis(ts) { var d = new Date(ts); return pad(d.getHours()) + ":" + pad(d.getMinutes()); }
  function fmtMs(v) { return !isFinite(v) ? "-" : (v < 100 ? v.toFixed(2) : Math.round(v)) + "ms"; }
  function carrierOf(name) { if (name.indexOf("电信") >= 0) return "电信"; if (name.indexOf("联通") >= 0) return "联通"; if (name.indexOf("移动") >= 0) return "移动"; return "其他"; }
  function cityOf(name) { for (var i = 0; i < knownCities.length; i++) { if (name.indexOf(knownCities[i]) >= 0) return knownCities[i]; } return "其他"; }

  function servicePasses(service) {
    var carrier = carrierOf(service.name || "");
    var city = cityOf(service.name || "");
    if (state.protocol !== "all" && service.type_name !== state.protocol) return false;
    if (state.carriers.size && !state.carriers.has(carrier)) return false;
    if (state.cities.size && !state.cities.has(city)) return false;
    return true;
  }

  function aggregateSeries() {
    var data = state.data;
    if (!data) return [];
    var serviceById = {};
    (data.services || []).forEach(function (s) { serviceById[String(s.id)] = s; });
    var grouped = {};
    (data.series || []).forEach(function (raw) {
      var service = serviceById[String(raw.service_id)] || raw;
      if (!servicePasses(service)) return;
      var key = String(raw.service_id);
      if (!grouped[key]) {
        grouped[key] = { service_id: raw.service_id, name: raw.service_name, type_name: raw.type_name, target: raw.target, display_index: raw.display_index || 0, buckets: {}, total: 0, count: 0 };
      }
      (raw.data_points || []).forEach(function (p) {
        if (!p || p.status === 0 || !(p.delay > 0)) return;
        var bucket = Math.floor(p.ts / 60000) * 60000;
        if (!grouped[key].buckets[bucket]) grouped[key].buckets[bucket] = [];
        grouped[key].buckets[bucket].push(p.delay);
        grouped[key].total += p.delay;
        grouped[key].count += 1;
      });
    });
    return Object.keys(grouped).map(function (key) {
      var g = grouped[key];
      var points = Object.keys(g.buckets).map(function (ts) {
        var values = g.buckets[ts];
        var sum = values.reduce(function (a, b) { return a + b; }, 0);
        return { ts: Number(ts), delay: sum / values.length };
      }).sort(function (a, b) { return a.ts - b.ts; });
      return { key: key, name: g.name, type_name: g.type_name, target: g.target, display_index: g.display_index, avg: g.count ? g.total / g.count : 0, points: points, carrier: carrierOf(g.name), city: cityOf(g.name) };
    }).filter(function (s) { return s.points.length; }).sort(function (a, b) {
      if (a.display_index !== b.display_index) return b.display_index - a.display_index;
      return a.name.localeCompare(b.name);
    });
  }

  function allServiceOptions(kind) {
    var values = new Set();
    ((state.data && state.data.services) || []).forEach(function (s) { values.add(kind === "carrier" ? carrierOf(s.name || "") : cityOf(s.name || "")); });
    var order = kind === "carrier" ? ["电信", "移动", "联通", "其他"] : knownCities.concat(["其他"]);
    return order.filter(function (item) { return values.has(item); });
  }

  function renderFilter(id, set, items) {
    var box = document.getElementById(id);
    box.innerHTML = "";
    items.forEach(function (item) {
      var btn = document.createElement("button");
      btn.className = "chip" + (set.has(item) ? " active" : "");
      btn.innerHTML = '<span class="box"></span><span>' + escapeHtml(item) + '</span>';
      btn.onclick = function () {
        if (set.has(item)) set.delete(item); else set.add(item);
        renderAll();
      };
      box.appendChild(btn);
    });
  }

  function syncFilters() {
    renderFilter("carrierFilters", state.carriers, allServiceOptions("carrier"));
    renderFilter("cityFilters", state.cities, allServiceOptions("city"));
    var sw = document.getElementById("protocolSwitch");
    sw.classList.toggle("tcp", state.protocol === "TCP");
    document.getElementById("leftProtocol").style.color = state.protocol === "TCP" ? "#64748b" : "#111827";
  }

  function updateDomain() {
    var end = (state.data && state.data.generated_at) || Date.now();
    var start = end - (periodMs[state.period] || periodMs["1d"]);
    state.domain = { start: start, end: end };
    if (!state.view || state.view.start < start || state.view.end > end) {
      state.view = { start: start, end: end };
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
    start = Math.max(state.domain.start, start);
    end = Math.min(state.domain.end, end);
    state.view = { start: start, end: end };
  }

  function updateRangeLabel() {
    document.getElementById("rangeLabel").textContent = fmtDate(state.view.start) + " - " + fmtDate(state.view.end);
  }

  function visiblePoints(series) {
    return series.points.filter(function (p) { return p.ts >= state.view.start && p.ts <= state.view.end; });
  }

  function renderLegend(series) {
    var legend = document.getElementById("legend");
    legend.innerHTML = "";
    series.forEach(function (s, index) {
      var visible = visiblePoints(s);
      var avg = visible.length ? visible.reduce(function (sum, p) { return sum + p.delay; }, 0) / visible.length : s.avg;
      var item = document.createElement("span");
      item.className = "legend-item";
      item.innerHTML = '<span class="dot" style="background:' + colors[index % colors.length] + '"></span>' + escapeHtml(s.name) + ' ' + fmtMs(avg);
      legend.appendChild(item);
    });
  }

  function drawChart() {
    var series = aggregateSeries();
    var empty = document.getElementById("emptyBox");
    empty.style.display = series.length ? "none" : "block";
    wrap.style.display = series.length ? "block" : "none";
    renderLegend(series);
    updateRangeLabel();
    if (!series.length) return;

    var rect = canvas.getBoundingClientRect();
    var dpr = window.devicePixelRatio || 1;
    canvas.width = Math.max(1, Math.floor(rect.width * dpr));
    canvas.height = Math.max(1, Math.floor(rect.height * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, rect.width, rect.height);

    var pad = { left: 42, right: 20, top: 22, bottom: 78 };
    var overviewH = 34;
    var plot = { left: pad.left, top: pad.top, width: rect.width - pad.left - pad.right, height: rect.height - pad.top - pad.bottom };
    var overview = { left: pad.left, top: rect.height - 46, width: rect.width - pad.left - pad.right, height: overviewH };
    lastPlot = plot;

    var visible = [];
    var all = [];
    series.forEach(function (s) {
      visiblePoints(s).forEach(function (p) { visible.push(p); });
      s.points.forEach(function (p) { all.push(p); });
    });
    var maxY = Math.max.apply(null, (visible.length ? visible : all).map(function (p) { return p.delay; }));
    maxY = Math.max(10, Math.ceil(maxY * 1.18 / 10) * 10);
    function x(ts) { return plot.left + (ts - state.view.start) / (state.view.end - state.view.start) * plot.width; }
    function y(v) { return plot.top + plot.height - v / maxY * plot.height; }
    function ox(ts) { return overview.left + (ts - state.domain.start) / (state.domain.end - state.domain.start) * overview.width; }
    function oy(v) { return overview.top + overview.height - v / maxY * overview.height; }

    ctx.strokeStyle = "#dbe2ee";
    ctx.lineWidth = 1;
    ctx.fillStyle = "#334155";
    ctx.font = "12px -apple-system,BlinkMacSystemFont,Segoe UI,Arial";
    for (var gy = 0; gy <= 4; gy++) {
      var value = maxY * gy / 4;
      var yy = y(value);
      ctx.beginPath(); ctx.moveTo(plot.left, yy); ctx.lineTo(plot.left + plot.width, yy); ctx.stroke();
      ctx.fillText(Math.round(value) + "ms", 0, yy + 4);
    }
    for (var gx = 0; gx <= 6; gx++) {
      var ts = state.view.start + (state.view.end - state.view.start) * gx / 6;
      var xx = plot.left + plot.width * gx / 6;
      ctx.fillText(gx === 6 ? fmtAxis(ts) : fmtAxis(ts), xx - 14, plot.top + plot.height + 18);
    }

    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
      if (!pts.length) return;
      ctx.strokeStyle = colors[index % colors.length];
      ctx.lineWidth = 1.8;
      ctx.beginPath();
      pts.forEach(function (p, i) { if (i === 0) ctx.moveTo(x(p.ts), y(p.delay)); else ctx.lineTo(x(p.ts), y(p.delay)); });
      ctx.stroke();
    });

    ctx.fillStyle = "#eef2f8";
    ctx.fillRect(overview.left, overview.top, overview.width, overview.height);
    ctx.strokeStyle = "#d8e0ee";
    ctx.strokeRect(overview.left, overview.top, overview.width, overview.height);
    series.forEach(function (s, index) {
      ctx.strokeStyle = colors[index % colors.length];
      ctx.globalAlpha = 0.35;
      ctx.lineWidth = 1;
      ctx.beginPath();
      s.points.forEach(function (p, i) { if (i === 0) ctx.moveTo(ox(p.ts), oy(p.delay)); else ctx.lineTo(ox(p.ts), oy(p.delay)); });
      ctx.stroke();
      ctx.globalAlpha = 1;
    });
    var sx = ox(state.view.start);
    var ex = ox(state.view.end);
    ctx.fillStyle = "rgba(194, 205, 226, .45)";
    ctx.fillRect(sx, overview.top, ex - sx, overview.height);
    ctx.strokeStyle = "#9fb1cf";
    ctx.strokeRect(sx, overview.top, ex - sx, overview.height);

    if (state.hover) drawHover(series, plot, x, y);
  }

  function drawHover(series, plot, x, y) {
    var hoverTs = state.view.start + (state.hover.x - plot.left) / plot.width * (state.view.end - state.view.start);
    if (state.hover.x < plot.left || state.hover.x > plot.left + plot.width || state.hover.y < plot.top || state.hover.y > plot.top + plot.height) {
      tooltip.style.display = "none";
      return;
    }
    var items = [];
    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
      if (!pts.length) return;
      var nearest = pts[0];
      for (var i = 1; i < pts.length; i++) {
        if (Math.abs(pts[i].ts - hoverTs) < Math.abs(nearest.ts - hoverTs)) nearest = pts[i];
      }
      items.push({ s: s, p: nearest, color: colors[index % colors.length] });
    });
    if (!items.length) { tooltip.style.display = "none"; return; }
    var lineX = x(items[0].p.ts);
    ctx.strokeStyle = "#b9c5d8";
    ctx.setLineDash([4, 3]);
    ctx.beginPath(); ctx.moveTo(lineX, plot.top); ctx.lineTo(lineX, plot.top + plot.height); ctx.stroke();
    ctx.setLineDash([]);
    items.forEach(function (item) {
      ctx.fillStyle = item.color;
      ctx.beginPath(); ctx.arc(x(item.p.ts), y(item.p.delay), 4, 0, Math.PI * 2); ctx.fill();
      ctx.strokeStyle = "#fff"; ctx.lineWidth = 1.5; ctx.stroke();
    });

    tooltip.innerHTML = '<div class="tooltip-time">' + fmtTime(items[0].p.ts) + '</div>' + items.map(function (item) {
      return '<div class="tooltip-row"><span class="tooltip-name"><span class="dot" style="background:' + item.color + '"></span>' + escapeHtml(item.s.name) + '</span><strong>' + fmtMs(item.p.delay) + '</strong></div>';
    }).join("");
    tooltip.style.display = "block";
    var left = Math.min(wrap.clientWidth - tooltip.offsetWidth - 10, Math.max(10, state.hover.x + 18));
    var top = Math.min(wrap.clientHeight - tooltip.offsetHeight - 10, Math.max(10, state.hover.y - 16));
    tooltip.style.left = left + "px";
    tooltip.style.top = top + "px";
  }

  function renderAll() {
    syncFilters();
    drawChart();
  }

  async function load() {
    var error = document.getElementById("errorBox");
    error.style.display = "none";
    try {
      var headers = {};
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      var res = await fetch("/api/v1/netwatch/latency?period=" + encodeURIComponent(state.period), { headers: headers, credentials: "same-origin" });
      var body = await res.json();
      if (!body.success) throw new Error(body.error || "请求失败");
      state.data = body.data;
      state.view = null;
      updateDomain();
      renderAll();
    } catch (err) {
      error.textContent = err.message || String(err);
      error.style.display = "block";
    }
  }

  document.getElementById("protocolSwitch").onclick = function () {
    state.protocol = state.protocol === "TCP" ? "ICMP" : "TCP";
    renderAll();
  };
  document.getElementById("carrierAll").onclick = function () {
    allServiceOptions("carrier").forEach(function (item) { state.carriers.add(item); });
    renderAll();
  };
  document.getElementById("cityAll").onclick = function () {
    state.cities.clear();
    renderAll();
  };
  document.getElementById("prevBtn").onclick = function () {
    var range = state.view.end - state.view.start;
    clampView(state.view.start - range * 0.85, state.view.end - range * 0.85);
    drawChart();
  };
  document.getElementById("nextBtn").onclick = function () {
    var range = state.view.end - state.view.start;
    clampView(state.view.start + range * 0.85, state.view.end + range * 0.85);
    drawChart();
  };
  canvas.addEventListener("wheel", function (event) {
    if (!state.domain || !state.view || !lastPlot) return;
    event.preventDefault();
    var rect = canvas.getBoundingClientRect();
    var x = event.clientX - rect.left;
    var center = state.view.start + Math.max(0, Math.min(1, (x - lastPlot.left) / lastPlot.width)) * (state.view.end - state.view.start);
    var factor = event.deltaY > 0 ? 1.22 : 0.82;
    var range = (state.view.end - state.view.start) * factor;
    clampView(center - range / 2, center + range / 2);
    drawChart();
  }, { passive: false });
  canvas.addEventListener("mousemove", function (event) {
    var rect = canvas.getBoundingClientRect();
    state.hover = { x: event.clientX - rect.left, y: event.clientY - rect.top };
    drawChart();
  });
  canvas.addEventListener("mouseleave", function () {
    state.hover = null;
    tooltip.style.display = "none";
    drawChart();
  });
  canvas.addEventListener("dblclick", function () {
    state.view = { start: state.domain.start, end: state.domain.end };
    drawChart();
  });
  window.addEventListener("resize", drawChart);
  load();
})();
</script>
</body>
</html>`
