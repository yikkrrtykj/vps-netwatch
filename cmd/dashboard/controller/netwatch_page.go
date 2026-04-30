package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/service/singleton"
)

func netwatchShouldInjectUserIndex(filePath string) bool {
	return filePath == singleton.Conf.UserTemplate+"/index.html"
}

func netwatchShouldRewriteUserAsset(filePath string) bool {
	return filePath == singleton.Conf.UserTemplate+"/manifest.json"
}

func netwatchCanServeUserIndex(content []byte) bool {
	html := strings.TrimSpace(string(content))
	if html == "" {
		return false
	}
	lower := strings.ToLower(html)
	return strings.Contains(lower, "<!doctype") ||
		strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<body") ||
		strings.Contains(lower, "<div") ||
		strings.Contains(lower, "<script")
}

func netwatchServeInjectedUserIndex(c *gin.Context, statusCode int, content []byte) {
	html := netwatchApplyUserBranding(string(content))
	c.Header("Cache-Control", "no-store, max-age=0")
	c.Data(statusCode, "text/html; charset=utf-8", []byte(html))
}

func netwatchServeBrandedUserAsset(c *gin.Context, statusCode int, filePath string, content []byte) {
	contentType := "text/plain; charset=utf-8"
	if strings.HasSuffix(filePath, ".json") {
		contentType = "application/manifest+json; charset=utf-8"
	} else if strings.HasSuffix(filePath, ".js") {
		contentType = "application/javascript; charset=utf-8"
	}
	c.Header("Cache-Control", "no-store, max-age=0")
	c.Data(statusCode, contentType, []byte(netwatchApplyUserBranding(string(content))))
}

func netwatchApplyUserBranding(content string) string {
	return strings.NewReplacer(
		"https://github.com/naiba/nezha", "https://github.com/yikkrrtykj/vps-netwatch",
		"https://github.com/hamster1963/nezha-dash-v1/commit/e4ba96e", "https://github.com/yikkrrtykj/vps-netwatch",
		"https://github.com/hamster1963/nezha-dash", "https://github.com/yikkrrtykj/vps-netwatch",
		"https://github.com/nezhahq/nezha", "https://github.com/yikkrrtykj/vps-netwatch",
		"哪吒监控 Nezha Monitoring", "vps-netwatch",
		"Nezha Monitoring", "vps-netwatch",
		"哪吒监控", "vps-netwatch",
		"nezha-dash", "vps-netwatch",
		"NEZHA", "VPS-NETWATCH",
		"Nezha", "vps-netwatch",
	).Replace(content)
}

