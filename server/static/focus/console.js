// Focus Console client — vanilla JS, no build step. Server-rendered HTML
// is the source of truth; this script only appends live stream items,
// flips density, opens the mobile sheet, and reports connectivity.
// It never invents state: every card comes from an SSE event.
(function () {
  'use strict';
  var $ = function (s) { return document.querySelector(s); };
  var esc = function (s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  };
  var dev = function () { return document.documentElement.dataset.persona === 'dev'; };

  /* ---------- density (display only; same DOM, same state) ---------- */
  function setPersona(p) {
    document.documentElement.dataset.persona = p;
    try { localStorage.setItem('focus-persona', p); } catch (e) { /* private mode */ }
    var ops = $('#pOps'), d = $('#pDev');
    if (ops && d) {
      ops.classList.toggle('on', p !== 'dev');
      d.classList.toggle('on', p === 'dev');
    }
  }
  var saved = null;
  try { saved = localStorage.getItem('focus-persona'); } catch (e) {}
  setPersona(saved === 'dev' ? 'dev' : 'ops');
  var pOps = $('#pOps'), pDev = $('#pDev');
  if (pOps) pOps.addEventListener('click', function () { setPersona('ops'); });
  if (pDev) pDev.addEventListener('click', function () { setPersona('dev'); });

  /* ---------- mobile pulse sheet ---------- */
  var lastFocused = null;
  function openPulse(on) {
    var pulse = $('#pulse'), overlay = $('#overlay');
    if (on && pulse && !pulse.classList.contains('open')) {
      lastFocused = document.activeElement;
    }
    if (pulse) pulse.classList.toggle('open', on);
    if (overlay) overlay.classList.toggle('on', on);
    if (on) {
      var stream = $('#stream');
      if (stream) stream.setAttribute('tabindex', '-1');
      if (stream) stream.focus({ preventScroll: true });
    } else if (lastFocused && lastFocused.focus) {
      lastFocused.focus({ preventScroll: true });
      lastFocused = null;
    }
  }
  var edge = $('#pulseEdge'), overlay = $('#overlay'), pulseBtn = $('#pulseBtn');
  if (edge) edge.addEventListener('click', function () { openPulse(true); });
  if (overlay) overlay.addEventListener('click', function () { openPulse(false); });
  if (pulseBtn) pulseBtn.addEventListener('click', function () { openPulse(true); });
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape') openPulse(false); });

  /* ---------- pulse filters (display only) ---------- */
  var pfBtns = document.querySelectorAll('#pfilters [data-pf]');
  function applyFilter(f) {
    Array.prototype.forEach.call(pfBtns, function (b) {
      b.classList.toggle('on', b.getAttribute('data-pf') === f);
    });
    var cards = document.querySelectorAll('#stream .ev');
    Array.prototype.forEach.call(cards, function (c) {
      var k = c.getAttribute('data-kind') || 'event';
      c.style.display = (f === 'all' || k === f) ? '' : 'none';
    });
  }
  Array.prototype.forEach.call(pfBtns, function (b) {
    b.addEventListener('click', function () { applyFilter(b.getAttribute('data-pf')); });
  });

  /* ---------- offline banner ---------- */
  var banner = $('#connBanner');
  function setOnline(on) {
    if (!banner) return;
    banner.hidden = on;
    banner.textContent = on ? '' : 'Connection lost — retrying… (stream resumes from the last received event)';
  }
  window.addEventListener('offline', function () { setOnline(false); });
  window.addEventListener('online', function () { setOnline(true); });

  /* ---------- live stream append ---------- */
  var stream = $('#stream');
  if (!stream || !stream.dataset.eventsUrl) return;
  var countEl = $('#pulseCount');

  function cardFor(item) {
    var div = document.createElement('div');
    var kind = 'event', filter = 'event';
    var title = '', detail = '', devd = '';
    if (item.kind === 'message') {
      title = (item.role || 'message') + ' (live)';
      detail = item.body || '';
      devd = 'seq ' + item.seq + ' · ref ' + (item.ref_id || '');
    } else if (item.kind === 'dispatch' && item.payload && typeof item.payload === 'object') {
      var st = item.payload.state || '';
      kind = st === 'acked' ? 'receipt' : 'alert';
      filter = kind;
      title = 'dispatch ' + (item.payload.command_kind || '') + ' ' + st + ' (live)';
      detail = item.payload.reason || '';
      devd = 'seq ' + item.seq + ' · build ' + (item.build_id || '') +
        ' · attempts ' + (item.payload.attempts || 0);
    } else {
      title = (item.type || 'event') + ' (live)';
      if (item.payload && typeof item.payload === 'object') {
        detail = item.payload.text || item.payload.body || '';
      }
      devd = 'seq ' + item.seq + ' · build ' + (item.build_id || '');
    }
    div.className = 'ev ' + kind;
    div.setAttribute('data-seq', item.seq);
    div.setAttribute('data-kind', filter);
    div.innerHTML = '<div class="t">' + esc(title) + '</div>' +
      '<div class="d">' + esc(detail) + '</div>' +
      (devd ? '<div class="devd">' + esc(devd) + '</div>' : '') +
      '<div class="tm">just now</div>';
    return div;
  }

  var src = null;
  try {
    src = new EventSource(stream.dataset.eventsUrl);
  } catch (e) {
    setOnline(false);
    return;
  }
  src.addEventListener('update', function (ev) {
    var item;
    try { item = JSON.parse(ev.data); } catch (e) { return; }
    if (!item || typeof item.seq === 'undefined') return;
    if (stream.querySelector('[data-seq="' + item.seq + '"]')) return; // no dupes on replay
    stream.appendChild(cardFor(item));
    if (countEl && countEl.dataset && countEl.dataset.count) {
      var n = parseInt(countEl.dataset.count, 10) + 1;
      countEl.dataset.count = String(n);
      countEl.textContent = n + ' items';
    }
    setOnline(true);
  });
  src.addEventListener('error', function () {
    if (src.readyState === EventSource.CLOSED) setOnline(false);
    // CONNECTING means the browser is retrying with Last-Event-ID;
    // the banner stays until items flow again.
  });

  /* ---------- line reply state ---------- */
  var replyQID = $('#replyQID'), replyChip = $('#replyChip'),
      replyLabel = $('#replyLabel'), replyCancel = $('#replyCancel'),
      lineInput = $('#lineInput');
  function clearReply() {
    if (replyQID) replyQID.value = '';
    if (replyChip) replyChip.hidden = true;
  }
  document.addEventListener('click', function (e) {
    var b = e.target.closest('[data-reply-qid]');
    if (!b || !replyQID) return;
    replyQID.value = b.getAttribute('data-reply-qid');
    if (replyLabel) replyLabel.textContent = 'Replying to ' + (b.getAttribute('data-reply-kind') || 'question');
    if (replyChip) replyChip.hidden = false;
    if (lineInput) lineInput.focus();
  });
  if (replyCancel) replyCancel.addEventListener('click', clearReply);

  /* ---------- hold-to-confirm kill ---------- */
  var kill = $('#kill'), killForm = $('#killForm'), killFill = $('#killFill');
  if (kill && killForm) {
    kill.disabled = false; // hold timing needs JS; without it the control honestly stays off
    var killT = null, killFired = false;
    var HOLD_MS = 700;
    function killStop() {
      if (killT) { cancelAnimationFrame(killT); killT = null; }
      if (killFill) killFill.style.width = '0';
      killFired = false;
    }
    function killStart(ev) {
      if (ev) ev.preventDefault();
      if (killFired) return;
      killFired = false;
      var t0 = Date.now();
      var step = function () {
        var p = Math.min(1, (Date.now() - t0) / HOLD_MS);
        if (killFill) killFill.style.width = (p * 100) + '%';
        if (p >= 1) {
          killStop();
          killFired = true;
          killForm.submit();
          return;
        }
        killT = requestAnimationFrame(step);
      };
      killT = requestAnimationFrame(step);
    }
    kill.addEventListener('pointerdown', killStart);
    kill.addEventListener('pointerup', killStop);
    kill.addEventListener('pointerleave', killStop);
    kill.addEventListener('keydown', function (e) {
      if ((e.key === ' ' || e.key === 'Enter') && !e.repeat) killStart(e);
    });
    kill.addEventListener('keyup', function (e) {
      if (e.key === ' ' || e.key === 'Enter') { killStop(); killFired = false; }
    });
    killForm.addEventListener('submit', function (e) {
      // The form submits only after a completed hold; anything else
      // (Enter key default, scripted submit) is rejected. No-JS
      // clients never get here: the button stays disabled.
      if (!killFired) e.preventDefault();
    });
  }
})();
