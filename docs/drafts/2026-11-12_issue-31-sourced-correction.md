Correction to my close comment: the "UA rule changed vs your 156 measurement" claim is wrong. There is no Chromium version boundary:

- The spec rule for `[hidden]` is plain `display: none` — never `!important` ([html spec, 4.4.10](https://html.spec.whatwg.org/multipage/interactive-elements.html#the-details-element) rendering section). The `!important` variant was proposed and REJECTED — [WebKit bug 152883](https://bugs.webkit.org/show_bug.cgi?id=152883).
- Chromium's UA stylesheet (html.css) has no `[hidden]` selector at all; the attribute maps as a presentational hint, which loses to ANY author-level `display` declaration in every engine and every version.

So `inline-flex` defeated the bare attribute on 156 AND 157 — the bug reproduces everywhere. My earlier "on 157 the bare attribute already wins" reading was the test harness's compensating `[hidden] { display: none !important }` rule masking it locally; that rule was removed later the same day (2026-11-11) to make the harness production-honest, and `TestBrowser_ConnectionPill` now asserts exactly-one-visible-state via computed style, which catches the bug on any host.

The class-pair fix is unchanged and correct — it just is not browser-version-dependent, and never was.
