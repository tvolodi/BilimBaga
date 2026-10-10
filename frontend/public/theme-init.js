// FR-BB321 AC-3 / D-4: applies the theme before first paint. Same rules as ThemeProvider.tsx
// (storage key bb-theme; light | dark | system; invalid or missing value means system).
// Classic blocking script, same origin, no requests. Keep the two in step.
(function () {
  try {
    var stored = null;
    try {
      stored = window.localStorage.getItem('bb-theme');
    } catch (e) {
      stored = null; // storage blocked: follow the OS
    }
    var dark;
    if (stored === 'dark' || stored === 'light') {
      dark = stored === 'dark';
    } else {
      dark = !!(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
    }
    var root = document.documentElement;
    root.classList.toggle('dark', dark);
    root.style.colorScheme = dark ? 'dark' : 'light';
  } catch (e) {
    // Anything unexpected leaves the light default in place.
  }
})();
