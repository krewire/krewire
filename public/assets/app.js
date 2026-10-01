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

// ── 3. Forge Drawer & Mobile Menu Controller ────────────────────────────────
(function () {
  'use strict';

  function getDrawerElements(id) {
    var drawer = document.getElementById(id);
    var backdrop = document.getElementById(id + '-backdrop') || document.querySelector('[data-drawer-backdrop="' + id + '"]');
    var togglers = document.querySelectorAll('[data-drawer-toggle="' + id + '"], [aria-controls="' + id + '"]');
    return { drawer: drawer, backdrop: backdrop, togglers: togglers };
  }

  window.forgeDrawer = {
    open: function (id) {
      var els = getDrawerElements(id);
      if (!els.drawer) return;
      els.drawer.classList.add('drawer-open');
      if (els.backdrop) els.backdrop.classList.add('backdrop-open');
      els.togglers.forEach(function (btn) {
        btn.setAttribute('aria-expanded', 'true');
        btn.classList.add('is-active');
      });
      document.body.classList.add('drawer-active');
    },
    close: function (id) {
      var els = getDrawerElements(id);
      if (!els.drawer) return;
      els.drawer.classList.remove('drawer-open');
      if (els.backdrop) els.backdrop.classList.remove('backdrop-open');
      els.togglers.forEach(function (btn) {
        btn.setAttribute('aria-expanded', 'false');
        btn.classList.remove('is-active');
      });
      document.body.classList.remove('drawer-active');
    },
    toggle: function (id) {
      var els = getDrawerElements(id);
      if (!els.drawer) return;
      if (els.drawer.classList.contains('drawer-open')) {
        this.close(id);
      } else {
        this.open(id);
      }
    }
  };

  document.addEventListener('DOMContentLoaded', function () {
    // Event delegation for drawers
    document.addEventListener('click', function (e) {
      var toggleBtn = e.target.closest('[data-drawer-toggle]');
      if (toggleBtn) {
        var id = toggleBtn.getAttribute('data-drawer-toggle');
        if (id) {
          e.preventDefault();
          window.forgeDrawer.toggle(id);
          return;
        }
      }

      var closeBtn = e.target.closest('[data-drawer-close]');
      if (closeBtn) {
        var id = closeBtn.getAttribute('data-drawer-close');
        if (id) {
          e.preventDefault();
          window.forgeDrawer.close(id);
          return;
        }
      }

      var backdrop = e.target.closest('[data-drawer-backdrop]');
      if (backdrop) {
        var id = backdrop.getAttribute('data-drawer-backdrop');
        if (id) {
          e.preventDefault();
          window.forgeDrawer.close(id);
          return;
        }
      }

      // Close drawer when clicking any link inside drawer body
      var drawerLink = e.target.closest('.drawer a');
      if (drawerLink) {
        var drawer = drawerLink.closest('.drawer');
        if (drawer && drawer.id) {
          window.forgeDrawer.close(drawer.id);
        }
      }
    });

    // Close on Escape key
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') {
        document.querySelectorAll('.drawer.drawer-open').forEach(function (d) {
          window.forgeDrawer.close(d.id);
        });
      }
    });

    // Theme switches
    document.querySelectorAll('[data-theme-toggle]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        if (window.krewireTheme && window.krewireTheme.toggle) {
          window.krewireTheme.toggle();
        }
      });
    });
  });
})();
