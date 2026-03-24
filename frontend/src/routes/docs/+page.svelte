<svelte:head>
  <title>Documentation — My SCID</title>
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-6 py-16">
  <nav class="mb-8 text-xs text-[#e2e8f0]/40">
    <a href="/" class="hover:text-[#00d4ff]">Home</a>
    <span class="mx-2">/</span>
    <span class="text-[#e2e8f0]/70">Docs</span>
  </nav>

  <header class="mb-12">
    <h1 class="mb-3 text-4xl font-bold text-[#00d4ff]">My SCID</h1>
    <p class="text-lg text-[#e2e8f0]/60">Unofficial Star Citizen Identity Provider</p>
  </header>

  <div class="prose-scid space-y-8">
    <p class="text-[#e2e8f0]/80 leading-relaxed">
      My SCID lets fan sites offer <strong class="text-[#e2e8f0]">"Login with SCID"</strong> — a standard
      OpenID Connect login backed by verified RSI account ownership. No RSI passwords are ever shared.
    </p>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">How it works</h2>
      <ol class="space-y-3">
        {#each [
          'A user creates a SCID account with a passkey (via Pocket ID).',
          'They verify ownership of their RSI handle by placing a short token in their public RSI bio.',
          'SCID scrapes the public RSI profile and organisations pages, then stores the result on the Pocket ID account.',
          'Fan sites integrate SCID as a standard OIDC provider — any OIDC client library works.',
          'When a user logs in, the fan site receives the verified RSI handle and org memberships as standard JWT claims.',
        ] as step, i}
          <li class="flex items-start gap-3">
            <span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-[#00d4ff]/30 bg-[#00d4ff]/10 text-xs font-bold text-[#00d4ff]">{i + 1}</span>
            <span class="text-sm text-[#e2e8f0]/70">{step}</span>
          </li>
        {/each}
      </ol>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Guides</h2>
      <div class="grid gap-4 sm:grid-cols-2">
        <a
          href="/docs/integration"
          class="group rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-6 transition-colors hover:border-[#00d4ff]/40"
        >
          <p class="mb-1 font-semibold text-[#e2e8f0] group-hover:text-[#00d4ff]">Integration Guide</p>
          <p class="text-sm text-[#e2e8f0]/50">OIDC client setup for fan site operators — endpoints, flows, and examples.</p>
        </a>
        <a
          href="/docs/claims"
          class="group rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-6 transition-colors hover:border-[#00d4ff]/40"
        >
          <p class="mb-1 font-semibold text-[#e2e8f0] group-hover:text-[#00d4ff]">OIDC Claims Reference</p>
          <p class="text-sm text-[#e2e8f0]/50">All claims returned in ID tokens and the userinfo endpoint.</p>
        </a>
      </div>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Quick start (fan site operators)</h2>
      <ol class="space-y-3">
        {#each [
          'Log in to SCID with a verified account.',
          'Go to My Apps and register a new application.',
          'Fill in your application name, launch URL, redirect URIs, and any optional logout URIs.',
          'Choose whether the client is public, PKCE-only, verified-only, and whether it should be listed in the public directory.',
          'Submit for approval if the instance requires it, then use the issued Client ID and Client Secret with any OIDC library.',
        ] as step, i}
          <li class="flex items-start gap-3">
            <span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-[#1e3a5f] text-xs font-bold text-[#e2e8f0]/40">{i + 1}</span>
            <span class="text-sm text-[#e2e8f0]/70">{step}</span>
          </li>
        {/each}
      </ol>
      <div class="mt-5 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4">
        <p class="mb-1 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Discovery URL</p>
        <code class="font-mono text-sm text-[#00d4ff]">https://auth.scid.my/.well-known/openid-configuration</code>
      </div>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">What the companion service stores</h2>
      <ul class="space-y-2">
        {#each [
          'RSI handle',
          'Verification timestamp',
          'RSI enlistment date',
          'UEE Citizen Record number (when available)',
          'RSI org membership as namespaced Pocket ID groups',
          'Cached org metadata and org logos',
        ] as item}
          <li class="flex items-center gap-2 text-sm text-[#e2e8f0]/60">
            <span class="h-1 w-1 shrink-0 rounded-full bg-[#00d4ff]/40"></span>
            {item}
          </li>
        {/each}
      </ul>
    </section>
  </div>

  <footer class="mt-16 border-t border-[#1e3a5f] pt-6 text-xs text-[#e2e8f0]/30">
    This is an unofficial fan project — not affiliated with Cloud Imperium Games.
  </footer>
</div>
