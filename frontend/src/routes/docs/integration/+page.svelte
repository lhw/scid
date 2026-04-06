<svelte:head>
  <title>Integration Guide — My SCID Docs</title>
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-6 py-16">
  <nav class="mb-8 text-xs text-[#e2e8f0]/40">
    <a href="/" class="hover:text-[#00d4ff]">Home</a>
    <span class="mx-2">/</span>
    <a href="/docs" class="hover:text-[#00d4ff]">Docs</a>
    <span class="mx-2">/</span>
    <span class="text-[#e2e8f0]/70">Integration Guide</span>
  </nav>

  <header class="mb-12">
    <h1 class="mb-3 text-4xl font-bold text-[#00d4ff]">Integration Guide</h1>
    <p class="text-lg text-[#e2e8f0]/60">Integrating "Login with SCID" into your fan site</p>
  </header>

  <div class="space-y-12">

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Prerequisites</h2>
      <ul class="space-y-2">
        {#each [
          'A verified SCID account (you need to link your RSI handle first)',
          'A registered application in My Apps',
        ] as item}
          <li class="flex items-start gap-2 text-sm text-[#e2e8f0]/70">
            <span class="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-[#00d4ff]/50"></span>
            {item}
          </li>
        {/each}
      </ul>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Registering an application</h2>
      <p class="mb-4 text-sm text-[#e2e8f0]/70 leading-relaxed">
        In <a href="/apps" class="text-[#00d4ff] hover:underline">My Apps</a>, you can configure:
      </p>
      <ul class="space-y-1.5">
        {#each [
          'Application name (alphanumeric, spaces, hyphens)',
          'Launch URL for the public app directory',
          'One or more redirect URIs (https:// or http://localhost)',
          'Optional logout redirect URIs',
          'Public client mode for SPAs and mobile apps (no client secret)',
          'Confidential client mode for server-side apps (uses a client secret)',
          'PKCE requirement (recommended for all clients)',
          'Verified-only access (restricts logins to RSI-verified users)',
          'Optional app listing in the public directory (listing may require approval)',
          'Optional app logo',
        ] as item}
          <li class="flex items-start gap-2 text-sm text-[#e2e8f0]/60">
            <span class="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-[#1e3a5f]"></span>
            {item}
          </li>
        {/each}
      </ul>
      <p class="mt-4 text-sm text-[#e2e8f0]/50">
        New apps only require admin approval if you opt in to the public app directory listing.
        Unlisted clients are usable immediately while listing approval is pending.
      </p>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">OIDC Discovery</h2>
      <p class="mb-4 text-sm text-[#e2e8f0]/70 leading-relaxed">
        The auto-discovery document is available at:
      </p>
      <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4">
        <code class="font-mono text-sm text-[#00d4ff]">https://auth.scid.my/.well-known/openid-configuration</code>
      </div>
      <p class="mt-3 text-sm text-[#e2e8f0]/50">
        All OIDC endpoints (authorization, token, userinfo, JWKS) are listed there.
        Most libraries accept a discovery URL and configure themselves automatically.
      </p>
    </section>

    <section>
      <h2 class="mb-6 text-xl font-semibold text-[#e2e8f0]">Flow: Authorization Code + PKCE</h2>
      <p class="mb-6 text-sm text-[#e2e8f0]/70 leading-relaxed">
        My SCID supports <strong class="text-[#e2e8f0]/80">Authorization Code Flow with PKCE</strong>
        (recommended for all clients) and the standard Authorization Code Flow with a client secret.
      </p>

      <div class="space-y-6">
        <div>
          <p class="mb-3 text-sm font-semibold text-[#e2e8f0]/80">Step 1 — Redirect the user to My SCID</p>
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4 overflow-x-auto">
<pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">GET https://auth.scid.my/authorize
  ?response_type=code
  &client_id=YOUR_CLIENT_ID
  &redirect_uri=https://yoursite.example/callback
  &scope=openid profile email
  &code_challenge=BASE64URL(SHA256(code_verifier))
  &code_challenge_method=S256
  &state=RANDOM_CSRF_TOKEN</pre>
          </div>
        </div>

        <div>
          <p class="mb-3 text-sm font-semibold text-[#e2e8f0]/80">Step 2 — Exchange the code for tokens</p>
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4 overflow-x-auto">
<pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">POST https://auth.scid.my/api/oidc/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=AUTH_CODE
&redirect_uri=https://yoursite.example/callback
&client_id=YOUR_CLIENT_ID
&code_verifier=YOUR_CODE_VERIFIER
# Confidential clients also include:
# &client_secret=YOUR_CLIENT_SECRET</pre>
          </div>
          <p class="mt-3 text-xs text-[#e2e8f0]/40">Response includes <code class="font-mono">access_token</code>, <code class="font-mono">id_token</code>, and <code class="font-mono">expires_in</code>.</p>
        </div>

        <div>
          <p class="mb-3 text-sm font-semibold text-[#e2e8f0]/80">Step 3 — Fetch the userinfo</p>
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4 overflow-x-auto">
<pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">GET https://auth.scid.my/api/oidc/userinfo
Authorization: Bearer ACCESS_TOKEN</pre>
          </div>
          <p class="mt-3 text-xs text-[#e2e8f0]/40">
            See the <a href="/docs/claims" class="text-[#00d4ff] hover:underline">OIDC Claims Reference</a> for all returned fields.
          </p>
        </div>
      </div>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Public vs confidential clients</h2>
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-5">
          <p class="mb-2 font-semibold text-[#e2e8f0]">Public client</p>
          <p class="mb-3 text-xs text-[#e2e8f0]/50">For SPAs, mobile apps, or CLI tools that cannot safely store a secret.</p>
          <ul class="space-y-1">
            {#each ['No client secret', 'PKCE required', 'Suitable for JavaScript frontends'] as item}
              <li class="flex items-start gap-2 text-xs text-[#e2e8f0]/60">
                <span class="mt-1 h-1 w-1 shrink-0 rounded-full bg-[#00d4ff]/40"></span>
                {item}
              </li>
            {/each}
          </ul>
        </div>
        <div class="rounded-xl border border-[#00d4ff]/30 bg-[#0d1526] p-5">
          <p class="mb-2 font-semibold text-[#e2e8f0]">Confidential client <span class="ml-1 text-xs font-normal text-[#00d4ff]/60">recommended</span></p>
          <p class="mb-3 text-xs text-[#e2e8f0]/50">For server-side web apps and services that can securely store a client secret.</p>
          <ul class="space-y-1">
            {#each ['Client secret + PKCE', 'Code exchange requires server authentication', 'Stronger security posture'] as item}
              <li class="flex items-start gap-2 text-xs text-[#e2e8f0]/60">
                <span class="mt-1 h-1 w-1 shrink-0 rounded-full bg-[#00d4ff]/60"></span>
                {item}
              </li>
            {/each}
          </ul>
        </div>
      </div>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Restricting access to verified users</h2>
      <p class="mb-4 text-sm text-[#e2e8f0]/70 leading-relaxed">
        Enable "Verified Only" on your OIDC client registration. Users not in the
        <code class="rounded bg-[#1e3a5f] px-1 py-0.5 font-mono text-[#00d4ff]">verified</code> group
        will be denied access before the consent screen.
      </p>
      <p class="mb-3 text-sm text-[#e2e8f0]/60">
        You can also check the <code class="font-mono text-xs text-[#00d4ff]">groups</code> claim in your application:
      </p>
      <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4">
<pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">"groups": ["verified", "rsi:SPAWO"]</pre>
      </div>
      <p class="mt-3 text-sm text-[#e2e8f0]/50">
        If <code class="font-mono text-xs">"verified"</code> is present in the <code class="font-mono text-xs">groups</code> claim,
        the user has a confirmed RSI identity.
      </p>
    </section>

  </div>

  <div class="mt-12 flex items-center justify-between border-t border-[#1e3a5f] pt-6">
    <a href="/docs" class="text-sm text-[#e2e8f0]/40 hover:text-[#00d4ff]">← Documentation</a>
    <a href="/docs/claims" class="text-sm text-[#e2e8f0]/40 hover:text-[#00d4ff]">Claims Reference →</a>
  </div>
</div>
