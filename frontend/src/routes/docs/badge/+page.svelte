<svelte:head>
  <title>Badges &amp; Buttons — My SCID Docs</title>
</svelte:head>

<script lang="ts">
  import CopyButton from '$lib/components/CopyButton.svelte';

  const htmlImgButton = `<a href="https://auth.scid.my/authorize?response_type=code&client_id=YOUR_CLIENT_ID&redirect_uri=YOUR_REDIRECT_URI&scope=openid+profile+email&code_challenge=CODE_CHALLENGE&code_challenge_method=S256&state=RANDOM_STATE">
  <img
    src="https://scid.my/badges/login-with-scid.svg"
    alt="Login with SCID"
    width="220"
    height="44"
  />
</a>`;

  const htmlImgLogo = `<a href="https://auth.scid.my/authorize?...">
  <img
    src="https://scid.my/badges/scid-logo.svg"
    alt="SCID"
    width="44"
    height="44"
  />
</a>`;

  const cssButtonCode = `/* Place anywhere in your CSS */
.btn-scid {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  height: 44px;
  padding: 0 20px 0 12px;
  background: linear-gradient(135deg, #0d1526 0%, #111827 100%);
  border: 1px solid rgba(0, 212, 255, 0.35);
  border-radius: 8px;
  color: #e2e8f0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.02em;
  cursor: pointer;
  text-decoration: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.btn-scid:hover {
  border-color: rgba(0, 212, 255, 0.7);
  box-shadow: 0 0 0 1px rgba(0, 212, 255, 0.15), 0 4px 12px rgba(0, 212, 255, 0.1);
}
.btn-scid__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: rgba(0, 212, 255, 0.08);
  border-radius: 5px;
  flex-shrink: 0;
}`;

  const cssButtonHtml = `<a href="YOUR_SCID_AUTH_URL" class="btn-scid">
  <span class="btn-scid__icon">
    <img src="https://scid.my/badges/scid-logo.svg" alt="" width="22" height="22" />
  </span>
  Login with SCID
</a>`;
</script>

