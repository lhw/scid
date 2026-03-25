<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { Plus, AppWindow, ShieldCheck, Trash2, Eye, ChevronRight } from '@lucide/svelte';
  import { goto } from '$app/navigation';
  import type { PageData } from './$types';
  import type { AppRegistration, CreateAppRequest } from '$lib/utils/api';
  import { listApps, createApp, uploadAppLogo } from '$lib/utils/api';

  let { data }: { data: PageData } = $props();

  let apps = $state<AppRegistration[]>([]);
  let loading = $state(true);
  let showForm = $state(false);
  let submitting = $state(false);

  // Form state
  let formName = $state('');
  let formDescription = $state('');
  let formLaunchURL = $state('');
  let formRedirectURIs = $state(['']);
  let formLogoutURIs = $state<string[]>([]);
  let formIsPublic = $state(false);
  let formPkceRequired = $state(false);
  let formVerifiedOnly = $state(false);
  let formLogoFile = $state<File | null>(null);
  let formErrors = $state<Record<string, string>>({});

  const allowedLogoTypes = new Set(['image/png', 'image/jpeg', 'image/webp']);

  onMount(async () => {
    if (!data.status?.verified) {
      loading = false;
      return;
    }
    try {
      apps = await listApps();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load applications');
    } finally {
      loading = false;
    }
  });

  // Auto-enable PKCE when public client is selected
  $effect(() => {
    if (formIsPublic) formPkceRequired = true;
  });

  function addRedirectURI() {
    if (formRedirectURIs.length < 10) formRedirectURIs = [...formRedirectURIs, ''];
  }
  function removeRedirectURI(i: number) {
    formRedirectURIs = formRedirectURIs.filter((_, idx) => idx !== i);
  }
  function addLogoutURI() {
    if (formLogoutURIs.length < 10) formLogoutURIs = [...formLogoutURIs, ''];
  }
  function removeLogoutURI(i: number) {
    formLogoutURIs = formLogoutURIs.filter((_, idx) => idx !== i);
  }

  function validateForm(): boolean {
    const errs: Record<string, string> = {};
    if (!formName.trim()) errs.name = 'Name is required';
    else if (formName.trim().length > 50) errs.name = 'Name must be 50 characters or fewer';
    if (formLaunchURL && !formLaunchURL.startsWith('https://')) errs.launchURL = 'Must start with https://';
    const validURIs = formRedirectURIs.filter(u => u.trim());
    if (validURIs.length === 0) errs.redirectURIs = 'At least one redirect URI is required';
    for (const u of validURIs) {
      if (!u.startsWith('https://') && !u.startsWith('http://localhost') && !u.startsWith('http://127.0.0.1')) {
        errs.redirectURIs = 'URIs must be https:// or http://localhost/127.0.0.1';
        break;
      }
    }
    formErrors = errs;
    return Object.keys(errs).length === 0;
  }

  async function handleSubmit() {
    if (!validateForm()) return;
    submitting = true;
    try {
      const req: CreateAppRequest = {
        name: formName.trim(),
        description: formDescription.trim() || undefined,
        launch_url: formLaunchURL.trim() || undefined,
        redirect_uris: formRedirectURIs.filter(u => u.trim()),
        logout_uris: formLogoutURIs.filter(u => u.trim()),
        is_public: formIsPublic,
        pkce_required: formPkceRequired,
        verified_only: formVerifiedOnly,
      };
      const created = await createApp(req);
      // Upload logo if one was selected (non-fatal if it fails).
      if (formLogoFile) {
        try {
          await uploadAppLogo(created.id, formLogoFile);
        } catch (logoErr) {
          toast.error('App created, but logo upload failed — you can upload it from the app settings.');
        }
      }
      apps = [...apps, created];
      toast.success('Application registered!');
      if (created.client_secret) {
        // Navigate to detail page, passing the secret via history state (never in URL).
        goto(`/apps/${created.id}`, { state: { secret: created.client_secret } });
      } else {
        resetForm();
        showForm = false;
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      submitting = false;
    }
  }

  function resetForm() {
    formName = '';
    formDescription = '';
    formLaunchURL = '';
    formRedirectURIs = [''];
    formLogoutURIs = [];
    formIsPublic = false;
    formPkceRequired = false;
    formVerifiedOnly = false;
    formLogoFile = null;
    formErrors = {};
  }

  function handleLogoSelection(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0] ?? null;
    if (!file) {
      formLogoFile = null;
      return;
    }
    if (!allowedLogoTypes.has(file.type)) {
      formLogoFile = null;
      toast.error('Logo must be a PNG, JPEG, or WebP image');
      return;
    }
    formLogoFile = file;
  }
</script>

