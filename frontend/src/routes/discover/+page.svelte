<script lang="ts">
  import { onMount } from 'svelte';
  import { AppWindow, ExternalLink, ShieldCheck } from '@lucide/svelte';
  import { listPublicApps } from '$lib/utils/api';
  import type { DirectoryApp } from '$lib/utils/api';

  let apps = $state<DirectoryApp[]>([]);
  let loading = $state(true);
  let error = $state('');

  onMount(async () => {
    try {
      apps = await listPublicApps();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load apps';
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head>
  <title>Discover Apps — My SCID</title>
</svelte:head>

<div class="mx-auto w-full max-w-4xl px-6 py-16">
  <div class="mb-10">
    <h1 class="text-3xl font-bold tracking-tight text-[#e2e8f0]">
      Discover <span class="text-[#00d4ff]">Apps</span>
    </h1>
    <p class="mt-2 text-sm text-[#e2e8f0]/50">
      Fan sites and tools that use SCID for authentication. Click to visit — you can sign in with your SCID account.
    </p>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-[#e2e8f0]/40">Loading…</div>
  {:else if error}
    <div class="rounded-xl border border-red-500/20 bg-red-500/5 px-6 py-4 text-sm text-red-400">{error}</div>
  {:else if apps.length === 0}
    <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] px-8 py-16 text-center">
      <AppWindow class="mx-auto mb-4 h-12 w-12 text-[#00d4ff]/20" />
      <p class="text-[#e2e8f0]/40">No apps listed yet.</p>
      <p class="mt-1 text-xs text-[#e2e8f0]/25">
        App owners can opt into the directory from their app settings once approved.
      </p>
    </div>
  {:else}
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {#each apps as app (app.id)}
        <a
          href={app.launch_url}
          target="_blank"
          rel="noopener noreferrer"
          class="group flex flex-col rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-5 shadow-[0_0_30px_rgba(0,0,0,0.3)] transition-all hover:border-[#00d4ff]/30 hover:bg-[#111827] hover:shadow-[0_0_40px_rgba(0,212,255,0.06)]"
        >
          <div class="mb-4 flex items-center gap-3">
            <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center overflow-hidden rounded-xl border border-[#1e3a5f] bg-[#0a0e1a] transition-colors group-hover:border-[#00d4ff]/20">
              {#if app.has_logo}
                <img
                  src="/api/oidc/clients/{app.id}/logo"
                  alt="{app.name} logo"
                  class="h-full w-full object-contain"
                />
              {:else}
                <AppWindow class="h-6 w-6 text-[#00d4ff]/30" />
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <h2 class="truncate font-semibold text-[#e2e8f0] transition-colors group-hover:text-[#00d4ff]">
                {app.name}
              </h2>
              {#if app.verified_only}
                <div class="mt-0.5 flex items-center gap-1 text-xs text-emerald-400/80">
                  <ShieldCheck class="h-3 w-3" />
                  Verified citizens only
                </div>
              {/if}
            </div>
          </div>

          <div class="mt-auto flex items-center justify-between">
            <span class="truncate text-xs text-[#e2e8f0]/30">{app.launch_url.replace(/^https?:\/\//, '')}</span>
            <ExternalLink class="ml-2 h-3.5 w-3.5 flex-shrink-0 text-[#00d4ff]/40 transition-colors group-hover:text-[#00d4ff]/70" />
          </div>
        </a>
      {/each}
    </div>
  {/if}

  <div class="mt-10 text-center text-xs text-[#e2e8f0]/25">
    Running a fan site? <a href="/apps" class="text-[#00d4ff]/50 hover:text-[#00d4ff] transition-colors">Register your app →</a>
  </div>
</div>
