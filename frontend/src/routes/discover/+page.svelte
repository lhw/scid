<script lang="ts">
  import { onMount } from 'svelte';
  import { AppWindow, ExternalLink, ShieldCheck, Sparkles, X, ChevronLeft, ChevronRight } from '@lucide/svelte';
  import { listPublicApps, listScreenshots, screenshotUrl } from '$lib/utils/api';
  import type { DirectoryApp, ScreenshotMeta } from '$lib/utils/api';

  const CATEGORIES: { slug: string; label: string }[] = [
    { slug: 'community', label: 'Community' },
    { slug: 'fleet',     label: 'Fleet & Org' },
    { slug: 'trading',   label: 'Trading' },
    { slug: 'roleplay',  label: 'Roleplay' },
    { slug: 'stats',     label: 'Stats & Data' },
    { slug: 'tools',     label: 'Tools & Utilities' },
  ];

  const RONDELL_COUNT = 6;

  let apps = $state<DirectoryApp[]>([]);
  let loading = $state(true);
  let error = $state('');
  let activeCategory = $state('');

  // Modal state
  let selectedApp = $state<DirectoryApp | null>(null);
  let screenshotList = $state<ScreenshotMeta[]>([]);
  let carouselIndex = $state(0);

  $effect(() => {
    if (selectedApp) {
      carouselIndex = 0;
      listScreenshots(selectedApp.id).then(s => { screenshotList = s; }).catch(() => { screenshotList = []; });
    } else {
      screenshotList = [];
      carouselIndex = 0;
    }
  });

  function openApp(app: DirectoryApp) {
    selectedApp = app;
  }

  function closeModal() {
    selectedApp = null;
  }

  function prevScreenshot() {
    if (carouselIndex > 0) carouselIndex--;
  }

  function nextScreenshot() {
    if (screenshotList.length > 0 && carouselIndex < screenshotList.length - 1) carouselIndex++;
  }

  let newestApps = $derived(
    [...apps]
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, RONDELL_COUNT)
  );

  let filteredApps = $derived(
    activeCategory
      ? apps.filter(a => a.category === activeCategory)
      : apps
  );

  onMount(async () => {
    try {
      const raw = await listPublicApps();
      // Randomise order within each category bucket so no app gets permanent top placement.
      const shuffled = [...raw];
      for (let i = shuffled.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
      }
      apps = shuffled;
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

<div class="mx-auto w-full max-w-5xl px-6 py-16">
  <!-- Header -->
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

    <!-- ── Newest Apps Rondell ───────────────────────────────── -->
    {#if newestApps.length > 0}
      <section class="mb-12">
        <div class="mb-4 flex items-center gap-2">
          <Sparkles class="h-4 w-4 text-[#00d4ff]/60" />
          <h2 class="text-sm font-semibold uppercase tracking-widest text-[#e2e8f0]/50">Newest</h2>
        </div>
        <div
          class="flex snap-x snap-mandatory gap-4 overflow-x-auto pb-3 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
        >
          {#each newestApps as app (app.id)}
            <button
              type="button"
              onclick={() => openApp(app)}
              class="group relative flex w-64 flex-shrink-0 snap-start flex-col rounded-2xl border border-[#1e3a5f] bg-gradient-to-br from-[#0d1526] to-[#0a0e1a] p-5 text-left shadow-[0_0_30px_rgba(0,0,0,0.4)] transition-all hover:border-[#00d4ff]/40 hover:shadow-[0_0_40px_rgba(0,212,255,0.08)]"
            >
              <!-- glow accent -->
              <div class="pointer-events-none absolute inset-0 rounded-2xl bg-[radial-gradient(ellipse_at_top_left,rgba(0,212,255,0.04)_0%,transparent_70%)]"></div>

              <div class="mb-4 flex items-center gap-3">
                <div class="flex h-11 w-11 flex-shrink-0 items-center justify-center overflow-hidden rounded-xl border border-[#1e3a5f] bg-[#060a14] transition-colors group-hover:border-[#00d4ff]/20">
                  {#if app.has_logo}
                    <img src="/api/oidc/clients/{app.id}/logo" alt="{app.name} logo" class="h-full w-full object-contain" />
                  {:else}
                    <AppWindow class="h-5 w-5 text-[#00d4ff]/30" />
                  {/if}
                </div>
                <div class="min-w-0">
                  <p class="truncate text-sm font-semibold text-[#e2e8f0] transition-colors group-hover:text-[#00d4ff]">{app.name}</p>
                  {#if app.verified_only}
                    <p class="mt-0.5 flex items-center gap-1 text-xs text-emerald-400/80">
                      <ShieldCheck class="h-3 w-3" /> Verified only
                    </p>
                  {/if}
                </div>
              </div>

              {#if app.description}
                <p class="mb-3 line-clamp-2 text-xs text-[#e2e8f0]/50">{app.description}</p>
              {/if}

              <div class="mt-auto flex items-center justify-between">
                <span class="truncate text-xs text-[#e2e8f0]/25">{app.launch_url.replace(/^https?:\/\//, '')}</span>
                <ExternalLink class="ml-2 h-3 w-3 flex-shrink-0 text-[#00d4ff]/30 transition-colors group-hover:text-[#00d4ff]/60" />
              </div>
            </button>
          {/each}
        </div>
      </section>
    {/if}

    <!-- ── Category Filters ──────────────────────────────────── -->
    <div class="mb-6 flex flex-wrap gap-2">
      <button
        onclick={() => (activeCategory = '')}
        class="rounded-full border px-4 py-1.5 text-xs font-medium transition-colors {activeCategory === ''
          ? 'border-[#00d4ff]/60 bg-[#00d4ff]/10 text-[#00d4ff]'
          : 'border-[#1e3a5f] text-[#e2e8f0]/50 hover:border-[#00d4ff]/30 hover:text-[#e2e8f0]/80'}"
      >
        All
      </button>
      {#each CATEGORIES as cat (cat.slug)}
        {@const count = apps.filter(a => a.category === cat.slug).length}
        {#if count > 0}
          <button
            onclick={() => (activeCategory = activeCategory === cat.slug ? '' : cat.slug)}
            class="rounded-full border px-4 py-1.5 text-xs font-medium transition-colors {activeCategory === cat.slug
              ? 'border-[#00d4ff]/60 bg-[#00d4ff]/10 text-[#00d4ff]'
              : 'border-[#1e3a5f] text-[#e2e8f0]/50 hover:border-[#00d4ff]/30 hover:text-[#e2e8f0]/80'}"
          >
            {cat.label} <span class="ml-1 opacity-50">{count}</span>
          </button>
        {/if}
      {/each}
    </div>

    <!-- ── App Grid ──────────────────────────────────────────── -->
    {#if filteredApps.length === 0}
      <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] px-8 py-12 text-center">
        <p class="text-sm text-[#e2e8f0]/40">No apps in this category yet.</p>
      </div>
    {:else}
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {#each filteredApps as app (app.id)}
          <button
            type="button"
            onclick={() => openApp(app)}
            class="group flex flex-col rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-5 text-left shadow-[0_0_30px_rgba(0,0,0,0.3)] transition-all hover:border-[#00d4ff]/30 hover:bg-[#111827] hover:shadow-[0_0_40px_rgba(0,212,255,0.06)]"
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
                <div class="mt-0.5 flex flex-wrap items-center gap-1.5">
                  {#if app.verified_only}
                    <span class="flex items-center gap-1 text-xs text-emerald-400/80">
                      <ShieldCheck class="h-3 w-3" /> Verified only
                    </span>
                  {/if}
                  {#if app.category}
                    <span class="rounded-full border border-[#1e3a5f] px-2 py-0.5 text-xs text-[#e2e8f0]/35">
                      {CATEGORIES.find(c => c.slug === app.category)?.label ?? app.category}
                    </span>
                  {/if}
                </div>
              </div>
            </div>

            {#if app.description}
              <p class="mb-3 text-xs text-[#e2e8f0]/50 line-clamp-2">{app.description}</p>
            {/if}

            <div class="mt-auto flex items-center justify-between">
              <span class="truncate text-xs text-[#e2e8f0]/30">{app.launch_url.replace(/^https?:\/\//, '')}</span>
              <ExternalLink class="ml-2 h-3.5 w-3.5 flex-shrink-0 text-[#00d4ff]/40 transition-colors group-hover:text-[#00d4ff]/70" />
            </div>
          </button>
        {/each}
      </div>
    {/if}

  {/if}

  <div class="mt-10 text-center text-xs text-[#e2e8f0]/25">
    Running a fan site? <a href="/apps" class="text-[#00d4ff]/50 hover:text-[#00d4ff] transition-colors">Register your app →</a>
  </div>
</div>

<!-- ── App Detail Modal ──────────────────────────────────────── -->
{#if selectedApp}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
    role="dialog"
    aria-modal="true"
    aria-label="{selectedApp.name} details"
  >
    <!-- Click-outside overlay -->
    <button
      type="button"
      class="absolute inset-0 bg-black/70 backdrop-blur-sm"
      onclick={closeModal}
      aria-label="Close"
    ></button>

    <!-- Panel -->
    <div class="relative z-10 w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-2xl border border-[#1e3a5f] bg-[#0d1526] shadow-[0_0_80px_rgba(0,0,0,0.7)]">
      <!-- Close button -->
      <button
        type="button"
        onclick={closeModal}
        class="absolute right-4 top-4 rounded-lg border border-[#1e3a5f] p-1.5 text-[#e2e8f0]/40 transition-colors hover:border-[#00d4ff]/40 hover:text-[#e2e8f0]"
        aria-label="Close"
      >
        <X class="h-4 w-4" />
      </button>

      <div class="p-6">
        <!-- Header -->
        <div class="flex items-start gap-4 pr-10">
          <div class="flex h-16 w-16 flex-shrink-0 items-center justify-center overflow-hidden rounded-xl border border-[#1e3a5f] bg-[#0a0e1a]">
            {#if selectedApp.has_logo}
              <img src="/api/oidc/clients/{selectedApp.id}/logo" alt="{selectedApp.name} logo" class="h-full w-full object-contain" />
            {:else}
              <AppWindow class="h-8 w-8 text-[#00d4ff]/30" />
            {/if}
          </div>
          <div class="min-w-0 flex-1">
            <h2 class="text-xl font-bold text-[#e2e8f0]">{selectedApp.name}</h2>
            <div class="mt-1 flex flex-wrap items-center gap-2">
              {#if selectedApp.verified_only}
                <span class="flex items-center gap-1 text-xs text-emerald-400/80">
                  <ShieldCheck class="h-3 w-3" /> Verified only
                </span>
              {/if}
              {#if selectedApp.category}
                <span class="rounded-full border border-[#1e3a5f] px-2 py-0.5 text-xs text-[#e2e8f0]/40">
                  {CATEGORIES.find(c => c.slug === selectedApp!.category)?.label ?? selectedApp.category}
                </span>
              {/if}
            </div>
          </div>
        </div>

        <!-- Short description -->
        {#if selectedApp.description}
          <p class="mt-4 text-sm text-[#e2e8f0]/60">{selectedApp.description}</p>
        {/if}

        <!-- Long description -->
        {#if selectedApp.long_description}
          <div class="mt-4 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4">
            <p class="whitespace-pre-wrap text-sm leading-relaxed text-[#e2e8f0]/80">{selectedApp.long_description}</p>
          </div>
        {/if}

        <!-- Screenshot carousel -->
        {#if screenshotList.length > 0}
          <div class="mt-5">
            <div class="relative overflow-hidden rounded-xl border border-[#1e3a5f] bg-[#060a14]">
              <img
                src={screenshotUrl(selectedApp.id, screenshotList[carouselIndex].id)}
                alt="Screenshot {carouselIndex + 1} of {screenshotList.length}"
                class="h-64 w-full object-contain"
              />
              {#if screenshotList.length > 1}
                <button
                  type="button"
                  onclick={prevScreenshot}
                  disabled={carouselIndex === 0}
                  class="absolute left-2 top-1/2 -translate-y-1/2 rounded-lg bg-black/60 p-1.5 text-white transition-opacity hover:bg-black/80 disabled:opacity-30"
                  aria-label="Previous screenshot"
                >
                  <ChevronLeft class="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onclick={nextScreenshot}
                  disabled={carouselIndex === screenshotList.length - 1}
                  class="absolute right-2 top-1/2 -translate-y-1/2 rounded-lg bg-black/60 p-1.5 text-white transition-opacity hover:bg-black/80 disabled:opacity-30"
                  aria-label="Next screenshot"
                >
                  <ChevronRight class="h-4 w-4" />
                </button>
              {/if}
            </div>
            {#if screenshotList.length > 1}
              <div class="mt-2 flex justify-center gap-1.5">
                {#each screenshotList as _, i}
                  <button
                    type="button"
                    onclick={() => { carouselIndex = i; }}
                    class="h-1.5 rounded-full transition-all {i === carouselIndex ? 'w-4 bg-[#00d4ff]' : 'w-1.5 bg-[#1e3a5f] hover:bg-[#00d4ff]/40'}"
                    aria-label="Go to screenshot {i + 1}"
                  ></button>
                {/each}
              </div>
            {/if}
          </div>
        {/if}

        <!-- Action -->
        <div class="mt-6 flex justify-end">
          <a
            href={selectedApp.launch_url}
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center gap-2 rounded-lg border border-[#00d4ff] bg-[#00d4ff]/10 px-5 py-2 text-sm font-medium text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/20"
          >
            Visit App <ExternalLink class="h-3.5 w-3.5" />
          </a>
        </div>
      </div>
    </div>
  </div>
{/if}
