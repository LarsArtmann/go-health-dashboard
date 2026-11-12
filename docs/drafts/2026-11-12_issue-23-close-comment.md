Shipped in v0.13.3 (`5e39639`, tagged 2026-11-12): `<html>` now carries `data-nonce="<nonce>"` whenever a CSP nonce is wired (issue #23). The Datastar SDK reads the page nonce from the root element, so its expressions evaluate without the `new Function` path that a strict CSP forbids.

- The nonce value was already forwarded to every `<script>` tag (v0.13.1's `WithNonceExtractor`/`WithNonce`); only the root stamp was unreachable — from both the library's `View()` literal and consumer-side renders (`PageProps.HTMLDataAttrs` is now wired through `rootNonceAttrs` in `status.go`).
- Pages served without a nonce stay byte-identical (nil attributes) — pinned by `TestCSP_NoNonceOmitsRootDataAttrs`; the stamped shape is pinned by `TestCSP_NonceStampedOnHTMLRoot`.
- Verified in the runtime browser suite: the page self-hosts under a strict CSP with the nonce wired, the SSE connection opens, and the DOM patches (run 43164966815, Chromium 157).
