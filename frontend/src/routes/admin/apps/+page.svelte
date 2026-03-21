<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { ShieldCheck, CheckCircle, XCircle, Clock, AlertCircle } from '@lucide/svelte';
  import { goto } from '$app/navigation';
  import type { PageData } from './$types';
  import type { AppRegistration } from '$lib/utils/api';
  import { adminListApps, adminApproveApp, adminRejectApp } from '$lib/utils/api';

  let { data }: { data: PageData } = $props();

  let apps = $state<AppRegistration[]>([]);
  let loading = $state(true);
  let filterStatus = $state('pending');
  let rejectingId = $state('');
  let rejectReason = $state('');
  let showRejectModal = $state(false);
  let actionLoading = $state(false);

  onMount(async () => {
    if (!data.status?.admin) {
      goto('/');
      return;
    }
    await loadApps();
  });

  async function loadApps() {
    loading = true;
    try {
      apps = await adminListApps(filterStatus || undefined);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load apps');
    } finally {
      loading = false;
    }
  }

  async function approve(id: string) {
    actionLoading = true;
    try {
      const updated = await adminApproveApp(id);
      apps = apps.map(a => a.id === id ? updated : a);
      toast.success('Application approved');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Approval failed');
    } finally {
      actionLoading = false;
    }
  }

  function openRejectModal(id: string) {
    rejectingId = id;
    rejectReason = '';
    showRejectModal = true;
  }

  async function confirmReject() {
    actionLoading = true;
    try {
      const updated = await adminRejectApp(rejectingId, rejectReason.trim() || undefined);
      apps = apps.map(a => a.id === rejectingId ? updated : a);
      toast.success('Application rejected');
      showRejectModal = false;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Rejection failed');
    } finally {
      actionLoading = false;
    }
  }

  function statusBadge(status: string) {
    switch (status) {
      case 'approved': return { label: 'Approved', cls: 'text-green-400 bg-green-400/10 border-green-400/20' };
      case 'rejected': return { label: 'Rejected', cls: 'text-red-400 bg-red-400/10 border-red-400/20' };
      default:         return { label: 'Pending', cls: 'text-yellow-400 bg-yellow-400/10 border-yellow-400/20' };
    }
  }
</script>

