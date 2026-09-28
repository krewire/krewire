// =============================================================================
// Krewire Documentation — Interactive Reader Enhancements (docs.js)
// Zero external dependencies — pure fast vanilla JS
// =============================================================================
(function () {
  'use strict';

  // ── 1. Code Block Copy Buttons & Language Badges ─────────────────────────
  function initCodeBlocks() {
    document.querySelectorAll('.chapter pre').forEach(function (pre) {
      if (pre.querySelector('.code-copy-btn')) return;

      var btn = document.createElement('button');
      btn.className = 'code-copy-btn';
      btn.setAttribute('aria-label', 'Copy code to clipboard');
      btn.innerHTML = '<span class="copy-txt">Copy</span>';

      btn.addEventListener('click', function () {
        var code = pre.querySelector('code');
        var text = (code ? code.innerText : pre.innerText).replace(/\n$/, '');

        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).then(function () {
            btn.innerHTML = '<span class="copy-txt copied">✓ Copied!</span>';
            setTimeout(function () {
              btn.innerHTML = '<span class="copy-txt">Copy</span>';
            }, 2000);
          });
        } else {
          var ta = document.createElement('textarea');
          ta.value = text;
          ta.style.position = 'fixed';
          ta.style.opacity = '0';
          document.body.appendChild(ta);
          ta.select();
          document.execCommand('copy');
          document.body.removeChild(ta);
          btn.innerHTML = '<span class="copy-txt copied">✓ Copied!</span>';
          setTimeout(function () {
            btn.innerHTML = '<span class="copy-txt">Copy</span>';
          }, 2000);
        }
      });

      pre.appendChild(btn);
    });
  }

  // ── 2. Heading Anchor Links ──────────────────────────────────────────────
  function initHeadingAnchors() {
    document.querySelectorAll('.chapter h2, .chapter h3').forEach(function (h) {
      if (!h.id || h.querySelector('.anchor-link')) return;

      var a = document.createElement('a');
      a.className = 'anchor-link';
      a.href = '#' + h.id;
      a.innerHTML = '#';
      a.setAttribute('aria-label', 'Permalink to ' + h.innerText);
      a.style.marginLeft = '0.4rem';
      a.style.color = 'var(--primary)';
      a.style.textDecoration = 'none';
      a.style.opacity = '0.35';
      a.style.fontSize = '0.85em';
      a.style.transition = 'opacity 0.15s ease';

      h.addEventListener('mouseenter', function () { a.style.opacity = '1'; });
      h.addEventListener('mouseleave', function () { a.style.opacity = '0.35'; });

      h.appendChild(a);
    });
  }

  // ── 3. Keyboard Pager Navigation (Left / Right Arrows) ───────────────────
  function initKeyboardNav() {
    document.addEventListener('keydown', function (e) {
      // Don't trigger inside inputs or textareas
      var tag = (e.target && e.target.tagName) || '';
      if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target.isContentEditable) return;

      if (e.key === 'ArrowLeft') {
        var prev = document.querySelector('nav.pager a.prev');
        if (prev && prev.href) window.location.href = prev.href;
      } else if (e.key === 'ArrowRight') {
        var next = document.querySelector('nav.pager a.next');
        if (next && next.href) window.location.href = next.href;
      }
    });
  }

  // ── 4. Auto-Run on DOMContentLoaded ──────────────────────────────────────
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      initCodeBlocks();
      initHeadingAnchors();
      initKeyboardNav();
    });
  } else {
    initCodeBlocks();
    initHeadingAnchors();
    initKeyboardNav();
  }
})();
