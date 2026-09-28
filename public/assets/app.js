// =============================================================================
// Krewire — Core Application & Interactive Script (app.js)
// Zero external dependencies — pure fast vanilla JS
// =============================================================================

// ── 1. Theme Management (runs immediately in <head> to prevent FOUC) ─────────
(function () {
  'use strict';
  try {
    var stored = localStorage.getItem('krewire-theme') || 'auto';
    var mode = stored === 'auto'
      ? (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : stored;
    document.documentElement.dataset.theme = mode;
    document.documentElement.classList.toggle('dark', mode === 'dark');
  } catch (e) {}

  window.krewireTheme = {
    toggle: function () {
      var cur = document.documentElement.dataset.theme || (document.documentElement.classList.contains('dark') ? 'dark' : 'light');
      var nxt = cur === 'dark' ? 'light' : 'dark';
      document.documentElement.dataset.theme = nxt;
      document.documentElement.classList.toggle('dark', nxt === 'dark');
      try {
        localStorage.setItem('krewire-theme', nxt);
      } catch (e) {}
    }
  };
})();

// ── 2. Copy Command Helper ───────────────────────────────────────────────────
function copyCmd(btn, text) {
  if (!btn) return;
  var orig = btn.innerText;
  function showCopied() {
    btn.innerText = '✓ Copied!';
    setTimeout(function () { btn.innerText = orig; }, 2000);
  }

  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(showCopied).catch(function () {
      fallbackCopy(text, showCopied);
    });
  } else {
    fallbackCopy(text, showCopied);
  }
}

function fallbackCopy(text, cb) {
  try {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.left = '-9999px';
    ta.style.top = '-9999px';
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
    if (cb) cb();
  } catch (err) {}
}

window.copyCmd = copyCmd;

// ── 3. Navbar Mobile Menu & Interactive Handlers ─────────────────────────────
document.addEventListener('DOMContentLoaded', function () {
  var toggler = document.getElementById('nav-toggler');
  var menu = document.getElementById('nav-mobile-menu');

  if (toggler && menu) {
    toggler.addEventListener('click', function (e) {
      e.stopPropagation();
      var open = menu.classList.toggle('menu-open');
      toggler.setAttribute('aria-expanded', open ? 'true' : 'false');
      if (open) {
        menu.classList.remove('opacity-0', 'pointer-events-none', '-translate-y-2');
        menu.classList.add('opacity-100', 'pointer-events-auto', 'translate-y-0');
      } else {
        menu.classList.add('opacity-0', 'pointer-events-none', '-translate-y-2');
        menu.classList.remove('opacity-100', 'pointer-events-auto', 'translate-y-0');
      }
    });

    // Close when clicking any link inside menu
    menu.querySelectorAll('a').forEach(function (link) {
      link.addEventListener('click', function () {
        menu.classList.add('opacity-0', 'pointer-events-none', '-translate-y-2');
        menu.classList.remove('opacity-100', 'pointer-events-auto', 'translate-y-0');
        menu.classList.remove('menu-open');
        toggler.setAttribute('aria-expanded', 'false');
      });
    });

    // Close when clicking anywhere outside
    document.addEventListener('click', function (e) {
      if (!menu.contains(e.target) && !toggler.contains(e.target)) {
        if (menu.classList.contains('menu-open')) {
          menu.classList.add('opacity-0', 'pointer-events-none', '-translate-y-2');
          menu.classList.remove('opacity-100', 'pointer-events-auto', 'translate-y-0');
          menu.classList.remove('menu-open');
          toggler.setAttribute('aria-expanded', 'false');
        }
      }
    });
  }

  // Also bind any buttons with [data-theme-toggle]
  document.querySelectorAll('[data-theme-toggle]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      if (window.krewireTheme && window.krewireTheme.toggle) {
        window.krewireTheme.toggle();
      }
    });
  });
});
