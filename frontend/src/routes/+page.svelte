<script lang="ts">
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();
</script>

<div class="mx-auto w-full max-w-6xl px-6 py-16">
  <!-- Verified banner -->
  {#if data.status?.verified && data.status?.handle}
    <div
      class="mb-10 flex items-center gap-3 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-5 py-3 text-emerald-400"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="h-5 w-5 shrink-0"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
        <polyline points="22 4 12 14.01 9 11.01" />
      </svg>
      <span class="text-sm font-medium">
        ✓ Verified as <span class="font-bold">{data.status.handle}</span>
      </span>
    </div>
  {/if}

  <!-- Hero -->
  <section class="mb-20 text-center">
    <h1
      class="mb-4 bg-gradient-to-r from-[#00d4ff] via-[#38bdf8] to-[#00d4ff] bg-clip-text text-7xl font-extrabold tracking-widest text-transparent"
    >
      SCID
    </h1>
    <p class="mb-6 text-2xl font-light text-[#e2e8f0]/80">
      Your Star Citizen identity, any fan site
    </p>
    <p class="mx-auto mb-10 max-w-2xl text-base leading-relaxed text-[#e2e8f0]/60">
      SCID verifies your RSI account by having you place a short token in your public profile bio —
      no passwords shared, no third-party access granted. Once verified, fan sites can use
      <span class="font-medium text-[#00d4ff]">Login with SCID</span> to recognise you as a real
      citizen, automatically.
    </p>

    <a
      href="/verify"
      class="inline-flex items-center gap-2 rounded-lg border border-[#00d4ff] bg-[#00d4ff]/10 px-8 py-3 text-base font-semibold text-[#00d4ff] shadow-[0_0_24px_rgba(0,212,255,0.15)] transition-all hover:bg-[#00d4ff]/20 hover:shadow-[0_0_32px_rgba(0,212,255,0.25)]"
    >
      Verify Your RSI Identity
      <span aria-hidden="true">→</span>
    </a>
  </section>

  <!-- Feature cards -->
  <section class="grid gap-6 sm:grid-cols-2">
    <!-- Card 1 -->
    <div
      class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-8 transition-colors hover:border-[#00d4ff]/30"
    >
      <div class="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[#00d4ff]/10">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5 text-[#00d4ff]"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        </svg>
      </div>
      <h2 class="mb-3 text-lg font-semibold text-[#e2e8f0]">RSI Bio Verification</h2>
      <p class="text-sm leading-relaxed text-[#e2e8f0]/60">
        SCID generates a unique token (e.g. <code class="rounded bg-[#1e3a5f] px-1 py-0.5 font-mono text-[#00d4ff]">scid:abc123</code>)
        that you paste into your public RSI profile bio. SCID then fetches your profile, confirms the
        token is present, and marks your handle as verified — all without ever touching your RSI
        password or passkeys.
      </p>
    </div>

    <!-- Card 2 -->
    <div
      class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-8 transition-colors hover:border-[#00d4ff]/30"
    >
      <div class="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[#00d4ff]/10">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5 text-[#00d4ff]"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="2" y1="12" x2="22" y2="12" />
          <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
        </svg>
      </div>
      <h2 class="mb-3 text-lg font-semibold text-[#e2e8f0]">One Login, Any Fan Site</h2>
      <p class="text-sm leading-relaxed text-[#e2e8f0]/60">
        SCID is a standard <span class="font-medium text-[#e2e8f0]/80">OpenID Connect (OIDC)</span>
        provider. Any fan site — trade tools, org managers, fleet trackers — can integrate
        "Login with SCID" using any OIDC client library. Your verified RSI handle and org
        memberships are shared as standard claims, no custom integration required.
      </p>
    </div>
  </section>
</div>