const netwatchHomeButtonScript = `<script id="vps-netwatch-home-button">
(function () {
  var marker = "data-vps-netwatch-latency";
  var buttonClass = "inset-shadow-2xs inset-shadow-white/20 flex cursor-pointer flex-col items-center gap-0 rounded-[50px] bg-blue-100 p-2.5 text-blue-600 transition-all dark:bg-blue-900 dark:text-blue-100";
  var activeClass = "inset-shadow-black/20 bg-blue-600 text-white ring-2 ring-blue-300/70 dark:bg-blue-600 dark:text-white dark:ring-blue-300/70";
  var colors = ["#5276d8", "#f2c14e", "#73bf69", "#e4576b", "#69bde7", "#8b5cf6", "#14b8a6", "#f97316", "#ec4899", "#64748b"];
  var state = { visible: false, loaded: false, loading: false, data: null, domain: null, view: null, hover: null, date: "", protocol: "ICMP", showExtremes: false, showAverage: false, selectedServerId: "", peerTargetServerId: "", peerSaving: false, refreshTimer: 0, peerTimer: 0, peerWaitUntil: 0 };
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
      ".vpsnw-head{display:grid;grid-template-columns:max-content minmax(0,1fr);align-items:center;gap:8px 10px;padding:12px 16px;border-bottom:1px solid rgba(148,163,184,.22)}" +
      ".vpsnw-title{grid-column:1;grid-row:1;font-weight:800;font-size:14px;white-space:nowrap}" +
      ".vpsnw-server-tabs{grid-column:2;grid-row:1;display:flex;flex-wrap:wrap;justify-content:flex-start;gap:6px;min-width:0}.vpsnw-server-btn{border:1px solid rgba(148,163,184,.35);border-radius:999px;background:rgba(248,250,252,.9);color:#334155;cursor:pointer;font-size:12px;line-height:1;padding:5px 10px;white-space:nowrap}.vpsnw-server-btn.active{border-color:#2563eb;background:#2563eb;color:#fff}.dark .vpsnw-server-btn{background:rgba(255,255,255,.08);border-color:rgba(255,255,255,.16);color:#e2e8f0}.dark .vpsnw-server-btn.active{background:#dbeafe;border-color:#dbeafe;color:#1d4ed8}" +
      ".vpsnw-peer-row{grid-column:2;grid-row:2;display:flex;flex-wrap:wrap;align-items:center;justify-content:flex-start;gap:6px;min-width:0;font-size:12px;color:#64748b}.dark .vpsnw-peer-row{color:#94a3b8}.vpsnw-peer-label{grid-column:1;grid-row:2;font-size:12px;white-space:nowrap;color:#64748b}.dark .vpsnw-peer-label{color:#94a3b8}.vpsnw-peer-tabs{display:flex;flex-wrap:wrap;justify-content:flex-start;gap:6px;min-width:0}.vpsnw-peer-note{min-width:48px;color:#2563eb}.dark .vpsnw-peer-note{color:#93c5fd}.vpsnw-peer-btn{border:1px solid rgba(148,163,184,.35);border-radius:7px;background:rgba(248,250,252,.9);color:#334155;cursor:pointer;font-size:12px;line-height:1;padding:5px 9px;white-space:nowrap}.vpsnw-peer-btn.active{border-color:#0f172a;background:#0f172a;color:#fff}.vpsnw-peer-btn:disabled{cursor:not-allowed;opacity:.58}.dark .vpsnw-peer-btn{background:rgba(255,255,255,.08);border-color:rgba(255,255,255,.16);color:#e2e8f0}.dark .vpsnw-peer-btn.active{background:#f8fafc;border-color:#f8fafc;color:#0f172a}" +
      ".vpsnw-tools{display:flex;flex-wrap:wrap;align-items:center;gap:8px;padding:8px 16px;border-bottom:1px solid rgba(148,163,184,.18)}.vpsnw-tool,.vpsnw-date,.vpsnw-protocol{display:inline-flex;align-items:center;gap:7px;min-height:30px;border:1px solid rgba(148,163,184,.32);border-radius:7px;background:rgba(248,250,252,.88);color:#334155;font-size:12px;line-height:1;padding:4px 8px}.dark .vpsnw-tool,.dark .vpsnw-date,.dark .vpsnw-protocol{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14);color:#e2e8f0}.vpsnw-date input{border:0;outline:0;background:transparent;color:inherit;font:inherit;font-weight:700;min-width:116px}.vpsnw-icon-btn{display:inline-flex;align-items:center;justify-content:center;width:30px;height:30px;border:1px solid rgba(148,163,184,.32);border-radius:7px;background:rgba(248,250,252,.88);color:#334155;cursor:pointer}.dark .vpsnw-icon-btn{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14);color:#e2e8f0}.vpsnw-icon-btn svg{width:15px;height:15px}.vpsnw-chip{border:1px solid rgba(148,163,184,.32);border-radius:6px;background:transparent;color:inherit;cursor:pointer;font:inherit;line-height:1;padding:5px 8px}.vpsnw-chip.active{border-color:#2563eb;background:#2563eb;color:#fff}.dark .vpsnw-chip.active{border-color:#dbeafe;background:#dbeafe;color:#1d4ed8}.vpsnw-muted{color:#64748b}.dark .vpsnw-muted{color:#94a3b8}" +
      ".vpsnw-targets{display:grid;gap:8px;padding:10px 16px;border-bottom:1px solid rgba(148,163,184,.18)}.vpsnw-target-row{display:flex;flex-wrap:wrap;align-items:center;gap:8px}.vpsnw-target-row label{display:inline-flex;align-items:center;gap:7px;min-height:31px;border:1px solid rgba(148,163,184,.32);border-radius:7px;background:rgba(248,250,252,.88);color:#64748b;font-size:12px;padding:4px 8px}.dark .vpsnw-target-row label{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14);color:#94a3b8}.vpsnw-target-row input{min-width:150px;border:0;outline:0;background:transparent;color:#0f172a;font:inherit}.dark .vpsnw-target-row input{color:#f8fafc}.vpsnw-target-main input{min-width:220px}.vpsnw-action-btn{border:1px solid rgba(37,99,235,.35);border-radius:7px;background:#2563eb;color:#fff;cursor:pointer;font-size:12px;line-height:1;padding:8px 11px}.vpsnw-action-btn.secondary{background:rgba(248,250,252,.88);color:#1d4ed8}.dark .vpsnw-action-btn.secondary{background:rgba(255,255,255,.08);color:#bfdbfe}.vpsnw-target-note{font-size:12px;color:#2563eb}.dark .vpsnw-target-note{color:#93c5fd}.vpsnw-discover summary{cursor:pointer;font-size:12px;color:#64748b;user-select:none}.dark .vpsnw-discover summary{color:#94a3b8}.vpsnw-discover-list{display:flex;flex-wrap:wrap;gap:6px;margin-top:8px}.vpsnw-discover-item{display:inline-flex;align-items:center;gap:6px;border:1px solid rgba(148,163,184,.26);border-radius:999px;background:rgba(248,250,252,.82);font-size:12px;padding:5px 7px}.dark .vpsnw-discover-item{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14)}.vpsnw-mini-btn{border:0;border-radius:999px;background:#0f172a;color:#fff;cursor:pointer;font-size:11px;line-height:1;padding:5px 7px}.dark .vpsnw-mini-btn{background:#f8fafc;color:#0f172a}" +
      ".vpsnw-overview{border-bottom:1px solid rgba(148,163,184,.18);padding:10px 14px 12px}.vpsnw-overview-head{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:8px;font-size:12px;color:#64748b}.dark .vpsnw-overview-head{color:#94a3b8}.vpsnw-overview-head b{font-size:13px;color:#0f172a}.dark .vpsnw-overview-head b{color:#f8fafc}.vpsnw-overview-table{display:grid;gap:7px;max-height:300px;overflow:auto;padding-right:2px}.vpsnw-overview-row{display:grid;grid-template-columns:minmax(190px,1fr) minmax(220px,.9fr);align-items:start;gap:7px 12px;border:1px solid rgba(148,163,184,.2);border-radius:10px;background:rgba(248,250,252,.72);padding:9px 11px;cursor:pointer}.dark .vpsnw-overview-row{background:rgba(255,255,255,.055);border-color:rgba(255,255,255,.11)}.vpsnw-overview-row.active{border-color:#2563eb;background:rgba(219,234,254,.72)}.dark .vpsnw-overview-row.active{border-color:#93c5fd;background:rgba(37,99,235,.18)}.vpsnw-overview-main{display:flex;flex-direction:column;gap:6px;min-width:0}.vpsnw-overview-name{display:flex;align-items:center;gap:8px;min-width:0}.vpsnw-overview-name strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:13px}.vpsnw-status-dot{width:9px;height:9px;border-radius:999px;background:#ef4444;box-shadow:0 0 0 3px rgba(239,68,68,.12);flex:0 0 auto}.vpsnw-status-dot.online{background:#22c55e;box-shadow:0 0 0 3px rgba(34,197,94,.16)}.vpsnw-plan-tags{display:flex;flex-wrap:wrap;gap:5px;min-height:20px}.vpsnw-band-label,.vpsnw-remaining-label{border-radius:999px;font-style:normal;font-size:11px;line-height:1;padding:4px 7px;white-space:nowrap}.vpsnw-band-label{border:1px solid rgba(20,184,166,.28);background:rgba(20,184,166,.12);color:#0f766e}.vpsnw-remaining-label{border:1px solid rgba(99,102,241,.25);background:rgba(99,102,241,.12);color:#4f46e5}.dark .vpsnw-band-label{background:rgba(45,212,191,.16);color:#99f6e4}.dark .vpsnw-remaining-label{background:rgba(129,140,248,.16);color:#c7d2fe}.vpsnw-overview-band{display:flex;flex-wrap:wrap;justify-content:flex-end;gap:5px 10px;font-size:12px;color:#334155}.dark .vpsnw-overview-band{color:#dbeafe}.vpsnw-overview-band span,.vpsnw-overview-band small{white-space:nowrap}.vpsnw-overview-band small{color:#64748b}.dark .vpsnw-overview-band small{color:#94a3b8}.vpsnw-overview-latency{grid-column:1/-1;display:flex;flex-wrap:wrap;justify-content:flex-start;gap:5px;padding-top:7px;border-top:1px dashed rgba(148,163,184,.24);min-height:31px}.vpsnw-latency-title{font-size:11px;color:#64748b;line-height:1;padding:5px 0;white-space:nowrap}.dark .vpsnw-latency-title{color:#94a3b8}.vpsnw-latency-pill{display:inline-flex;align-items:center;gap:5px;border-radius:999px;background:rgba(37,99,235,.1);color:#1d4ed8;font-size:12px;line-height:1;padding:5px 8px;white-space:nowrap}.vpsnw-latency-pill b{font-size:12px}.vpsnw-latency-pill.peer{background:rgba(245,158,11,.16);color:#b45309}.dark .vpsnw-latency-pill{background:rgba(147,197,253,.16);color:#bfdbfe}.dark .vpsnw-latency-pill.peer{background:rgba(251,191,36,.18);color:#fde68a}.vpsnw-overview-empty{font-size:12px;color:#94a3b8}" +
      ".vpsnw-anomalies{display:none;flex-wrap:wrap;align-items:center;gap:6px 8px;padding:8px 14px;border-bottom:1px solid rgba(148,163,184,.18);font-size:12px}.vpsnw-anomalies b{margin-right:2px}.vpsnw-anomaly{border:1px solid rgba(245,158,11,.35);border-radius:999px;background:rgba(245,158,11,.14);color:#92400e;cursor:pointer;line-height:1;padding:6px 8px}.vpsnw-anomaly.loss{border-color:rgba(239,68,68,.35);background:rgba(239,68,68,.13);color:#b91c1c}.vpsnw-anomaly.jitter{border-color:rgba(168,85,247,.35);background:rgba(168,85,247,.13);color:#7e22ce}.dark .vpsnw-anomaly{color:#fde68a}.dark .vpsnw-anomaly.loss{color:#fecaca}.dark .vpsnw-anomaly.jitter{color:#e9d5ff}" +
      ".vpsnw-legend{display:flex;flex-wrap:wrap;justify-content:center;gap:10px 18px;padding:10px 14px 2px;font-size:13px}.vpsnw-legend span{display:inline-flex;align-items:center;gap:6px}.vpsnw-dot{width:10px;height:10px;border-radius:50%;display:inline-block}" +
      ".vpsnw-chart{position:relative;height:360px;padding:6px 14px 14px}.vpsnw-chart canvas{width:100%;height:100%;display:block}" +
      ".vpsnw-tip{display:none;position:absolute;z-index:20;min-width:160px;max-width:260px;padding:10px 12px;border:1px solid rgba(148,163,184,.35);border-radius:8px;background:rgba(255,255,255,.96);box-shadow:0 12px 28px rgba(15,23,42,.2);font-size:13px;color:#111827;pointer-events:none}.dark .vpsnw-tip{background:rgba(15,15,15,.96);color:#f8fafc}" +
      ".vpsnw-tip-time{color:#475569;margin-bottom:6px}.dark .vpsnw-tip-time{color:#cbd5e1}.vpsnw-tip-row{display:flex;align-items:center;justify-content:space-between;gap:14px;line-height:1.7}.vpsnw-tip-name{display:flex;align-items:center;gap:6px}" +
      ".vpsnw-empty{display:none;padding:22px 16px;color:#94a3b8;text-align:center;font-size:13px}" +
      "@media(max-width:760px){.vpsnw-chart{height:300px;padding-left:8px;padding-right:8px}.vpsnw-head{grid-template-columns:1fr}.vpsnw-title,.vpsnw-server-tabs,.vpsnw-peer-label,.vpsnw-peer-row{grid-column:1;grid-row:auto}.vpsnw-tools,.vpsnw-targets{padding:8px}.vpsnw-date{flex:1}.vpsnw-date input{min-width:0;width:100%}.vpsnw-target-row label,.vpsnw-target-row input{width:100%;min-width:0}.vpsnw-overview-row{grid-template-columns:1fr}.vpsnw-overview-band,.vpsnw-overview-latency{justify-content:flex-start}}";
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
        '<div class="vpsnw-head"><div class="vpsnw-title">延迟</div><div class="vpsnw-server-tabs" id="vpsnw-server-tabs"></div><span class="vpsnw-peer-label">互 ping 目标</span><div class="vpsnw-peer-row"><div class="vpsnw-peer-tabs" id="vpsnw-peer-tabs"></div><span class="vpsnw-peer-note" id="vpsnw-peer-note"></span></div></div>' +
        '<div class="vpsnw-targets"><div class="vpsnw-target-row vpsnw-target-main"><label><span>目标向导</span><input id="vpsnw-target" placeholder="IP / 域名 / IP:端口"></label><label><span>名称</span><input id="vpsnw-target-name" placeholder="可留空"></label><button class="vpsnw-action-btn" id="vpsnw-add-target">加入监控</button><span class="vpsnw-target-note" id="vpsnw-target-note"></span></div><details class="vpsnw-discover"><summary>mihomo / Clash 连接发现</summary><div class="vpsnw-target-row"><label><span>控制器</span><input id="vpsnw-mihomo-url" placeholder="http://127.0.0.1:9090"></label><label><span>密钥</span><input id="vpsnw-mihomo-secret" type="password" placeholder="可留空"></label><button class="vpsnw-action-btn secondary" id="vpsnw-discover">读取连接</button></div><div class="vpsnw-discover-list" id="vpsnw-discover-list"></div></details></div>' +
        '<div class="vpsnw-tools"><button class="vpsnw-icon-btn" id="vpsnw-prev-day" title="上一天"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M12.8 4.2a1 1 0 0 1 0 1.4L8.4 10l4.4 4.4a1 1 0 1 1-1.4 1.4l-5.1-5.1a1 1 0 0 1 0-1.4l5.1-5.1a1 1 0 0 1 1.4 0Z"/></svg></button><label class="vpsnw-date"><span class="vpsnw-muted">日期</span><input id="vpsnw-date" type="date"></label><button class="vpsnw-icon-btn" id="vpsnw-next-day" title="下一天"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M7.2 15.8a1 1 0 0 1 0-1.4l4.4-4.4-4.4-4.4a1 1 0 0 1 1.4-1.4l5.1 5.1a1 1 0 0 1 0 1.4l-5.1 5.1a1 1 0 0 1-1.4 0Z"/></svg></button><div class="vpsnw-protocol"><span class="vpsnw-muted">协议</span><button class="vpsnw-chip" data-vpsnw-protocol="ICMP">ICMP</button><button class="vpsnw-chip" data-vpsnw-protocol="TCP">TCP</button></div><div class="vpsnw-tool"><span class="vpsnw-muted">显示</span><button class="vpsnw-chip" id="vpsnw-extremes">极值</button><button class="vpsnw-chip" id="vpsnw-average">平均线</button></div></div>' +
        '<div class="vpsnw-overview" id="vpsnw-overview"></div>' +
        '<div class="vpsnw-anomalies" id="vpsnw-anomalies"></div>' +
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
      initControls();
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
  function dateInputValue(ts) { var d = new Date(ts); return d.getFullYear() + "-" + pad(d.getMonth()+1) + "-" + pad(d.getDate()); }
  function dayRange(value) {
    var text = value || dateInputValue(Date.now());
    var parts = text.split("-").map(Number);
    var start = new Date(parts[0], parts[1] - 1, parts[2]).getTime();
    return { start: start, end: start + 86400000 };
  }
  function fmtMs(v) { return !isFinite(v) ? "-" : (v < 100 ? v.toFixed(2) : Math.round(v)) + "ms"; }
  function fmtBytes(value) {
    var n = Math.max(0, Number(value) || 0);
    var units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
    var i = 0;
    while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
    var text = i === 0 ? String(Math.round(n)) : (n < 10 ? n.toFixed(2) : n < 100 ? n.toFixed(1) : String(Math.round(n)));
    return text + " " + units[i];
  }
  function fmtRate(value) { return fmtBytes(value) + "/s"; }
  function escapeHtml(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return {"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[ch];
    });
  }

  function setTargetNote(text, error) {
    var note = panel && panel.querySelector("#vpsnw-target-note");
    if (!note) return;
    note.textContent = text || "";
    note.style.color = error ? "#ef4444" : "";
  }

  function targetKindHint(value) {
    var text = String(value || "").trim();
    if (!text) return "IP/域名创建 ICMP，IP:端口创建 TCP";
    if (text.indexOf("://") >= 0) {
      try { text = new URL(text).host; } catch (_) {}
    }
    text = text.replace(/\/$/, "");
    if (/^\[[^\]]+\]:\d{1,5}$/.test(text) || /^[^:\s]+:\d{1,5}$/.test(text)) return "将创建 TCP 监控";
    if (/[\/?#\s]/.test(text)) return "请输入 IP、域名或 IP:端口";
    return "将创建 ICMP 监控";
  }

  function updateTargetHint() {
    var input = panel && panel.querySelector("#vpsnw-target");
    if (!input) return;
    setTargetNote(targetKindHint(input.value), false);
  }

  async function createMonitorTarget(target, name) {
    target = String(target || "").trim();
    name = String(name || "").trim();
    if (!target) {
      setTargetNote("请输入目标", true);
      return;
    }
    var headers = { "Content-Type": "application/json" };
    var token = getToken();
    if (token) headers.Authorization = "Bearer " + token;
    setTargetNote("正在加入");
    var res = await fetch("/api/v1/netwatch/target", {
      method: "POST",
      headers: headers,
      credentials: "same-origin",
      body: JSON.stringify({ target: target, name: name, duration: 30 })
    });
    var body = await res.json();
    if (!body.success) throw new Error(body.error || "加入失败");
    var data = body.data || {};
    setTargetNote((data.created ? "已加入 " : "已存在 ") + (data.type_name || "") + " " + (data.target || target));
    state.loaded = false;
    await load({ resetView: false });
  }

  async function discoverMihomoTargets() {
    var urlInput = panel.querySelector("#vpsnw-mihomo-url");
    var secretInput = panel.querySelector("#vpsnw-mihomo-secret");
    var list = panel.querySelector("#vpsnw-discover-list");
    var controller = urlInput ? urlInput.value.trim() : "";
    if (!controller) {
      setTargetNote("请输入 mihomo 控制器地址", true);
      return;
    }
    var headers = { "Content-Type": "application/json" };
    var token = getToken();
    if (token) headers.Authorization = "Bearer " + token;
    if (list) list.innerHTML = '<span class="vpsnw-overview-empty">读取中</span>';
    var res = await fetch("/api/v1/netwatch/mihomo/discover", {
      method: "POST",
      headers: headers,
      credentials: "same-origin",
      body: JSON.stringify({ controller: controller, secret: secretInput ? secretInput.value : "", limit: 60 })
    });
    var body = await res.json();
    if (!body.success) throw new Error(body.error || "读取失败");
    renderMihomoTargets((body.data && body.data.targets) || []);
  }

  function renderMihomoTargets(targets) {
    var list = panel && panel.querySelector("#vpsnw-discover-list");
    if (!list) return;
    if (!targets.length) {
      list.innerHTML = '<span class="vpsnw-overview-empty">没有发现可加入的连接目标</span>';
      return;
    }
    list.innerHTML = targets.map(function (item, index) {
      var meta = [item.type_name || "", item.network || "", item.rule || "", item.chain || "", item.process || ""].filter(Boolean).join(" · ");
      return '<span class="vpsnw-discover-item" title="' + escapeHtml(meta) + '"><b>' + escapeHtml(item.target) + '</b><small>' + escapeHtml(item.count || 1) + '</small><button class="vpsnw-mini-btn" data-vpsnw-discovered="' + index + '">加入</button></span>';
    }).join("");
    Array.prototype.forEach.call(list.querySelectorAll("[data-vpsnw-discovered]"), function (btn) {
      btn.onclick = function (event) {
        event.preventDefault();
        event.stopPropagation();
        var item = targets[Number(btn.getAttribute("data-vpsnw-discovered"))];
        if (!item) return;
        createMonitorTarget(item.target, "").catch(function (err) { setTargetNote(err.message || String(err), true); });
      };
    });
  }

  function latestValidPoint(points) {
    points = points || [];
    for (var i = points.length - 1; i >= 0; i--) {
      var p = points[i];
      if (p && p.status !== 0 && p.delay > 0) return p;
    }
    return null;
  }

  function overviewLatencyItems(serverId) {
    if (!state.data) return [];
    return (state.data.series || []).filter(function (raw) {
      if (String(raw.server_id) !== String(serverId)) return false;
      if (raw.type_name !== state.protocol) return false;
      if (raw.is_peer) {
        if (!state.peerTargetServerId || String(raw.peer_server_id) !== String(state.peerTargetServerId)) return false;
        if (String(raw.peer_server_id) === String(serverId)) return false;
      }
      return true;
    }).map(function (raw) {
      var point = latestValidPoint(raw.data_points);
      if (!point) return null;
      return {
        name: raw.service_name || raw.target || "Ping",
        delay: point.delay,
        display_index: raw.display_index || 0,
        is_peer: !!raw.is_peer
      };
    }).filter(Boolean).sort(function (a, b) {
      if (a.display_index !== b.display_index) return b.display_index - a.display_index;
      return a.name.localeCompare(b.name);
    });
  }

  function renderOverview() {
    var box = panel && panel.querySelector("#vpsnw-overview");
    if (!box || !state.data) return;
    ensureSelectedServer();
    var servers = state.data.servers || [];
    if (!servers.length) {
      box.innerHTML = "";
      return;
    }
    var rows = servers.map(function (server) {
      var id = String(server.id);
      var latencyItems = overviewLatencyItems(id);
      var visibleLatency = latencyItems.slice(0, 5).map(function (item) {
        return '<span class="vpsnw-latency-pill' + (item.is_peer ? " peer" : "") + '"><span>' + escapeHtml(item.name) + '</span><b>' + fmtMs(item.delay) + '</b></span>';
      }).join("");
      if (latencyItems.length > 5) {
        visibleLatency += '<span class="vpsnw-overview-empty">+' + (latencyItems.length - 5) + '</span>';
      }
      if (!visibleLatency) visibleLatency = '<span class="vpsnw-overview-empty">暂无' + escapeHtml(state.protocol) + '延迟</span>';
      var totalTransfer = (Number(server.net_in_transfer) || 0) + (Number(server.net_out_transfer) || 0);
      var planTags = "";
      if (server.bandwidth_label) planTags += '<em class="vpsnw-band-label" title="带宽">' + escapeHtml(server.bandwidth_label) + '</em>';
      if (server.remaining_label) planTags += '<em class="vpsnw-remaining-label" title="剩余时间">' + escapeHtml(server.remaining_label) + '</em>';
      return '<div class="vpsnw-overview-row' + (id === String(state.selectedServerId) ? " active" : "") + '" data-vpsnw-server-row="' + escapeHtml(id) + '">' +
        '<div class="vpsnw-overview-main"><div class="vpsnw-overview-name"><span class="vpsnw-status-dot' + (server.online ? " online" : "") + '"></span><strong title="' + escapeHtml(server.name || "") + '">' + escapeHtml(server.name || ("VPS " + id)) + '</strong></div><div class="vpsnw-plan-tags">' + planTags + '</div></div>' +
        '<div class="vpsnw-overview-band"><span>↑ ' + fmtRate(server.net_out_speed) + '</span><span>↓ ' + fmtRate(server.net_in_speed) + '</span><small>总 ' + fmtBytes(totalTransfer) + '</small></div>' +
        '<div class="vpsnw-overview-latency"><span class="vpsnw-latency-title">延迟</span>' + visibleLatency + '</div>' +
      '</div>';
    }).join("");
    box.innerHTML = '<div class="vpsnw-overview-head"><b>VPS 快览</b><span>' + escapeHtml(state.protocol) + ' · 带宽 / 剩余 / 速率 / 总传输 / 最新延迟</span></div><div class="vpsnw-overview-table">' + rows + '</div>';
    Array.prototype.forEach.call(box.querySelectorAll("[data-vpsnw-server-row]"), function (row) {
      row.onclick = function () { selectServer(row.getAttribute("data-vpsnw-server-row")); };
    });
  }

  function initControls() {
    var targetBox = panel.querySelector(".vpsnw-targets");
    if (targetBox && !getToken()) targetBox.style.display = "none";
    state.date = state.date || dateInputValue(Date.now());
    panel.querySelector("#vpsnw-date").value = state.date;
    panel.querySelector("#vpsnw-date").onchange = function () {
      if (!this.value) return;
      state.date = this.value;
      state.view = null;
      load({ resetView: true }).catch(function (err) { setEmptyText(err.message || String(err)); });
    };
    panel.querySelector("#vpsnw-prev-day").onclick = function () { shiftDate(-1); };
    panel.querySelector("#vpsnw-next-day").onclick = function () { shiftDate(1); };
    panel.querySelector("#vpsnw-extremes").onclick = function () { state.showExtremes = !state.showExtremes; draw(); };
    panel.querySelector("#vpsnw-average").onclick = function () { state.showAverage = !state.showAverage; draw(); };
    panel.querySelector("#vpsnw-add-target").onclick = function () {
      createMonitorTarget(panel.querySelector("#vpsnw-target").value, panel.querySelector("#vpsnw-target-name").value).catch(function (err) { setTargetNote(err.message || String(err), true); });
    };
    panel.querySelector("#vpsnw-target").onkeydown = function (event) {
      if (event.key === "Enter") panel.querySelector("#vpsnw-add-target").click();
    };
    panel.querySelector("#vpsnw-target").oninput = updateTargetHint;
    updateTargetHint();
    panel.querySelector("#vpsnw-discover").onclick = function () {
      discoverMihomoTargets().catch(function (err) { setTargetNote(err.message || String(err), true); });
    };
    Array.prototype.forEach.call(panel.querySelectorAll("[data-vpsnw-protocol]"), function (btn) {
      btn.onclick = function () {
        state.protocol = btn.getAttribute("data-vpsnw-protocol") || "ICMP";
        draw();
      };
    });
    syncControls();
  }

  function syncControls() {
    if (!panel) return;
    var dateInput = panel.querySelector("#vpsnw-date");
    if (dateInput) dateInput.value = state.date || dateInputValue(Date.now());
    Array.prototype.forEach.call(panel.querySelectorAll("[data-vpsnw-protocol]"), function (btn) {
      btn.classList.toggle("active", btn.getAttribute("data-vpsnw-protocol") === state.protocol);
    });
    var extremes = panel.querySelector("#vpsnw-extremes");
    var average = panel.querySelector("#vpsnw-average");
    if (extremes) extremes.classList.toggle("active", state.showExtremes);
    if (average) average.classList.toggle("active", state.showAverage);
  }

  function shiftDate(days) {
    var range = dayRange(state.date);
    state.date = dateInputValue(range.start + days * 86400000);
    state.view = null;
    syncControls();
    load({ resetView: true }).catch(function (err) { setEmptyText(err.message || String(err)); });
  }

  function latencyURL() {
    var range = dayRange(state.date);
    var params = new URLSearchParams();
    params.set("period", "1d");
    params.set("start", String(range.start));
    params.set("end", String(range.end));
    return "/api/v1/netwatch/latency?" + params.toString();
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
      btn.onclick = function () { selectServer(id); };
      box.appendChild(btn);
    });
  }

  function selectServer(id) {
    state.selectedServerId = id;
    state.hover = null;
    try { window.localStorage.setItem("vpsnw-selected-server", id); } catch (_) {}
    if (state.peerTargetServerId && state.peerTargetServerId !== id) startPeerWait();
    draw();
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

  function includeRawSeries(raw) {
    if (!raw) return false;
    ensureSelectedServer();
    if (state.selectedServerId && String(raw.server_id) !== String(state.selectedServerId)) return false;
    if (raw.type_name !== state.protocol) return false;
    var isPeer = !!raw.is_peer;
    if (isPeer) {
      if (!state.peerTargetServerId || String(raw.peer_server_id) !== String(state.peerTargetServerId)) return false;
      if (String(raw.peer_server_id) === String(state.selectedServerId)) return false;
    }
    return true;
  }

  function selectedRawSeries() {
    if (!state.data) return [];
    return (state.data.series || []).filter(includeRawSeries);
  }

  function aggregate() {
    if (!state.data) return [];
    ensureSelectedServer();
    var grouped = {};
    selectedRawSeries().forEach(function (raw) {
      if (state.selectedServerId && String(raw.server_id) !== String(state.selectedServerId)) return;
      if (raw.type_name !== state.protocol) return;
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

  function fmtShortTime(ts) {
    var d = new Date(ts);
    return pad(d.getHours()) + ":" + pad(d.getMinutes());
  }

  function visibleRawPoints(raw) {
    if (!state.view) return [];
    return (raw.data_points || []).filter(function (p) {
      return p && p.ts >= state.view.start && p.ts <= state.view.end;
    }).sort(function (a, b) { return a.ts - b.ts; });
  }

  function detectAnomalies() {
    var result = [];
    selectedRawSeries().forEach(function (raw) {
      var points = visibleRawPoints(raw);
      if (!points.length) return;
      var valid = points.filter(function (p) { return p.status !== 0 && p.delay > 0; });
      if (valid.length >= 3) {
        var total = valid.reduce(function (sum, p) { return sum + p.delay; }, 0);
        var avg = total / valid.length;
        var max = valid.reduce(function (best, p) { return p.delay > best.delay ? p : best; }, valid[0]);
        if (max.delay >= Math.max(avg + 30, avg * 1.8)) {
          result.push({ kind: "peak", label: "峰值 " + raw.service_name + " " + fmtMs(max.delay) + " " + fmtShortTime(max.ts), start: max.ts - 15 * 60000, end: max.ts + 15 * 60000 });
        }
        var jitterThreshold = Math.max(25, avg * 0.45);
        var jitterStart = 0, jitterHits = 0;
        for (var i = 1; i < valid.length; i++) {
          if (Math.abs(valid[i].delay - valid[i - 1].delay) >= jitterThreshold) {
            if (!jitterStart) jitterStart = valid[i - 1].ts;
            jitterHits++;
          } else if (jitterHits >= 3) {
            result.push({ kind: "jitter", label: "持续抖动 " + raw.service_name + " " + fmtShortTime(jitterStart) + "-" + fmtShortTime(valid[i - 1].ts), start: jitterStart, end: valid[i - 1].ts });
            jitterStart = 0;
            jitterHits = 0;
          } else {
            jitterStart = 0;
            jitterHits = 0;
          }
        }
        if (jitterHits >= 3) {
          result.push({ kind: "jitter", label: "持续抖动 " + raw.service_name + " " + fmtShortTime(jitterStart) + "-" + fmtShortTime(valid[valid.length - 1].ts), start: jitterStart, end: valid[valid.length - 1].ts });
        }
      }

      var lossStart = 0, lossEnd = 0, lossCount = 0;
      points.forEach(function (p) {
        if (p.status === 0) {
          if (!lossStart) lossStart = p.ts;
          lossEnd = p.ts;
          lossCount++;
          return;
        }
        if (lossCount > 0) {
          result.push({ kind: "loss", label: "丢包 " + raw.service_name + " " + fmtShortTime(lossStart) + "-" + fmtShortTime(lossEnd), start: lossStart, end: lossEnd + 5 * 60000 });
        }
        lossStart = 0;
        lossEnd = 0;
        lossCount = 0;
      });
      if (lossCount > 0) {
        result.push({ kind: "loss", label: "丢包 " + raw.service_name + " " + fmtShortTime(lossStart) + "-" + fmtShortTime(lossEnd), start: lossStart, end: lossEnd + 5 * 60000 });
      }
    });
    return result.slice(0, 12);
  }

  function renderAnomalies() {
    var box = panel && panel.querySelector("#vpsnw-anomalies");
    if (!box || !state.view) return;
    var items = detectAnomalies();
    if (!items.length) {
      box.style.display = "none";
      box.innerHTML = "";
      return;
    }
    box.style.display = "flex";
    box.innerHTML = '<b>异常标记</b>' + items.map(function (item, index) {
      return '<button class="vpsnw-anomaly ' + item.kind + '" data-vpsnw-anomaly="' + index + '">' + escapeHtml(item.label) + '</button>';
    }).join("");
    Array.prototype.forEach.call(box.querySelectorAll("[data-vpsnw-anomaly]"), function (btn) {
      btn.onclick = function () {
        var item = items[Number(btn.getAttribute("data-vpsnw-anomaly"))];
        if (!item) return;
        clampView(item.start - 10 * 60000, item.end + 10 * 60000);
        draw();
      };
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
    var range = state.data && state.data.start && state.data.end ? { start: state.data.start, end: state.data.end } : dayRange(state.date);
    var start = range.start;
    var end = range.end;
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
      state.date = state.date || dateInputValue(Date.now());
      var res = await fetch(latencyURL(), { headers: headers, credentials: "same-origin" });
      var body = await res.json();
      if (!body.success) throw new Error(body.error || "请求失败");
      state.data = body.data;
      state.loaded = true;
      syncPeerState();
      ensureSelectedServer();
      updateDomain(options.resetView !== false);
      syncControls();
      draw();
    } finally {
      state.loading = false;
    }
  }

  function visiblePoints(series) {
    return series.points.filter(function (p) { return p.ts >= state.view.start && p.ts <= state.view.end; });
  }

  function fillRoundRect(x, y, w, h, r) {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.lineTo(x + w - r, y);
    ctx.quadraticCurveTo(x + w, y, x + w, y + r);
    ctx.lineTo(x + w, y + h - r);
    ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
    ctx.lineTo(x + r, y + h);
    ctx.quadraticCurveTo(x, y + h, x, y + h - r);
    ctx.lineTo(x, y + r);
    ctx.quadraticCurveTo(x, y, x + r, y);
    ctx.closePath();
    ctx.fill();
  }

  function drawValueLabel(text, px, py, color, plot) {
    ctx.save();
    ctx.font = "700 11px -apple-system,BlinkMacSystemFont,Segoe UI,Arial";
    var w = ctx.measureText(text).width + 12;
    var h = 18;
    var x0 = Math.max(plot.left, Math.min(plot.left + plot.width - w, px - w / 2));
    var y0 = Math.max(plot.top, Math.min(plot.top + plot.height - h, py - 22));
    ctx.fillStyle = color;
    fillRoundRect(x0, y0, w, h, 4);
    ctx.fillStyle = "#fff";
    ctx.fillText(text, x0 + 6, y0 + 13);
    ctx.restore();
  }

  function seriesStats(points) {
    if (!points.length) return null;
    var min = points[0];
    var max = points[0];
    var total = 0;
    points.forEach(function (p) {
      if (p.delay < min.delay) min = p;
      if (p.delay > max.delay) max = p;
      total += p.delay;
    });
    return { min: min, max: max, avg: total / points.length };
  }

  function drawStats(series, plot, x, y) {
    if (!state.showAverage && !state.showExtremes) return;
    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
      var stats = seriesStats(pts);
      if (!stats) return;
      var color = colors[index % colors.length];
      if (state.showAverage) {
        var yy = y(stats.avg);
        ctx.save();
        ctx.strokeStyle = color;
        ctx.globalAlpha = 0.46;
        ctx.setLineDash([6, 4]);
        ctx.beginPath();
        ctx.moveTo(plot.left, yy);
        ctx.lineTo(plot.left + plot.width, yy);
        ctx.stroke();
        ctx.restore();
        drawValueLabel("平均 " + fmtMs(stats.avg), plot.left + plot.width - 42, yy, color, plot);
      }
      if (state.showExtremes) {
        drawValueLabel("↑ " + fmtMs(stats.max.delay), x(stats.max.ts), y(stats.max.delay), "#ef6461", plot);
        if (stats.min.ts !== stats.max.ts || Math.abs(stats.min.delay - stats.max.delay) > 0.01) {
          drawValueLabel("↓ " + fmtMs(stats.min.delay), x(stats.min.ts), y(stats.min.delay) + 28, "#4fa8df", plot);
        }
      }
    });
  }

  function draw() {
    if (!panel || panel.hidden || !canvas || !ctx || !state.data) return;
    syncControls();
    renderServerTabs();
    renderPeerTargets();
    renderOverview();
    var series = aggregate();
    updatePeerNote(series);
    renderAnomalies();
    if (!series.length) setEmptyText(state.peerTargetServerId && state.peerTargetServerId !== state.selectedServerId && state.peerWaitUntil && Date.now() <= state.peerWaitUntil ? "采集中" : "暂无延迟数据");
    panel.querySelector("#vpsnw-empty").style.display = series.length ? "none" : "block";
    panel.querySelector(".vpsnw-chart").style.display = series.length ? "block" : "none";
    renderLegend(series);
    if (!series.length) return;

    var rect = canvas.getBoundingClientRect();
    var dpr = window.devicePixelRatio || 1;
    canvas.width = Math.max(1, Math.floor(rect.width * dpr));
    canvas.height = Math.max(1, Math.floor(rect.height * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, rect.width, rect.height);
    var isDark = document.documentElement.classList.contains("dark") || document.body.classList.contains("dark");
    ctx.fillStyle = isDark ? "#090909" : "#ffffff";
    ctx.fillRect(0, 0, rect.width, rect.height);

    var pad = { left: 44, right: 18, top: 18, bottom: 54 };
    var plot = { left: pad.left, top: pad.top, width: rect.width - pad.left - pad.right, height: rect.height - pad.top - pad.bottom };
    lastPlot = plot;
    var visible = [];
    series.forEach(function (s) { visiblePoints(s).forEach(function (p) { visible.push(p); }); });
    var maxY = Math.max.apply(null, (visible.length ? visible : [{ delay: 10 }]).map(function (p) { return p.delay; }));
    maxY = Math.max(10, Math.ceil(maxY * 1.18 / 10) * 10);
    function x(ts) { return plot.left + (ts - state.view.start) / (state.view.end - state.view.start) * plot.width; }
    function y(v) { return plot.top + plot.height - v / maxY * plot.height; }

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
      var pts = visiblePoints(s);
      if (!pts.length) return;
      ctx.strokeStyle = colors[index % colors.length];
      ctx.lineWidth = 1.8;
      ctx.beginPath();
      pts.forEach(function (p, i) { if (i === 0) ctx.moveTo(x(p.ts), y(p.delay)); else ctx.lineTo(x(p.ts), y(p.delay)); });
      ctx.stroke();
    });
    drawStats(series, plot, x, y);

    if (state.hover) drawHover(series, plot, x, y);
  }

  function renderLegend(series) {
    var legend = panel.querySelector("#vpsnw-legend");
    legend.innerHTML = "";
    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
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
      var pts = visiblePoints(s);
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
    var btn = document.querySelector("[" + marker + "]");
    if (btn) btn.remove();
    if (state.visible && panel) {
      state.visible = false;
      panel.hidden = true;
    }
    return false;
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
  var board;
  var state = { servers: [], loading: false, error: "", view: localStorage.getItem("inline") === "1" ? "table" : "grid" };
  var serviceEmptyPattern = /No Service|no service|暂无服务|没有服务|服务数据|暂无数据/i;

  function injectStyle() {
    if (document.getElementById("vps-netwatch-server-board-style")) return;
    var style = document.createElement("style");
    style.id = "vps-netwatch-server-board-style";
    style.textContent =
      ".vpsnw-server-board-ready .server-overview-controls section>button:nth-of-type(2){display:none!important}" +
      ".vpsnw-server-board-ready .server-card-list,.vpsnw-server-board-ready .server-inline-list{display:none!important}" +
      "#vps-netwatch-server-board{margin-top:14px;color:#111827}" +
      ".dark #vps-netwatch-server-board{color:#f8fafc}" +
      ".vpsnw-server-board-head{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:10px}" +
      ".vpsnw-server-board-title{display:flex;align-items:center;gap:8px;font-size:13px;font-weight:800;color:#0f172a}.dark .vpsnw-server-board-title{color:#f8fafc}" +
      ".vpsnw-server-board-title span{width:8px;height:8px;border-radius:999px;background:#22c55e;box-shadow:0 0 0 3px rgba(34,197,94,.14)}" +
      ".vpsnw-server-view-tabs{display:flex;align-items:center;gap:6px;border:1px solid rgba(148,163,184,.24);border-radius:999px;background:rgba(255,255,255,.72);padding:3px}.dark .vpsnw-server-view-tabs{background:rgba(15,23,42,.62);border-color:rgba(255,255,255,.12)}" +
      ".vpsnw-server-view-tabs button{border:0;border-radius:999px;background:transparent;color:#64748b;cursor:pointer;font-size:12px;font-weight:700;line-height:1;padding:7px 10px}.dark .vpsnw-server-view-tabs button{color:#cbd5e1}.vpsnw-server-view-tabs button.active{background:#2563eb;color:#fff}" +
      ".vpsnw-server-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:10px}" +
      ".vpsnw-node-card{border:1px solid rgba(148,163,184,.24);border-radius:8px;background:rgba(255,255,255,.84);box-shadow:0 10px 24px rgba(15,23,42,.07);padding:12px;cursor:pointer;transition:transform .16s ease,border-color .16s ease,box-shadow .16s ease}.vpsnw-node-card:hover{transform:translateY(-1px);border-color:rgba(37,99,235,.38);box-shadow:0 14px 30px rgba(15,23,42,.12)}.dark .vpsnw-node-card{background:rgba(15,23,42,.76);border-color:rgba(255,255,255,.12);box-shadow:none}" +
      ".vpsnw-node-top{display:flex;align-items:flex-start;justify-content:space-between;gap:10px;margin-bottom:10px}.vpsnw-node-name{display:flex;align-items:center;gap:8px;min-width:0}.vpsnw-node-name strong{font-size:13px;line-height:1.2;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.vpsnw-node-sub{display:flex;align-items:center;gap:5px;margin-top:4px;flex-wrap:wrap}" +
      ".vpsnw-status{width:9px;height:9px;border-radius:999px;background:#ef4444;box-shadow:0 0 0 3px rgba(239,68,68,.13);flex:0 0 auto}.vpsnw-status.online{background:#22c55e;box-shadow:0 0 0 3px rgba(34,197,94,.16)}" +
      ".vpsnw-pill{display:inline-flex;align-items:center;border-radius:999px;font-size:11px;font-weight:700;line-height:1;padding:4px 7px;white-space:nowrap}.vpsnw-pill.ip{background:rgba(37,99,235,.11);color:#1d4ed8}.vpsnw-pill.band{background:rgba(20,184,166,.12);color:#0f766e}.vpsnw-pill.left{background:rgba(99,102,241,.12);color:#4f46e5}.vpsnw-pill.muted{background:rgba(100,116,139,.12);color:#64748b}.dark .vpsnw-pill.ip{background:rgba(96,165,250,.16);color:#bfdbfe}.dark .vpsnw-pill.band{background:rgba(45,212,191,.16);color:#99f6e4}.dark .vpsnw-pill.left{background:rgba(129,140,248,.18);color:#c7d2fe}.dark .vpsnw-pill.muted{background:rgba(148,163,184,.16);color:#cbd5e1}" +
      ".vpsnw-plan-row{display:flex;flex-wrap:wrap;gap:6px;margin-bottom:10px}.vpsnw-meter-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;margin-bottom:10px}.vpsnw-meter label{display:flex;justify-content:space-between;gap:6px;color:#64748b;font-size:11px}.dark .vpsnw-meter label{color:#94a3b8}.vpsnw-meter b{color:#0f172a;font-size:11px}.dark .vpsnw-meter b{color:#f8fafc}.vpsnw-bar{height:4px;border-radius:999px;background:rgba(148,163,184,.18);overflow:hidden;margin-top:5px}.vpsnw-bar span{display:block;height:100%;border-radius:999px;background:#60a5fa}.vpsnw-bar.mem span{background:#a78bfa}.vpsnw-bar.disk span{background:#f59e0b}" +
      ".vpsnw-speed-row{display:grid;grid-template-columns:1fr 1fr;gap:8px;border-top:1px solid rgba(148,163,184,.18);padding-top:9px}.vpsnw-speed-row div{display:flex;flex-direction:column;gap:3px}.vpsnw-speed-row span{color:#64748b;font-size:11px}.dark .vpsnw-speed-row span{color:#94a3b8}.vpsnw-speed-row b{font-size:12px;color:#0f172a}.dark .vpsnw-speed-row b{color:#f8fafc}.vpsnw-up{color:#2563eb!important}.vpsnw-down{color:#16a34a!important}" +
      ".vpsnw-server-table-wrap{overflow:auto;border:1px solid rgba(148,163,184,.22);border-radius:8px;background:rgba(255,255,255,.82);box-shadow:0 10px 24px rgba(15,23,42,.06)}.dark .vpsnw-server-table-wrap{background:rgba(15,23,42,.72);border-color:rgba(255,255,255,.12);box-shadow:none}" +
      ".vpsnw-server-table{width:100%;min-width:940px;border-collapse:separate;border-spacing:0}.vpsnw-server-table th{position:sticky;top:0;background:rgba(248,250,252,.92);color:#64748b;font-size:12px;font-weight:800;text-align:left;padding:11px 12px;white-space:nowrap}.dark .vpsnw-server-table th{background:rgba(15,23,42,.94);color:#cbd5e1}.vpsnw-server-table td{border-top:1px solid rgba(148,163,184,.18);font-size:12px;padding:11px 12px;vertical-align:middle}.vpsnw-server-table tr{cursor:pointer}.vpsnw-server-table tr:hover td{background:rgba(37,99,235,.06)}.dark .vpsnw-server-table tr:hover td{background:rgba(96,165,250,.08)}" +
      ".vpsnw-cell-node{display:flex;align-items:center;gap:8px;min-width:170px}.vpsnw-cell-node strong{display:block;max-width:210px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.vpsnw-cell-node small{display:block;color:#64748b;margin-top:3px}.dark .vpsnw-cell-node small{color:#94a3b8}.vpsnw-table-meter{display:grid;gap:4px;min-width:92px}.vpsnw-table-meter span{display:flex;justify-content:space-between;color:#64748b;font-size:11px}.dark .vpsnw-table-meter span{color:#94a3b8}" +
      ".vpsnw-server-empty{border:1px dashed rgba(148,163,184,.36);border-radius:8px;background:rgba(255,255,255,.66);color:#64748b;font-size:13px;padding:22px;text-align:center}.dark .vpsnw-server-empty{background:rgba(15,23,42,.5);color:#94a3b8;border-color:rgba(255,255,255,.16)}" +
      "@media(max-width:760px){#vps-netwatch-server-board{margin-top:10px}.vpsnw-server-board-head{align-items:flex-start;flex-direction:column}.vpsnw-meter-grid{grid-template-columns:1fr}.vpsnw-server-grid{grid-template-columns:1fr}.vpsnw-speed-row{grid-template-columns:1fr}}";
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

  function pct(used, total) {
    total = Number(total) || 0;
    if (!total) return 0;
    return Math.max(0, Math.min(100, (Number(used) || 0) / total * 100));
  }

  function protocolTags(server) {
    var tags = [];
    if (server.ipv4) tags.push("IPv4");
    if (server.ipv6) tags.push("IPv6");
    if (!tags.length && server.ip) tags.push(String(server.ip).indexOf(":") >= 0 ? "IPv6" : "IPv4");
    return tags.length ? tags : ["IP"];
  }

  function meter(label, value, kind) {
    var n = Math.max(0, Math.min(100, Number(value) || 0));
    return '<div class="vpsnw-meter"><label><span>' + label + '</span><b>' + n.toFixed(1) + '%</b></label><div class="vpsnw-bar ' + kind + '"><span style="width:' + n.toFixed(2) + '%"></span></div></div>';
  }

  function tableMeter(label, value, kind) {
    var n = Math.max(0, Math.min(100, Number(value) || 0));
    return '<div class="vpsnw-table-meter"><span><em>' + label + '</em><b>' + n.toFixed(1) + '%</b></span><div class="vpsnw-bar ' + kind + '"><span style="width:' + n.toFixed(2) + '%"></span></div></div>';
  }

  function planPills(server) {
    var remaining = server.remaining_label || "未设置剩余时间";
    var bandwidth = server.bandwidth_label || "未设置带宽";
    return '<span class="vpsnw-pill left">' + escapeHtml(remaining) + '</span><span class="vpsnw-pill band">' + escapeHtml(bandwidth) + '</span>';
  }

  function ipPills(server) {
    return protocolTags(server).map(function (tag) {
      return '<span class="vpsnw-pill ip">' + tag + '</span>';
    }).join("");
  }

  function renderGrid(servers) {
    return '<div class="vpsnw-server-grid">' + servers.map(function (server) {
      var cpu = Number(server.cpu) || 0;
      var mem = pct(server.mem_used, server.mem_total);
      var disk = pct(server.disk_used, server.disk_total);
      var ip = server.ip || server.ipv4 || server.ipv6 || "";
      return '<article class="vpsnw-node-card" data-vpsnw-server-id="' + server.id + '">' +
        '<div class="vpsnw-node-top"><div class="vpsnw-node-name"><span class="vpsnw-status ' + (server.online ? "online" : "") + '"></span><div><strong>' + escapeHtml(server.name) + '</strong><div class="vpsnw-node-sub">' + ipPills(server) + (ip ? '<span class="vpsnw-pill muted">' + escapeHtml(ip) + '</span>' : '') + '</div></div></div></div>' +
        '<div class="vpsnw-plan-row">' + planPills(server) + '</div>' +
        '<div class="vpsnw-meter-grid">' + meter("CPU", cpu, "cpu") + meter("内存", mem, "mem") + meter("硬盘", disk, "disk") + '</div>' +
        '<div class="vpsnw-speed-row"><div><span>实时上传</span><b class="vpsnw-up">↑ ' + fmtRate(server.net_out_speed) + '</b><span>累计 ' + fmtBytes(server.net_out_transfer) + '</span></div><div><span>实时下载</span><b class="vpsnw-down">↓ ' + fmtRate(server.net_in_speed) + '</b><span>累计 ' + fmtBytes(server.net_in_transfer) + '</span></div></div>' +
      '</article>';
    }).join("") + '</div>';
  }

  function renderTable(servers) {
    return '<div class="vpsnw-server-table-wrap"><table class="vpsnw-server-table"><thead><tr><th>节点</th><th>状态</th><th>CPU</th><th>内存</th><th>硬盘</th><th>剩余时间</th><th>最大带宽</th><th>实时网络速率</th><th>协议</th><th>总传输</th></tr></thead><tbody>' +
      servers.map(function (server) {
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
          '<td>↑ ' + fmtBytes(server.net_out_transfer) + '<br>↓ ' + fmtBytes(server.net_in_transfer) + '</td>' +
        '</tr>';
      }).join("") + '</tbody></table></div>';
  }

  function renderBoard() {
    if (!board) return;
    var servers = state.servers || [];
    var body = "";
    if (state.error) {
      body = '<div class="vpsnw-server-empty">' + escapeHtml(state.error) + '</div>';
    } else if (!servers.length) {
      body = '<div class="vpsnw-server-empty">' + (state.loading ? "正在读取服务器数据" : "暂无服务器数据") + '</div>';
    } else {
      body = state.view === "table" ? renderTable(servers) : renderGrid(servers);
    }
    board.innerHTML =
      '<div class="vpsnw-server-board-head"><div class="vpsnw-server-board-title"><span></span>服务器速览</div><div class="vpsnw-server-view-tabs"><button type="button" data-vpsnw-view="grid" class="' + (state.view === "grid" ? "active" : "") + '">卡片</button><button type="button" data-vpsnw-view="table" class="' + (state.view === "table" ? "active" : "") + '">表格</button></div></div>' +
      body;
  }

  async function loadServers() {
    if (state.loading) return;
    state.loading = true;
    try {
      var res = await fetch("/api/v1/netwatch/latency?period=1d&servers_only=1&_=" + Date.now(), { credentials: "same-origin" });
      var json = await res.json();
      if (!json.success) throw new Error(json.error || "服务器数据读取失败");
      state.servers = ((json.data || {}).servers || []).slice().sort(function (a, b) {
        return String(a.name || "").localeCompare(String(b.name || ""));
      });
      state.error = "";
    } catch (err) {
      state.error = (err && err.message) || "服务器数据读取失败";
    } finally {
      state.loading = false;
      renderBoard();
    }
  }

  function syncViewFromNative() {
    var next = localStorage.getItem("inline") === "1" ? "table" : "grid";
    if (state.view !== next) {
      state.view = next;
      renderBoard();
    }
  }

  function wireControls() {
    if (!board || board.dataset.vpsnwWired) return;
    board.dataset.vpsnwWired = "1";
    board.addEventListener("click", function (event) {
      var viewButton = event.target.closest("[data-vpsnw-view]");
      if (viewButton) {
        state.view = viewButton.getAttribute("data-vpsnw-view") || "grid";
        localStorage.setItem("inline", state.view === "table" ? "1" : "0");
        renderBoard();
        return;
      }
      var target = event.target.closest("[data-vpsnw-server-id]");
      if (target) {
        var id = target.getAttribute("data-vpsnw-server-id");
        if (id) {
          sessionStorage.setItem("fromMainPage", "true");
          window.location.href = "/server/" + encodeURIComponent(id);
        }
      }
    });
  }

  function wireNativeTableButton() {
    var controls = document.querySelector(".server-overview-controls section");
    if (!controls || controls.dataset.vpsnwServerBoardWired) return;
    controls.dataset.vpsnwServerBoardWired = "1";
    controls.addEventListener("click", function (event) {
      var buttons = Array.prototype.filter.call(controls.children, function (el) {
        return el.tagName === "BUTTON" && !el.hasAttribute("data-vps-netwatch-latency");
      });
      if (buttons[2] && (event.target === buttons[2] || buttons[2].contains(event.target))) {
        window.setTimeout(syncViewFromNative, 120);
      }
    }, true);
  }

  function hideEmptyServiceTracker() {
    var controls = document.querySelector(".server-overview-controls");
    if (!controls || !controls.parentElement) return;
    localStorage.setItem("showServices", "0");
    Array.prototype.forEach.call(controls.parentElement.children, function (el) {
      if (el === controls || el.id === "vps-netwatch-server-board" || el.classList.contains("server-overview") || el.classList.contains("server-card-list") || el.classList.contains("server-inline-list")) return;
      if (serviceEmptyPattern.test(el.textContent || "")) {
        el.style.display = "none";
      }
    });
  }

  function cleanupBranding() {
    if (!document.body) return;
    function textFromCodes(codes) {
      return String.fromCharCode.apply(String, codes);
    }
    var legacyName = textFromCodes([78,101,122,104,97]);
    var legacyNames = [
      textFromCodes([21738,21522,30417,25511,32,78,101,122,104,97,32,77,111,110,105,116,111,114,105,110,103]),
      textFromCodes([78,101,122,104,97,32,77,111,110,105,116,111,114,105,110,103]),
      textFromCodes([21738,21522,30417,25511]),
      textFromCodes([110,101,122,104,97,45,100,97,115,104]),
      legacyName
    ];
    var projectPath = "github.com/naiba/" + textFromCodes([110,101,122,104,97]);
    var themePath = "hamster1963/" + textFromCodes([110,101,122,104,97,45,100,97,115,104]);
    document.title = "vps-netwatch";
    Array.prototype.forEach.call(document.querySelectorAll('meta[name="apple-mobile-web-app-title"],meta[property="og:title"],meta[name="application-name"]'), function (meta) {
      meta.setAttribute("content", "vps-netwatch");
    });
    Array.prototype.forEach.call(document.querySelectorAll('a[href*="' + projectPath + '"]'), function (link) {
      link.href = "https://github.com/yikkrrtykj/vps-netwatch";
      link.textContent = "vps-netwatch";
    });
    Array.prototype.forEach.call(document.querySelectorAll('a[href*="' + themePath + '"]'), function (link) {
      var theme = link.closest(".server-footer-theme");
      if (theme) theme.remove();
    });
    var walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, {
      acceptNode: function (node) {
        var parent = node.parentElement;
        if (!parent || /SCRIPT|STYLE|TEXTAREA|INPUT/.test(parent.tagName)) return NodeFilter.FILTER_REJECT;
        var value = node.nodeValue || "";
        return legacyNames.some(function (name) { return value.indexOf(name) >= 0; }) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP;
      }
    });
    var nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(function (node) {
      var value = node.nodeValue || "";
      legacyNames.forEach(function (name) {
        value = value.split(name).join("vps-netwatch");
      });
      node.nodeValue = value;
    });
  }

  function ensureBoard() {
    injectStyle();
    cleanupBranding();
    hideEmptyServiceTracker();
    wireNativeTableButton();
    var controls = document.querySelector(".server-overview-controls");
    if (!controls) return false;
    document.body.classList.add("vpsnw-server-board-ready");
    if (!board) {
      board = document.createElement("section");
      board.id = "vps-netwatch-server-board";
      renderBoard();
      wireControls();
      loadServers();
      window.setInterval(loadServers, 3000);
    }
    if (!board.isConnected || board.previousElementSibling !== controls) {
      controls.insertAdjacentElement("afterend", board);
    }
    return true;
  }

  ensureBoard();
  var observer = new MutationObserver(ensureBoard);
  observer.observe(document.documentElement, { childList: true, subtree: true });
  window.setInterval(ensureBoard, 1500);
  document.addEventListener("visibilitychange", function () {
    if (!document.hidden) {
      ensureBoard();
      loadServers();
    }
  });
})();
</script>`
