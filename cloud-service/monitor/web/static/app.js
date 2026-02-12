(function () {
  // Kindle 页面脚本：拉取多账号告警数据并渲染紧凑列表。
  var listEl = document.getElementById("alert-list");
  var emptyEl = document.getElementById("empty");
  var totalEl = document.getElementById("total");
  var accountSuccessEl = document.getElementById("accounts-success");
  var accountTotalEl = document.getElementById("accounts-total");
  var updatedEl = document.getElementById("updated-at");
  var messageEl = document.getElementById("message");
  var refreshBtn = document.getElementById("refresh-btn");

  var refreshTimer = null;
  var refreshMs = 60000;

  function setMessage(text) {
    messageEl.textContent = text || "";
  }

  function escapeHTML(text) {
    if (text === null || text === undefined) {
      return "";
    }
    return String(text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/\"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  // 后端统一输出 UTC，这里按设备本地时区展示。
  function formatDeviceTime(utcText) {
    if (!utcText) {
      return "-";
    }

    var date = new Date(utcText);
    if (isNaN(date.getTime())) {
      return utcText;
    }

    try {
      return new Intl.DateTimeFormat(undefined, {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
        timeZoneName: "short"
      }).format(date);
    } catch (e) {
      return date.toLocaleString();
    }
  }

  function normalizeTag(tag) {
    return String(tag || "").trim().toUpperCase();
  }

  function isHighTag(tag) {
    var normalized = normalizeTag(tag);
    if (!normalized) {
      return false;
    }
    return (
      normalized.indexOf("CRITICAL") >= 0 ||
      normalized.indexOf("ERROR") >= 0 ||
      normalized.indexOf("WARN") >= 0 ||
      normalized.indexOf("FATAL") >= 0 ||
      normalized.indexOf("ALERT") >= 0
    );
  }

  function levelClass(tag) {
    return isHighTag(tag) ? "level level-high" : "level level-normal";
  }

  function levelText(tag, level) {
    var normalized = normalizeTag(tag);
    if (normalized) {
      return normalized;
    }
    if (level === null || level === undefined || level === "") {
      return "UNKNOWN";
    }
    return "LEVEL_" + String(level);
  }

  function accountLabel(item) {
    var name = String(item.account_name || "").trim();
    var key = String(item.account_key || "").trim();
    var label = name || key;
    if (!label) {
      return "";
    }
    return "[" + label + "] ";
  }

  // 只展示值班需要的核心字段，减少 Kindle 页面占用。
  function renderAlerts(alerts) {
    listEl.innerHTML = "";

    if (!alerts || alerts.length === 0) {
      emptyEl.style.display = "block";
      return;
    }

    emptyEl.style.display = "none";

    for (var i = 0; i < alerts.length; i++) {
      var item = alerts[i];
      var li = document.createElement("li");
      li.className = "alert-item";

      var ruleName = escapeHTML(item.rule_name || "未命名规则");
      var accountPrefix = escapeHTML(accountLabel(item));
      var instanceID = escapeHTML(item.instance_id || "-");
      var level = Number(item.level || 0);
      var alertTag = item.alert_tag || "";
      var levelTag = "<span class='" + levelClass(alertTag) + "'>" + escapeHTML(levelText(alertTag, level)) + "</span>";
      var maximumValue = escapeHTML(item.alert_value_maximum || "-");
      var alertTime = escapeHTML(formatDeviceTime(item.last_alert_time_utc));

      li.innerHTML =
        "<p class='alert-title'>" + levelTag + accountPrefix + ruleName + "</p>" +
        "<p class='alert-meta'>实例：" + instanceID + "</p>" +
        "<p class='alert-meta'>报警值(Maximum)：" + maximumValue + "</p>" +
        "<p class='alert-meta'>报警时间(本地)：" + alertTime + "</p>";

      listEl.appendChild(li);
    }
  }

  // 按后端下发刷新间隔轮询，便于低频更新电子墨水屏。
  function scheduleRefresh() {
    if (refreshTimer) {
      clearInterval(refreshTimer);
    }
    refreshTimer = setInterval(loadAlerts, refreshMs);
  }

  // 调用后端 API 获取最新告警并刷新页面。
  function loadAlerts() {
    setMessage("加载中...");

    var xhr = new XMLHttpRequest();
    xhr.open("GET", "/api/alerts", true);
    xhr.timeout = 15000;

    xhr.onreadystatechange = function () {
      if (xhr.readyState !== 4) {
        return;
      }

      if (xhr.status < 200 || xhr.status >= 300) {
        setMessage("加载失败，请稍后重试");
        return;
      }

      var payload;
      try {
        payload = JSON.parse(xhr.responseText);
      } catch (e) {
        setMessage("数据解析失败");
        return;
      }

      totalEl.textContent = String(payload.total || 0);
      accountSuccessEl.textContent = String(payload.accounts_success || 0);
      accountTotalEl.textContent = String(payload.accounts_total || 0);
      updatedEl.textContent = formatDeviceTime(payload.ts_utc || "");

      if (payload.refresh_seconds && payload.refresh_seconds > 0) {
        refreshMs = payload.refresh_seconds * 1000;
        scheduleRefresh();
      }

      renderAlerts(payload.alerts || []);

      var failed = Number(payload.accounts_failed || 0);
      var totalAccounts = Number(payload.accounts_total || 0);
      if (failed > 0) {
        setMessage("部分账号失败：" + failed + "/" + totalAccounts);
      } else {
        setMessage("");
      }
    };

    xhr.ontimeout = function () {
      setMessage("请求超时，请稍后重试");
    };

    xhr.onerror = function () {
      setMessage("网络错误，请检查连接");
    };

    xhr.send();
  }

  refreshBtn.addEventListener("click", function () {
    loadAlerts();
  });

  loadAlerts();
  scheduleRefresh();
})();
