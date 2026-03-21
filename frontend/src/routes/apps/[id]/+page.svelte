<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import {
    AppWindow, Copy, Check, RotateCcw, Trash2, Pencil, Plus, ShieldCheck, Upload, X
  } from '@lucide/svelte';
  import type { PageData } from './$types';
  import type { AppRegistration, CreateAppRequest } from '$lib/utils/api';
  import { getApp, updateApp, deleteApp, rotateSecret, uploadAppLogo } from '$lib/utils/api';
  import CopyButton from '$lib/components/CopyButton.svelte';

  let { data }: { data: PageData } = $props();

  const appId = data.id as string;

  let app = $state<AppRegistration | null>(null);
  let loading = $state(true);
  let editing = $state(false);
  let showDeleteConfirm = $state(false);
  let showRotateConfirm = $state(false);
  let deleting = $state(false);
  let rotating = $state(false);
  let savingEdit = $state(false);
  let uploadingLogo = $state(false);

  // Once-shown secret (from URL after create, or after rotation)
  let shownSecret = $state('');
  let secretCopied = $state(false);

  // Edit form state
  let editName = $state('');
  let editLaunchURL = $state('');
  let editRedirectURIs = $state(['']);
  let editLogoutURIs = $state<string[]>([]);
  let editIsPublic = $state(false);
  let editPkceRequired = $state(false);
  let editVerifiedOnly = $state(false);
  let editListed = $state(false);
  let editErrors = $state<Record<string, string>>({});

  onMount(async () => {
    // Check for secret in URL params (passed after create)
    const urlSecret = $page.url.searchParams.get('secret');
    if (urlSecret) {
      shownSecret = urlSecret;
      // Remove secret from URL without navigation
      const url = new URL($page.url);
      url.searchParams.delete('secret');
      url.searchParams.delete('new');
      history.replaceState({}, '', url);
    }

    try {
      app = await getApp(appId);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load application');
    } finally {
      loading = false;
    }
  });

  $effect(() => {
    if (editIsPublic) editPkceRequired = true;
  });

  function startEdit() {
    if (!app) return;
    editName = app.name;
    editLaunchURL = app.launch_url ?? '';
    editRedirectURIs = app.redirect_uris.length ? [...app.redirect_uris] : [''];
    editLogoutURIs = [...app.logout_uris];
    editIsPublic = app.is_public;
    editPkceRequired = app.pkce_required;
    editVerifiedOnly = app.verified_only;
    editListed = app.listed ?? false;
    editErrors = {};
    editing = true;
  }

  function validateEditForm(): boolean {
    const errs: Record<string, string> = {};
    if (!editName.trim()) errs.name = 'Name is required';
    else if (editName.trim().length > 50) errs.name = 'Name must be 50 characters or fewer';
    if (editLaunchURL && !editLaunchURL.startsWith('https://')) errs.launchURL = 'Must start with https://';
    const validURIs = editRedirectURIs.filter(u => u.trim());
    if (validURIs.length === 0) errs.redirectURIs = 'At least one redirect URI is required';
    for (const u of validURIs) {
      if (!u.startsWith('https://') && !u.startsWith('http://localhost') && !u.startsWith('http://127.0.0.1')) {
        errs.redirectURIs = 'URIs must be https:// or http://localhost/127.0.0.1';
        break;
      }
    }
    editErrors = errs;
    return Object.keys(errs).length === 0;
  }

  async function saveEdit() {
    if (!validateEditForm()) return;
    savingEdit = true;
    try {
      const req: CreateAppRequest = {
        name: editName.trim(),
        launch_url: editLaunchURL.trim() || undefined,
        redirect_uris: editRedirectURIs.filter(u => u.trim()),
        logout_uris: editLogoutURIs.filter(u => u.trim()),
        is_public: editIsPublic,
        pkce_required: editPkceRequired,
        verified_only: editVerifiedOnly,
        listed: editListed,
      };
      app = await updateApp(appId, req);
      editing = false;
      toast.success('Application updated');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Update failed');
    } finally {
      savingEdit = false;
    }
  }

  async function handleDelete() {
    deleting = true;
    try {
      await deleteApp(appId);
      toast.success('Application deleted');
      goto('/apps');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Delete failed');
      deleting = false;
    }
  }

  async function handleRotate() {
    rotating = true;
    try {
      const result = await rotateSecret(appId);
      shownSecret = result.client_secret;
      secretCopied = false;
      showRotateConfirm = false;
      toast.success('Secret rotated — copy it now, it will not be shown again');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Rotation failed');
    } finally {
      rotating = false;
    }
  }

  async function handleLogoUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (file.size > 1 << 20) {
      toast.error('Logo must be 1 MB or smaller');
      return;
    }
    uploadingLogo = true;
    try {
      await uploadAppLogo(appId, file);
      // Refetch to get updated hasLogo
      app = await getApp(appId);
      toast.success('Logo uploaded');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      uploadingLogo = false;
      input.value = '';
    }
  }

  function addEditRedirectURI() {
    if (editRedirectURIs.length < 10) editRedirectURIs = [...editRedirectURIs, ''];
  }
  function removeEditRedirectURI(i: number) {
    editRedirectURIs = editRedirectURIs.filter((_, idx) => idx !== i);
  }
  function addEditLogoutURI() {
    if (editLogoutURIs.length < 10) editLogoutURIs = [...editLogoutURIs, ''];
  }
  function removeEditLogoutURI(i: number) {
    editLogoutURIs = editLogoutURIs.filter((_, idx) => idx !== i);
  }
