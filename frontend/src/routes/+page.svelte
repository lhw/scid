<script lang="ts">
  import type { PageData } from './$types';
  import { toast } from 'svelte-sonner';
  import { PUBLIC_POCKET_ID_URL } from '$env/static/public';
  import { refreshVerify } from '$lib/utils/api';

  let { data }: { data: PageData } = $props();

  let status = $state(data.status);
  let refreshing = $state(false);

  async function handleRefresh() {
    refreshing = true;
    try {
      status = await refreshVerify();
      toast.success('RSI profile refreshed');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Refresh failed');
    } finally {
      refreshing = false;
    }
  }

  function profilePictureURL(): string {
    if (status?.user_id) return `${PUBLIC_POCKET_ID_URL}/api/users/${status.user_id}/profile-picture.png`;
    return '';
  }
</script>

{#if status?.verified}
  <div class="mx-auto w-full max-w-2xl px-6 py-16">
    <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-8 shadow-[0_0_40px_rgba(0,212,255,0.05)]">
      <div class="mb-6 flex items-center gap-5">
        {#if profilePictureURL()}
          <img
            src={profilePictureURL()}
            alt="RSI avatar"
            class="h-20 w-20 rounded-full border-2 border-[#00d4ff]/40 object-cover"
          />
        {:else}
          <div class="flex h-20 w-20 items-center justify-center rounded-full border-2 border-[#00d4ff]/30 bg-[#1e3a5f] text-3xl font-bold text-[#00d4ff]">
            {(status.handle ?? status.username ?? '?')[0].toUpperCase()}
          </div>
        {/if}
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-2xl font-bold text-[#00d4ff]">{status.handle}</h1>
            <span class="rounded-full border border-emerald-500/40 bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400">
              Verified
            </span>
          </div>
          {#if status.username && status.username !== status.handle}
            <p class="text-sm text-[#e2e8f0]/50">{status.username}</p>
          {/if}
        </div>
      </div>

      <dl class="mb-8 grid gap-4 sm:grid-cols-2">
        {#if status.verified_at}
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-4 py-3">
            <dt class="mb-1 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Verified</dt>
            <dd class="text-sm text-[#e2e8f0]/80">{new Date(status.verified_at).toLocaleDateString()}</dd>
          </div>
        {/if}
        {#if status.enlisted}
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-4 py-3">
            <dt class="mb-1 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Enlisted</dt>
            <dd class="text-sm text-[#e2e8f0]/80">{status.enlisted}</dd>
          </div>
        {/if}
        {#if status.citizen_record}
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-4 py-3">
            <dt class="mb-1 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Citizen Record</dt>
            <dd class="font-mono text-sm text-[#e2e8f0]/80">#{status.citizen_record}</dd>
          </div>
        {/if}
      </dl>

      <button
        onclick={handleRefresh}
        disabled={refreshing}
        class="flex items-center gap-2 rounded-lg border border-[#00d4ff]/40 bg-[#00d4ff]/5 px-5 py-2.5 text-sm font-medium text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/10 disabled:cursor-not-allowed disabled:opacity-50"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="h-4 w-4 {refreshing ? 'animate-spin' : ''}"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <polyline points="23 4 23 10 17 10" />
          <polyline points="1 20 1 14 7 14" />
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
        </svg>
        {refreshing ? 'Refreshing…' : 'Refresh RSI info'}
      </button>
    </div>
  </div>

{:else}
  <div class="mx-auto w-full max-w-6xl px-6 py-16">
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

    <section class="grid gap-6 sm:grid-cols-2">
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
{/if}