<div class="mx-auto w-full max-w-3xl px-6 py-16">
  <nav class="mb-8 text-xs text-[#e2e8f0]/40">
    <a href="/" class="hover:text-[#00d4ff]">Home</a>
    <span class="mx-2">/</span>
    <a href="/docs" class="hover:text-[#00d4ff]">Docs</a>
    <span class="mx-2">/</span>
    <span class="text-[#e2e8f0]/70">Badges &amp; Buttons</span>
  </nav>

  <header class="mb-12">
    <h1 class="mb-3 text-4xl font-bold text-[#00d4ff]">Badges &amp; Buttons</h1>
    <p class="text-lg text-[#e2e8f0]/60">Copy-ready "Login with SCID" assets for your site</p>
  </header>

  <div class="space-y-14">

    <!-- Preview -->
    <section>
      <h2 class="mb-6 text-xl font-semibold text-[#e2e8f0]">Preview</h2>
      <div class="flex flex-wrap items-center gap-6 rounded-xl border border-[#1e3a5f] bg-[#0a0e1a] p-8">
        <!-- Login button -->
        <div class="flex flex-col items-center gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="220" height="44" viewBox="0 0 220 44" role="img" aria-label="Login with SCID">
            <title>Login with SCID</title>
            <defs>
              <linearGradient id="preview-bg" x1="0" y1="0" x2="220" y2="44" gradientUnits="userSpaceOnUse">
                <stop offset="0%" stop-color="#0d1526"/>
                <stop offset="100%" stop-color="#111827"/>
              </linearGradient>
              <linearGradient id="preview-border" x1="0" y1="0" x2="220" y2="44" gradientUnits="userSpaceOnUse">
                <stop offset="0%" stop-color="#00d4ff" stop-opacity="0.6"/>
                <stop offset="100%" stop-color="#1e3a5f" stop-opacity="0.8"/>
              </linearGradient>
            </defs>
            <rect width="220" height="44" rx="8" fill="url(#preview-bg)"/>
            <rect x="0.5" y="0.5" width="219" height="43" rx="7.5" fill="none" stroke="url(#preview-border)" stroke-width="1"/>
            <rect x="1" y="1" width="43" height="42" rx="7" fill="#00d4ff" fill-opacity="0.08"/>
            <svg x="2" y="2" width="40" height="40" viewBox="0 0 32 32" fill="none">
              <polygon points="16,4 26,12 22,28 10,28 6,12" fill="none" stroke="#00d4ff" stroke-width="1.5" stroke-linejoin="round"/>
              <polygon points="16,8 23,13 20,26 12,26 9,13" fill="#00d4ff" fill-opacity="0.08"/>
              <text x="16" y="22" text-anchor="middle" font-family="Verdana,Arial,sans-serif" font-size="13" font-weight="700" fill="#00d4ff">S</text>
            </svg>
            <line x1="47" y1="10" x2="47" y2="34" stroke="#1e3a5f" stroke-width="1"/>
            <text x="134" y="27" text-anchor="middle" font-family="-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif" font-size="14" font-weight="600" fill="#e2e8f0" letter-spacing="0.2">Login with SCID</text>
          </svg>
          <span class="text-xs text-[#e2e8f0]/30">Full button (220×44)</span>
        </div>
        <!-- Logo only -->
        <div class="flex flex-col items-center gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="44" height="44" viewBox="0 0 32 32" fill="none" role="img" aria-label="SCID">
            <title>SCID</title>
            <defs>
              <linearGradient id="preview-logo-border" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
                <stop offset="0%" stop-color="#00d4ff" stop-opacity="0.6"/>
                <stop offset="100%" stop-color="#1e3a5f" stop-opacity="0.8"/>
              </linearGradient>
            </defs>
            <rect width="32" height="32" rx="6" fill="#0d1526"/>
            <rect x="0.5" y="0.5" width="31" height="31" rx="5.5" fill="none" stroke="url(#preview-logo-border)" stroke-width="0.75"/>
            <polygon points="16,4 26,12 22,28 10,28 6,12" fill="none" stroke="#00d4ff" stroke-width="1.5" stroke-linejoin="round"/>
            <polygon points="16,8 23,13 20,26 12,26 9,13" fill="#00d4ff" fill-opacity="0.08"/>
            <text x="16" y="22" text-anchor="middle" font-family="Verdana,Arial,sans-serif" font-size="13" font-weight="700" fill="#00d4ff">S</text>
          </svg>
          <span class="text-xs text-[#e2e8f0]/30">Logo mark (44×44)</span>
        </div>
      </div>
    </section>

    <!-- Option 1: download SVG files -->
    <section>
      <h2 class="mb-2 text-xl font-semibold text-[#e2e8f0]">Option 1 — Download SVG files</h2>
      <p class="mb-5 text-sm text-[#e2e8f0]/60 leading-relaxed">
        Download the SVG files and serve them from your own domain. Use them inline, as an
        <code class="font-mono text-xs text-[#00d4ff]">&lt;img&gt;</code> src, or as a CSS background.
      </p>
      <div class="flex flex-wrap gap-4">
        <a
          href="/badges/login-with-scid.svg"
          download="login-with-scid.svg"
          class="flex items-center gap-2 rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-4 py-3 text-sm text-[#e2e8f0]/70 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
        >
          <img src="/badges/login-with-scid.svg" alt="" width="110" height="22" class="opacity-90" />
          <span class="ml-1 text-xs text-[#e2e8f0]/40">login-with-scid.svg</span>
        </a>
        <a
          href="/badges/scid-logo.svg"
          download="scid-logo.svg"
          class="flex items-center gap-2 rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-4 py-3 text-sm text-[#e2e8f0]/70 transition-colors hover:border-[#00d4ff]/40 hover:text-[#00d4ff]"
        >
          <img src="/badges/scid-logo.svg" alt="" width="28" height="28" class="opacity-90" />
          <span class="ml-1 text-xs text-[#e2e8f0]/40">scid-logo.svg</span>
        </a>
      </div>
    </section>

    <!-- Option 2: hosted SVG via img tag -->
    <section>
      <h2 class="mb-2 text-xl font-semibold text-[#e2e8f0]">Option 2 — Hosted SVG <span class="text-sm font-normal text-[#e2e8f0]/40">(img tag)</span></h2>
      <p class="mb-5 text-sm text-[#e2e8f0]/60 leading-relaxed">
        Reference the SVG files hosted on <code class="font-mono text-xs text-[#00d4ff]">scid.my</code> directly.
        Simpler to embed but requires an outbound request to load the image.
      </p>

      <div class="mb-5">
        <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Full login button</p>
        <div class="relative rounded-lg border border-[#1e3a5f] bg-[#0a0e1a]">
          <div class="absolute right-3 top-3">
            <CopyButton text={htmlImgButton} />
          </div>
          <div class="overflow-x-auto p-4 pr-14">
            <pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">{htmlImgButton}</pre>
          </div>
        </div>
      </div>

      <div>
        <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Logo mark only</p>
        <div class="relative rounded-lg border border-[#1e3a5f] bg-[#0a0e1a]">
          <div class="absolute right-3 top-3">
            <CopyButton text={htmlImgLogo} />
          </div>
          <div class="overflow-x-auto p-4 pr-14">
            <pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">{htmlImgLogo}</pre>
          </div>
        </div>
      </div>

      <div class="mt-4 rounded-lg border border-[#1e3a5f] bg-[#0d1526] p-4">
        <p class="mb-1.5 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Hosted asset URLs</p>
        <div class="space-y-1">
          <p class="font-mono text-xs text-[#00d4ff]">https://scid.my/badges/login-with-scid.svg</p>
          <p class="font-mono text-xs text-[#00d4ff]">https://scid.my/badges/scid-logo.svg</p>
        </div>
      </div>
    </section>

    <!-- Option 3: CSS button -->
    <section>
      <h2 class="mb-2 text-xl font-semibold text-[#e2e8f0]">Option 3 — Pure CSS button</h2>
      <p class="mb-5 text-sm text-[#e2e8f0]/60 leading-relaxed">
        If you prefer CSS over SVG, use this class. Add the CSS once and apply
        <code class="font-mono text-xs text-[#00d4ff]">btn-scid</code> to any <code class="font-mono text-xs text-[#00d4ff]">&lt;a&gt;</code> or <code class="font-mono text-xs text-[#00d4ff]">&lt;button&gt;</code>.
      </p>

      <div class="mb-5">
        <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">CSS</p>
        <div class="relative rounded-lg border border-[#1e3a5f] bg-[#0a0e1a]">
          <div class="absolute right-3 top-3">
            <CopyButton text={cssButtonCode} />
          </div>
          <div class="overflow-x-auto p-4 pr-14">
            <pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">{cssButtonCode}</pre>
          </div>
        </div>
      </div>

      <div>
        <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">HTML</p>
        <div class="relative rounded-lg border border-[#1e3a5f] bg-[#0a0e1a]">
          <div class="absolute right-3 top-3">
            <CopyButton text={cssButtonHtml} />
          </div>
          <div class="overflow-x-auto p-4 pr-14">
            <pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">{cssButtonHtml}</pre>
          </div>
        </div>
      </div>
    </section>

  </div>

  <div class="mt-12 flex items-center justify-between border-t border-[#1e3a5f] pt-6">
    <a href="/docs" class="text-sm text-[#e2e8f0]/40 hover:text-[#00d4ff]">← Documentation</a>
    <a href="/docs/integration" class="text-sm text-[#e2e8f0]/40 hover:text-[#00d4ff]">Integration Guide →</a>
  </div>

  <footer class="mt-12 text-xs text-[#e2e8f0]/30">
    This is an unofficial fan project — not affiliated with Cloud Imperium Games.
  </footer>
</div>