</script>

<div class="mx-auto w-full max-w-2xl px-6 py-16">
  <div class="mb-6">
    <a href="/apps" class="text-sm text-[#00d4ff]/60 transition-colors hover:text-[#00d4ff]">← My Applications</a>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-[#e2e8f0]/40">Loading…</div>
  {:else if !app}
    <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-8 text-center">
      <p class="text-[#e2e8f0]/50">Application not found.</p>
      <a href="/apps" class="mt-4 inline-block text-sm text-[#00d4ff] hover:underline">← Back to Apps</a>
    </div>
  {:else}
    <!-- Header card -->
    <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6 shadow-[0_0_40px_rgba(0,212,255,0.04)]">
      <div class="mb-5 flex items-start gap-4">
        <!-- Logo -->
        <div class="relative flex-shrink-0">
          <div class="flex h-16 w-16 items-center justify-center overflow-hidden rounded-xl border border-[#1e3a5f] bg-[#0a0e1a]">
            {#if app.has_logo}
              <img src="/api/oidc/clients/{appId}/logo" alt="{app.name} logo" class="h-full w-full object-contain" />
            {:else}
              <AppWindow class="h-8 w-8 text-[#00d4ff]/30" />
            {/if}
          </div>
          <label
            class="absolute -bottom-1 -right-1 flex h-6 w-6 cursor-pointer items-center justify-center rounded-full border border-[#1e3a5f] bg-[#0d1526] text-[#e2e8f0]/40 transition-colors hover:text-[#00d4ff]"
            title="Upload logo"
          >
            <input type="file" accept="image/png,image/jpeg,image/webp,image/svg+xml" onchange={handleLogoUpload} class="sr-only" />
            {#if uploadingLogo}
              <span class="h-3 w-3 animate-spin rounded-full border border-[#00d4ff] border-t-transparent"></span>
            {:else}
              <Upload class="h-3 w-3" />
            {/if}
          </label>
        </div>

        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <h1 class="text-xl font-bold text-[#e2e8f0]">{app.name}</h1>
            {#if app.status === 'pending'}
              <span class="rounded-full border border-yellow-500/30 bg-yellow-500/10 px-2 py-0.5 text-xs text-yellow-400">
                Pending Approval
              </span>
            {:else if app.status === 'rejected'}
              <span class="rounded-full border border-red-500/30 bg-red-500/10 px-2 py-0.5 text-xs text-red-400">
                Rejected
              </span>
            {/if}
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
          <p class="mt-1 text-xs text-[#e2e8f0]/30">Created {app.created_at.slice(0, 10)}</p>
          {#if app.status === 'pending'}
            <p class="mt-1 text-xs text-yellow-400/70">Your application is awaiting admin review. It will not be usable until approved.</p>
          {:else if app.status === 'rejected' && app.rejection_reason}
            <p class="mt-1 text-xs text-red-400/70">Reason: {app.rejection_reason}</p>
          {/if}
        </div>

        {#if !editing}
          <button
            onclick={startEdit}
            class="flex items-center gap-1.5 rounded-lg border border-[#1e3a5f] px-3 py-1.5 text-xs text-[#e2e8f0]/50 transition-colors hover:text-[#e2e8f0]/80"
          >
            <Pencil class="h-3 w-3" /> Edit
          </button>
        {/if}
      </div>

      <!-- Client ID -->
      <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-4 py-3">
        <p class="mb-1 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Client ID (OIDC client_id)</p>
        <div class="flex items-center gap-2">
          <code class="flex-1 truncate font-mono text-sm text-[#00d4ff]/80">{app.id}</code>
          <CopyButton text={app.id} />
        </div>
      </div>

      <!-- Client Secret -->
      {#if !app.is_public}
        <div class="mt-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-4 py-3">
          <p class="mb-1 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Client Secret</p>
          {#if shownSecret}
            <div class="mb-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-400">
              ⚠ Copy this secret now — it will not be shown again after you leave this page.
            </div>
            <div class="flex items-center gap-2">
              <code class="flex-1 break-all font-mono text-sm text-emerald-400">{shownSecret}</code>
              <CopyButton text={shownSecret} />
            </div>
          {:else}
            <div class="flex items-center justify-between">
              <code class="text-sm text-[#e2e8f0]/20">●●●●●●●●●●●●●●●●●●●●</code>
              {#if !showRotateConfirm}
                <button
                  onclick={() => (showRotateConfirm = true)}
                  class="flex items-center gap-1.5 text-xs text-[#e2e8f0]/40 transition-colors hover:text-[#00d4ff]"
                >
                  <RotateCcw class="h-3 w-3" /> Rotate
                </button>
              {/if}
            </div>
          {/if}

          {#if showRotateConfirm}
            <div class="mt-3 flex items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
              <p class="flex-1 text-xs text-amber-400/90">Rotating will invalidate the current secret. Continue?</p>
              <button
                onclick={handleRotate}
                disabled={rotating}
                class="rounded bg-amber-600 px-3 py-1 text-xs font-medium text-white hover:bg-amber-700 disabled:opacity-50"
              >
                {rotating ? 'Rotating…' : 'Rotate'}
              </button>
              <button
                onclick={() => (showRotateConfirm = false)}
                disabled={rotating}
                class="text-xs text-[#e2e8f0]/40 hover:text-[#e2e8f0]/70 disabled:opacity-50"
              >
                Cancel
              </button>
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Edit form -->
    {#if editing}
      <div class="mt-6 rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6">
        <h2 class="mb-5 text-base font-semibold text-[#e2e8f0]">Edit Application</h2>
        <div class="space-y-4">
          <div>
            <label class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">Name</label>
            <input type="text" bind:value={editName} maxlength="50"
              class="w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30" />
            {#if editErrors.name}<p class="mt-1 text-xs text-red-400">{editErrors.name}</p>{/if}
          </div>

          <div>
            <label class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">Launch URL</label>
            <input type="url" bind:value={editLaunchURL} placeholder="https://example.com"
              class="w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30" />
            {#if editErrors.launchURL}<p class="mt-1 text-xs text-red-400">{editErrors.launchURL}</p>{/if}
          </div>

          <div>
            <label class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">Redirect URIs</label>
            <div class="space-y-2">
              {#each editRedirectURIs as uri, i}
                <div class="flex gap-2">
                  <input type="text" value={uri}
                    oninput={(e) => { editRedirectURIs[i] = (e.target as HTMLInputElement).value; }}
                    placeholder="https://example.com/callback"
                    class="flex-1 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30" />
                  {#if editRedirectURIs.length > 1}
                    <button onclick={() => removeEditRedirectURI(i)} type="button"
                      class="rounded-lg border border-[#1e3a5f] px-3 py-2 text-[#e2e8f0]/40 hover:text-red-400">
                      <Trash2 class="h-4 w-4" />
                    </button>
                  {/if}
                </div>
              {/each}
              {#if editRedirectURIs.length < 10}
                <button onclick={addEditRedirectURI} type="button"
                  class="flex items-center gap-1.5 text-xs text-[#00d4ff]/60 hover:text-[#00d4ff]">
                  <Plus class="h-3 w-3" /> Add URI
                </button>
              {/if}
            </div>
            {#if editErrors.redirectURIs}<p class="mt-1 text-xs text-red-400">{editErrors.redirectURIs}</p>{/if}
          </div>

          <div>
            <label class="mb-1.5 block text-sm font-medium text-[#e2e8f0]/70">Logout URIs</label>
            <div class="space-y-2">
              {#each editLogoutURIs as uri, i}
                <div class="flex gap-2">
                  <input type="text" value={uri}
                    oninput={(e) => { editLogoutURIs[i] = (e.target as HTMLInputElement).value; }}
                    placeholder="https://example.com/logout"
                    class="flex-1 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none focus:ring-1 focus:ring-[#00d4ff]/30" />
                  <button onclick={() => removeEditLogoutURI(i)} type="button"
                    class="rounded-lg border border-[#1e3a5f] px-3 py-2 text-[#e2e8f0]/40 hover:text-red-400">
                    <Trash2 class="h-4 w-4" />
                  </button>
                </div>
              {/each}
              {#if editLogoutURIs.length < 10}
                <button onclick={addEditLogoutURI} type="button"
                  class="flex items-center gap-1.5 text-xs text-[#00d4ff]/60 hover:text-[#00d4ff]">
                  <Plus class="h-3 w-3" /> Add URI
                </button>
              {/if}
            </div>
          </div>

          <div class="grid gap-4 sm:grid-cols-3">
            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-3">
              <input type="checkbox" bind:checked={editIsPublic} class="mt-0.5 accent-[#00d4ff]" />
              <div>
                <p class="text-sm font-medium text-[#e2e8f0]">Public Client</p>
                <p class="mt-0.5 text-xs text-[#e2e8f0]/40">No client secret</p>
              </div>
            </label>
            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-3">
              <input type="checkbox" bind:checked={editPkceRequired} disabled={editIsPublic} class="mt-0.5 accent-[#00d4ff] disabled:opacity-50" />
              <div>
                <p class="text-sm font-medium text-[#e2e8f0]">Require PKCE</p>
                <p class="mt-0.5 text-xs text-[#e2e8f0]/40">Recommended</p>
              </div>
            </label>
            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-3">
              <input type="checkbox" bind:checked={editVerifiedOnly} class="mt-0.5 accent-[#00d4ff]" />
              <div>
                <p class="text-sm font-medium text-[#e2e8f0]">Verified Only</p>
                <p class="mt-0.5 text-xs text-[#e2e8f0]/40">Restrict access</p>
              </div>
            </label>
          </div>

          {#if app?.status === 'approved'}
            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-3">
              <input
                type="checkbox"
                bind:checked={editListed}
                disabled={!editLaunchURL.trim()}
                class="mt-0.5 accent-[#00d4ff] disabled:cursor-not-allowed disabled:opacity-40"
              />
              <div>
                <p class="text-sm font-medium text-[#e2e8f0]">List in App Directory</p>
                <p class="mt-0.5 text-xs text-[#e2e8f0]/40">
                  Show this app on the public Discover page — requires a Launch URL.
                </p>
              </div>
            </label>
          {/if}
        </div>

        <div class="mt-5 flex items-center gap-3">
          <button onclick={saveEdit} disabled={savingEdit}
            class="rounded-lg border border-[#00d4ff] bg-[#00d4ff]/10 px-5 py-2 text-sm font-medium text-[#00d4ff] transition-colors hover:bg-[#00d4ff]/20 disabled:opacity-50">
            {savingEdit ? 'Saving…' : 'Save Changes'}
          </button>
          <button onclick={() => { editing = false; editErrors = {}; }} disabled={savingEdit}
            class="rounded-lg border border-[#1e3a5f] px-4 py-2 text-sm text-[#e2e8f0]/50 hover:text-[#e2e8f0]/80 disabled:opacity-50">
            Cancel
          </button>
        </div>
      </div>
    {:else}
      <!-- Detail view -->
      <div class="mt-6 space-y-4">
        {#if app.launch_url}
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-4 py-3">
            <p class="mb-1 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Launch URL</p>
            <a href={app.launch_url} target="_blank" rel="noopener" class="text-sm text-[#00d4ff]/70 hover:underline">{app.launch_url}</a>
          </div>
        {/if}

        <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-4 py-3">
          <p class="mb-2 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Redirect URIs</p>
          <ul class="space-y-1">
            {#each app.redirect_uris as uri}
              <li class="font-mono text-sm text-[#e2e8f0]/70">{uri}</li>
            {/each}
          </ul>
        </div>

        {#if app.logout_uris.length > 0}
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-4 py-3">
            <p class="mb-2 text-xs font-medium uppercase tracking-wider text-[#e2e8f0]/40">Logout URIs</p>
            <ul class="space-y-1">
              {#each app.logout_uris as uri}
                <li class="font-mono text-sm text-[#e2e8f0]/70">{uri}</li>
              {/each}
            </ul>
          </div>
        {/if}

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-3 py-2 text-center">
            <p class="text-xs text-[#e2e8f0]/40">Public</p>
            <p class="mt-0.5 text-sm font-medium text-[#e2e8f0]">{app.is_public ? 'Yes' : 'No'}</p>
          </div>
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-3 py-2 text-center">
            <p class="text-xs text-[#e2e8f0]/40">PKCE</p>
            <p class="mt-0.5 text-sm font-medium text-[#e2e8f0]">{app.pkce_required ? 'Required' : 'Optional'}</p>
          </div>
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-3 py-2 text-center">
            <p class="text-xs text-[#e2e8f0]/40">Verified Only</p>
            <p class="mt-0.5 text-sm font-medium {app.verified_only ? 'text-emerald-400' : 'text-[#e2e8f0]'}">{app.verified_only ? 'Yes' : 'No'}</p>
          </div>
          <div class="rounded-lg border border-[#1e3a5f] bg-[#0d1526] px-3 py-2 text-center">
            <p class="text-xs text-[#e2e8f0]/40">Logo</p>
            <p class="mt-0.5 text-sm font-medium text-[#e2e8f0]">{app.has_logo ? 'Set' : 'None'}</p>
          </div>
        </div>
      </div>
    {/if}

    <!-- Delete -->
    <div class="mt-8">
      {#if showDeleteConfirm}
        <div class="flex items-center gap-3 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3">
          <p class="flex-1 text-sm text-red-400/90"><strong>Warning:</strong> This will permanently delete this application and revoke all access tokens. Cannot be undone.</p>
          <button onclick={handleDelete} disabled={deleting}
            class="rounded-lg bg-red-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">
            {deleting ? 'Deleting…' : 'Yes, delete'}
          </button>
          <button onclick={() => (showDeleteConfirm = false)} disabled={deleting}
            class="rounded-lg border border-[#1e3a5f] px-4 py-1.5 text-sm text-[#e2e8f0]/60 hover:text-[#e2e8f0]/90 disabled:opacity-50">
            Cancel
          </button>
        </div>
      {:else}
        <button onclick={() => (showDeleteConfirm = true)}
          class="flex items-center gap-1.5 text-xs text-red-400/50 transition-colors hover:text-red-400/90">
          <Trash2 class="h-3 w-3" /> Delete Application
        </button>
      {/if}
    </div>
  {/if}
</div>