<div class="mx-auto w-full max-w-4xl px-6 py-16">
  <div class="mb-8 flex items-start justify-between">
    <div>
      <h1 class="text-3xl font-bold text-[#ffd700]">Admin — App Registrations</h1>
      <p class="mt-1 text-sm text-[#e2e8f0]/50">Review and approve OIDC client registrations</p>
    </div>
    <!-- Filter tabs -->
    <div class="flex rounded-lg border border-[#1e3a5f] overflow-hidden text-xs font-medium">
      {#each [['pending', 'Pending'], ['approved', 'Approved'], ['rejected', 'Rejected'], ['', 'All']] as [val, label]}
        <button
          onclick={() => { filterStatus = val; loadApps(); }}
          class="px-3 py-2 transition-colors {filterStatus === val ? 'bg-[#1e3a5f] text-[#00d4ff]' : 'text-[#e2e8f0]/50 hover:text-[#e2e8f0]'}"
        >
          {label}
        </button>
      {/each}
    </div>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-[#e2e8f0]/40">Loading…</div>
  {:else if !data.status?.admin}
    <div class="rounded-2xl border border-red-900/40 bg-red-950/20 p-8 text-center">
      <AlertCircle class="mx-auto mb-3 h-10 w-10 text-red-400/60" />
      <p class="text-[#e2e8f0]/60">You do not have admin access.</p>
    </div>
  {:else if apps.length === 0}
    <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-12 text-center">
      <CheckCircle class="mx-auto mb-4 h-12 w-12 text-green-400/40" />
      <p class="text-[#e2e8f0]/50">No applications match the current filter.</p>
    </div>
  {:else}
    <div class="space-y-4">
      {#each apps as app (app.id)}
        {@const badge = statusBadge(app.status)}
        <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-5">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-semibold text-[#e2e8f0]">{app.name}</span>
                <span class="rounded border px-1.5 py-0.5 text-[10px] font-medium {badge.cls}">{badge.label}</span>
                {#if app.verified_only}
                  <span class="rounded border border-[#00d4ff]/20 bg-[#00d4ff]/10 px-1.5 py-0.5 text-[10px] font-medium text-[#00d4ff]">Verified Only</span>
                {/if}
              </div>
              <div class="mt-1 flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-[#e2e8f0]/50">
                <span>Owner: <span class="text-[#e2e8f0]/70">{app.owner_username ?? app.id.slice(0, 8)}</span></span>
                <span>Client ID: <code class="text-[#e2e8f0]/70">{app.id.slice(0, 16)}…</code></span>
                <span>Submitted: {new Date(app.created_at).toLocaleDateString()}</span>
              </div>
              {#if app.launch_url}
                <p class="mt-0.5 truncate text-xs text-[#00d4ff]/60">{app.launch_url}</p>
              {/if}
              {#if app.rejection_reason}
                <p class="mt-1 text-xs text-red-400/80">Reason: {app.rejection_reason}</p>
              {/if}
              <!-- Redirect URIs summary -->
              <div class="mt-2 flex flex-wrap gap-1">
                {#each app.redirect_uris.slice(0, 3) as uri}
                  <code class="rounded bg-[#0a0e1a] px-1.5 py-0.5 text-[10px] text-[#e2e8f0]/50">{uri}</code>
                {/each}
                {#if app.redirect_uris.length > 3}
                  <span class="text-xs text-[#e2e8f0]/30">+{app.redirect_uris.length - 3} more</span>
                {/if}
              </div>
            </div>
            <!-- Actions -->
            {#if app.status !== 'approved'}
              <button
                onclick={() => approve(app.id)}
                disabled={actionLoading}
                class="flex shrink-0 items-center gap-1.5 rounded-lg border border-green-500/30 bg-green-500/10 px-3 py-1.5 text-xs font-medium text-green-400 transition-colors hover:bg-green-500/20 disabled:opacity-50"
              >
                <CheckCircle class="h-3.5 w-3.5" />
                Approve
              </button>
            {/if}
            {#if app.status !== 'rejected'}
              <button
                onclick={() => openRejectModal(app.id)}
                disabled={actionLoading}
                class="flex shrink-0 items-center gap-1.5 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-1.5 text-xs font-medium text-red-400 transition-colors hover:bg-red-500/20 disabled:opacity-50"
              >
                <XCircle class="h-3.5 w-3.5" />
                Reject
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Reject modal -->
{#if showRejectModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    onclick={(e) => { if (e.target === e.currentTarget) showRejectModal = false; }}
    onkeydown={(e) => { if (e.key === 'Escape') showRejectModal = false; }}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="w-full max-w-md rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6 shadow-xl">
      <h2 class="mb-4 text-base font-semibold text-[#e2e8f0]">Reject Application</h2>
      <label for="reject-reason" class="mb-1.5 block text-sm text-[#e2e8f0]/70">Reason (optional)</label>
      <textarea
        id="reject-reason"
        bind:value={rejectReason}
        rows="3"
        placeholder="Explain why this application was rejected…"
        class="w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 focus:border-[#00d4ff]/50 focus:outline-none"
      ></textarea>
      <div class="mt-4 flex justify-end gap-3">
        <button
          onclick={() => showRejectModal = false}
          class="rounded-lg border border-[#1e3a5f] px-4 py-2 text-sm text-[#e2e8f0]/60 transition-colors hover:text-[#e2e8f0]"
        >
          Cancel
        </button>
        <button
          onclick={confirmReject}
          disabled={actionLoading}
          class="rounded-lg bg-red-600/80 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-600 disabled:opacity-50"
        >
          Confirm Reject
        </button>
      </div>
    </div>
  </div>
{/if}