<div class="mx-auto w-full max-w-3xl px-6 py-16">
  {#if !data.status?.verified}
    <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-8 text-center">
      <ShieldCheck class="mx-auto mb-4 h-12 w-12 text-[#00d4ff]/40" />
      <h2 class="mb-2 text-xl font-bold text-[#e2e8f0]">Verification Required</h2>
      <p class="text-sm text-[#e2e8f0]/50">
        Only verified RSI citizens can register applications.
        <a href="/verify" class="text-[#00d4ff] hover:underline">Verify your identity →</a>
      </p>
    </div>
  {:else}
    <!-- Header -->
    <div class="mb-8 flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold text-[#00d4ff]">My Applications</h1>
        <p class="mt-1 text-sm text-[#e2e8f0]/50">OIDC clients registered via SCID ({apps.length}/5)</p>
        <a href="/docs/integration" class="mt-1 text-xs text-[#00d4ff]/50 hover:text-[#00d4ff] transition-colors">Integration docs →</a>
      </div>
      {#if !showForm && apps.length < 5}
        <button
          onclick={() => { resetForm(); showForm = true; }}
          class="flex items-center gap-2 rounded-lg border border-[#00d4ff] bg-[#00d4ff]/10 px-4 py-2 text-sm font-medium text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/20"
        >
          <Plus class="h-4 w-4" />
          Register New App
        </button>
      {/if}
    </div>

    <!-- New app form -->
    {#if showForm}
      <div class="mb-8 rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6">
        <h2 class="mb-6 text-lg font-semibold text-[#e2e8f0]">Register New Application</h2>

        <div class="space-y-5">
          <!-- Name -->
          <div>
            <label for="app-name" class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">
              Application Name <span class="text-red-400">*</span>
            </label>
            <input
              id="app-name"
              type="text"
              bind:value={formName}
              maxlength="50"
              placeholder="My Star Citizen App"
              class="w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30"
            />
            {#if formErrors.name}<p class="mt-1 text-xs text-red-400">{formErrors.name}</p>{/if}
          </div>

          <!-- Launch URL -->
          <div>
            <label for="app-launch-url" class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">Launch URL</label>
            <input
              id="app-launch-url"
              type="url"
              bind:value={formLaunchURL}
              placeholder="https://example.com"
              class="w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30"
            />
            {#if formErrors.launchURL}<p class="mt-1 text-xs text-red-400">{formErrors.launchURL}</p>{/if}
          </div>

          <!-- Description -->
          <div>
            <label for="app-description" class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">Description</label>
            <textarea
              id="app-description"
              bind:value={formDescription}
              maxlength="200"
              rows="2"
              placeholder="A short description shown in the app directory"
              class="w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30 resize-none"
            ></textarea>
            <p class="mt-1 text-xs text-[#e2e8f0]/30">{formDescription.length}/200</p>
          </div>
          <div>
            <p class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">
              Redirect URIs <span class="text-red-400">*</span>
            </p>
            <div class="space-y-2">
              {#each formRedirectURIs as uri, i}
                <div class="flex gap-2">
                  <input
                    type="text"
                    value={uri}
                    oninput={(e) => { formRedirectURIs[i] = (e.target as HTMLInputElement).value; }}
                    placeholder="https://example.com/callback"
                    class="flex-1 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30"
                  />
                  {#if formRedirectURIs.length > 1}
                    <button
                      onclick={() => removeRedirectURI(i)}
                      class="rounded-lg border border-[#1e3a5f] px-3 py-2 text-[#e2e8f0]/40 transition-colors hover:text-red-400"
                      type="button"
                    >
                      <Trash2 class="h-4 w-4" />
                    </button>
                  {/if}
                </div>
              {/each}
              {#if formRedirectURIs.length < 10}
                <button
                  onclick={addRedirectURI}
                  type="button"
                  class="flex items-center gap-1.5 text-xs text-[#00d4ff]/60 transition-colors hover:text-[#00d4ff]"
                >
                  <Plus class="h-3 w-3" /> Add URI
                </button>
              {/if}
            </div>
            {#if formErrors.redirectURIs}<p class="mt-1 text-xs text-red-400">{formErrors.redirectURIs}</p>{/if}
          </div>

          <!-- Logout URIs -->
          <div>
            <p class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">Logout Redirect URIs</p>
            <div class="space-y-2">
              {#each formLogoutURIs as uri, i}
                <div class="flex gap-2">
                  <input
                    type="text"
                    value={uri}
                    oninput={(e) => { formLogoutURIs[i] = (e.target as HTMLInputElement).value; }}
                    placeholder="https://example.com/logout"
                    class="flex-1 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30"
                  />
                  <button
                    onclick={() => removeLogoutURI(i)}
                    class="rounded-lg border border-[#1e3a5f] px-3 py-2 text-[#e2e8f0]/40 transition-colors hover:text-red-400"
                    type="button"
                  >
                    <Trash2 class="h-4 w-4" />
                  </button>
                </div>
              {/each}
              {#if formLogoutURIs.length < 10}
                <button
                  onclick={addLogoutURI}
                  type="button"
                  class="flex items-center gap-1.5 text-xs text-[#00d4ff]/60 transition-colors hover:text-[#00d4ff]"
                >
                  <Plus class="h-3 w-3" /> Add URI
                </button>
              {/if}
            </div>
          </div>

          <!-- Toggles -->
          <div class="grid gap-4 sm:grid-cols-3">
            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-3">
              <input type="checkbox" bind:checked={formIsPublic} class="mt-0.5 accent-[#00d4ff]" />
              <div>
                <p class="text-sm font-medium text-[#e2e8f0]">Public Client</p>
                <p class="mt-0.5 text-xs text-[#e2e8f0]/40">No client secret (SPA / mobile)</p>
              </div>
            </label>

            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-3">
              <input type="checkbox" bind:checked={formPkceRequired} disabled={formIsPublic} class="mt-0.5 accent-[#00d4ff] disabled:opacity-50" />
              <div>
                <p class="text-sm font-medium text-[#e2e8f0]">Require PKCE</p>
                <p class="mt-0.5 text-xs text-[#e2e8f0]/40">Recommended for all clients</p>
              </div>
            </label>

            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-3">
              <input type="checkbox" bind:checked={formVerifiedOnly} class="mt-0.5 accent-[#00d4ff]" />
              <div>
                <p class="text-sm font-medium text-[#e2e8f0]">Verified Only</p>
                <p class="mt-0.5 text-xs text-[#e2e8f0]/40">Restrict to verified RSI citizens</p>
              </div>
            </label>
          </div>
        </div>

        <!-- Logo -->
        <div class="mt-4">
          <p class="mb-1 text-sm font-medium text-[#e2e8f0]/70">Logo (optional)</p>
          <label class="flex cursor-pointer items-center gap-3 rounded-lg border border-dashed border-[#1e3a5f] bg-[#0a0e1a] p-3 hover:border-[#00d4ff]/50">
            <input
              type="file"
              accept="image/png,image/jpeg,image/webp"
              onchange={handleLogoSelection}
              class="sr-only"
            />
            <span class="text-sm text-[#e2e8f0]/50">
              {formLogoFile ? formLogoFile.name : 'Click to choose an image…'}
            </span>
          </label>
        </div>

        <div class="mt-6 flex items-center gap-3">
          <button
            onclick={handleSubmit}
            disabled={submitting}
            class="rounded-lg border border-[#00d4ff] bg-[#00d4ff]/10 px-5 py-2 text-sm font-medium text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/20 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitting ? 'Registering…' : 'Register Application'}
          </button>
          <button
            onclick={() => { resetForm(); showForm = false; }}
            disabled={submitting}
            class="rounded-lg border border-[#1e3a5f] px-4 py-2 text-sm text-[#e2e8f0]/50 transition-colors hover:text-[#e2e8f0]/80 disabled:opacity-50"
          >
            Cancel
          </button>
        </div>
      </div>
    {/if}

    <!-- App list -->
    {#if loading}
      <div class="py-16 text-center text-sm text-[#e2e8f0]/40">Loading…</div>
    {:else if apps.length === 0 && !showForm}
      <div class="rounded-2xl border border-dashed border-[#1e3a5f] p-12 text-center">
        <AppWindow class="mx-auto mb-4 h-12 w-12 text-[#00d4ff]/20" />
        <h2 class="mb-1 text-lg font-semibold text-[#e2e8f0]/60">No Applications Yet</h2>
        <p class="mb-6 text-sm text-[#e2e8f0]/30">Register your first OIDC client to let players log in via SCID.</p>
        <button
          onclick={() => { resetForm(); showForm = true; }}
          class="rounded-lg border border-[#00d4ff] bg-[#00d4ff]/10 px-5 py-2 text-sm font-medium text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/20"
        >
          Register New App
        </button>
      </div>
    {:else}
      <div class="space-y-3">
        {#each apps as app (app.id)}
          <a
            href="/apps/{app.id}"
            class="group flex items-center gap-4 rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-5 transition-colors hover:border-[#00d4ff]/30"
          >
            <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-[#1e3a5f] bg-[#0a0e1a]">
              {#if app.has_logo}
                <img src="/api/oidc/clients/{app.id}/logo" alt="{app.name} logo" class="h-8 w-8 rounded object-contain" />
              {:else}
                <AppWindow class="h-5 w-5 text-[#00d4ff]/40" />
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="font-medium text-[#e2e8f0]">{app.name}</span>
                {#if app.verified_only}
                  <span class="rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-400">
                    Verified Only
                  </span>
                {/if}
                {#if app.is_public}
                  <span class="rounded-full border border-[#00d4ff]/30 bg-[#00d4ff]/10 px-2 py-0.5 text-xs text-[#00d4ff]/70">
                    Public
                  </span>
                {/if}
              </div>
              <p class="mt-0.5 truncate font-mono text-xs text-[#e2e8f0]/30">{app.id}</p>
            </div>
            <div class="flex-shrink-0 text-xs text-[#e2e8f0]/30">
              {app.created_at.slice(0, 10)}
            </div>
            <ChevronRight class="h-4 w-4 flex-shrink-0 text-[#e2e8f0]/20 transition-colors group-hover:text-[#00d4ff]/50" />
          </a>
        {/each}
      </div>
    {/if}
  {/if}
</div>
