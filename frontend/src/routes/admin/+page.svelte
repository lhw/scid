<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import {
    ShieldCheck, CheckCircle, XCircle, AlertCircle, LoaderCircle,
    Users, Building2, Flag, ShieldX, Trash2, User, ImageOff,
  } from '@lucide/svelte';
  import type { PageData } from './$types';
  import type { AppRegistration, AdminUserEntry, AdminBlockedHandle, AdminOrgEntry, AdminReport } from '$lib/utils/api';
  import {
    adminListApps, adminApproveApp, adminRejectApp,
    adminListUsers, adminDeleteUser, adminListBlockedHandles, adminBlockHandle, adminUnblockHandle,
    adminListOrgs, adminBlockOrgLogo, adminUnblockOrgLogo,
    adminListReports, adminReviewReport, adminDismissReport,
  } from '$lib/utils/api';

  let { data }: { data: PageData } = $props();

  // ---- Tab state ----
  let activeTab = $derived($page.url.searchParams.get('tab') ?? 'apps');

  function setTab(tab: string) {
    goto(`?tab=${tab}`, { replaceState: false, noScroll: true });
  }

  const tabs = [
    { id: 'apps',    label: 'App Registrations' },
    { id: 'users',   label: 'Users & Handles' },
    { id: 'orgs',    label: 'Org Logos' },
    { id: 'reports', label: 'Reports' },
  ];

  // ---- Apps ----
  let apps = $state<AppRegistration[]>([]);
  let appsLoading = $state(false);
  let appsFilterStatus = $state('pending');
  let rejectingId = $state('');
  let rejectReason = $state('');
  let showRejectModal = $state(false);
  let appsActionLoading = $state(false);

  async function loadApps() {
    appsLoading = true;
    try {
      apps = await adminListApps(appsFilterStatus || undefined);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load apps');
    } finally {
      appsLoading = false;
    }
  }

  async function approveApp(id: string) {
    appsActionLoading = true;
    try {
      const updated = await adminApproveApp(id);
      apps = apps.map(a => a.id === id ? updated : a);
      toast.success('Application approved');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Approval failed');
    } finally {
      appsActionLoading = false;
    }
  }

  function openRejectModal(id: string) {
    rejectingId = id;
    rejectReason = '';
    showRejectModal = true;
  }

  async function confirmReject() {
    appsActionLoading = true;
    try {
      const updated = await adminRejectApp(rejectingId, rejectReason.trim() || undefined);
      apps = apps.map(a => a.id === rejectingId ? updated : a);
      toast.success('Application rejected');
      showRejectModal = false;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Rejection failed');
    } finally {
      appsActionLoading = false;
    }
  }

  function appStatusBadge(status: string) {
    switch (status) {
      case 'approved': return { label: 'Approved', cls: 'text-green-400 bg-green-400/10 border-green-400/20' };
      case 'rejected': return { label: 'Rejected', cls: 'text-red-400 bg-red-400/10 border-red-400/20' };
      default:         return { label: 'Pending',  cls: 'text-yellow-400 bg-yellow-400/10 border-yellow-400/20' };
    }
  }

  // ---- Users ----
  let users = $state<AdminUserEntry[]>([]);
  let blockedHandles = $state<AdminBlockedHandle[]>([]);
  let usersLoading = $state(false);
  let usersActionLoading = $state(false);
  let deleteTarget = $state<AdminUserEntry | null>(null);
  let deleteBlockHandle = $state(false);
  let deleteReason = $state('');
  let blockTarget = $state('');
  let blockReason = $state('');
  let showBlockModal = $state(false);

  async function loadUsers() {
    usersLoading = true;
    try {
      [users, blockedHandles] = await Promise.all([adminListUsers(), adminListBlockedHandles()]);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      usersLoading = false;
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    usersActionLoading = true;
    try {
      await adminDeleteUser(deleteTarget.user_id, deleteBlockHandle, deleteReason.trim() || undefined);
      toast.success(`User ${deleteTarget.handle} removed`);
      deleteTarget = null;
      await loadUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Delete failed');
    } finally {
      usersActionLoading = false;
    }
  }

  async function confirmBlock() {
    usersActionLoading = true;
    try {
      await adminBlockHandle(blockTarget.trim(), blockReason.trim() || undefined);
      toast.success(`Handle ${blockTarget.trim()} blocked`);
      showBlockModal = false;
      blockTarget = '';
      blockReason = '';
      blockedHandles = await adminListBlockedHandles();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Block failed');
    } finally {
      usersActionLoading = false;
    }
  }

  async function unblockHandle(handle: string) {
    usersActionLoading = true;
    try {
      await adminUnblockHandle(handle);
      toast.success(`Handle ${handle} unblocked`);
      blockedHandles = blockedHandles.filter(b => b.handle !== handle);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Unblock failed');
    } finally {
      usersActionLoading = false;
    }
  }

  // ---- Orgs ----
  let orgs = $state<AdminOrgEntry[]>([]);
  let orgsLoading = $state(false);
  let orgsActionLoading = $state('');

  async function loadOrgs() {
    orgsLoading = true;
    try {
      orgs = await adminListOrgs();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load orgs');
    } finally {
      orgsLoading = false;
    }
  }

  async function toggleOrgLogo(org: AdminOrgEntry) {
    orgsActionLoading = org.sid;
    try {
      if (org.logo_blocked) {
        await adminUnblockOrgLogo(org.sid);
        toast.success(`Logo unblocked for ${org.sid}`);
        orgs = orgs.map(o => o.sid === org.sid ? { ...o, logo_blocked: false } : o);
      } else {
        await adminBlockOrgLogo(org.sid);
        toast.success(`Logo blocked for ${org.sid}`);
        orgs = orgs.map(o => o.sid === org.sid ? { ...o, logo_blocked: true, has_logo: false } : o);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Action failed');
    } finally {
      orgsActionLoading = '';
    }
  }

  // ---- Reports ----
  let reports = $state<AdminReport[]>([]);
  let reportsLoading = $state(false);
  let reportsFilterStatus = $state('pending');
  let reportsActionLoading = $state('');

  async function loadReports() {
    reportsLoading = true;
    try {
      reports = await adminListReports(reportsFilterStatus || undefined);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load reports');
    } finally {
      reportsLoading = false;
    }
  }

  async function reviewReport(id: string) {
    reportsActionLoading = id;
    try {
      await adminReviewReport(id);
      toast.success('Marked as reviewed');
      reports = reports.map(r => r.id === id ? { ...r, status: 'reviewed' } : r);
      reports = reports.filter(r => reportsFilterStatus === '' || r.status === reportsFilterStatus);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Action failed');
    } finally {
      reportsActionLoading = '';
    }
  }

  async function dismissReport(id: string) {
    reportsActionLoading = id;
    try {
      await adminDismissReport(id);
      toast.success('Report dismissed');
      reports = reports.map(r => r.id === id ? { ...r, status: 'dismissed' } : r);
      reports = reports.filter(r => reportsFilterStatus === '' || r.status === reportsFilterStatus);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Action failed');
    } finally {
      reportsActionLoading = '';
    }
  }

  function reportStatusBadge(status: string) {
    switch (status) {
      case 'reviewed':  return { label: 'Reviewed',  cls: 'text-green-400 bg-green-400/10 border-green-400/20' };
      case 'dismissed': return { label: 'Dismissed', cls: 'text-[#e2e8f0]/40 bg-[#e2e8f0]/5 border-[#1e3a5f]' };
      default:          return { label: 'Pending',   cls: 'text-yellow-400 bg-yellow-400/10 border-yellow-400/20' };
    }
  }

  // ---- Mount ----
  onMount(async () => {
    if (!data.status?.admin) { goto('/'); return; }
    await Promise.all([loadApps(), loadUsers(), loadOrgs(), loadReports()]);
  });
</script>

<div class="mx-auto w-full max-w-4xl px-6 py-16">
  <!-- Page header -->
  <div class="mb-8">
    <h1 class="text-3xl font-bold text-[#ffd700]">Administration</h1>
    <p class="mt-1 text-sm text-[#e2e8f0]/50">Manage applications, users, org logos, and reports</p>
  </div>

  <!-- Tab bar -->
  <div class="mb-8 border-b border-[#1e3a5f]">
    <nav class="-mb-px flex gap-0 overflow-x-auto" aria-label="Admin sections">
      {#each tabs as tab}
        <button
          onclick={() => setTab(tab.id)}
          class="shrink-0 border-b-2 px-5 py-3 text-sm font-medium transition-colors
            {activeTab === tab.id
              ? 'border-[#ffd700] text-[#ffd700]'
              : 'border-transparent text-[#e2e8f0]/50 hover:border-[#e2e8f0]/20 hover:text-[#e2e8f0]'}"
        >
          {tab.label}
        </button>
      {/each}
    </nav>
  </div>

  {#if !data.status?.admin}
    <div class="rounded-2xl border border-red-900/40 bg-red-950/20 p-8 text-center">
      <AlertCircle class="mx-auto mb-3 h-10 w-10 text-red-400/60" />
      <p class="text-[#e2e8f0]/60">You do not have admin access.</p>
    </div>

  {:else}

    <!-- ===== Apps tab ===== -->
    {#if activeTab === 'apps'}
      <div class="mb-6 flex items-start justify-between gap-4">
        <div>
          <h2 class="text-xl font-semibold text-[#e2e8f0]">App Registrations</h2>
          <p class="mt-0.5 text-xs text-[#e2e8f0]/50">Review and approve OIDC client registrations</p>
        </div>
        <div class="flex overflow-hidden rounded-lg border border-[#1e3a5f] text-xs font-medium">
          {#each [['pending', 'Pending'], ['approved', 'Approved'], ['rejected', 'Rejected'], ['', 'All']] as [val, label]}
            <button
              onclick={() => { appsFilterStatus = val; loadApps(); }}
              class="px-3 py-2 transition-colors {appsFilterStatus === val ? 'bg-[#1e3a5f] text-[#00d4ff]' : 'text-[#e2e8f0]/50 hover:text-[#e2e8f0]'}"
            >{label}</button>
          {/each}
        </div>
      </div>

      {#if appsLoading}
        <div class="py-16 text-center text-sm text-[#e2e8f0]/40">
          <LoaderCircle class="mx-auto mb-3 h-6 w-6 animate-spin opacity-40" />
          Loading…
        </div>
      {:else if apps.length === 0}
        <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-12 text-center">
          <CheckCircle class="mx-auto mb-4 h-12 w-12 text-green-400/40" />
          <p class="text-[#e2e8f0]/50">No applications match the current filter.</p>
        </div>
      {:else}
        <div class="space-y-4">
          {#each apps as app (app.id)}
            {@const badge = appStatusBadge(app.status)}
            <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-5">
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
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
                  <div class="mt-2 flex flex-wrap gap-1">
                    {#each app.redirect_uris.slice(0, 3) as uri}
                      <code class="rounded bg-[#0a0e1a] px-1.5 py-0.5 text-[10px] text-[#e2e8f0]/50">{uri}</code>
                    {/each}
                    {#if app.redirect_uris.length > 3}
                      <span class="text-xs text-[#e2e8f0]/30">+{app.redirect_uris.length - 3} more</span>
                    {/if}
                  </div>
                </div>
                {#if app.status !== 'approved'}
                  <button
                    onclick={() => approveApp(app.id)}
                    disabled={appsActionLoading}
                    class="flex shrink-0 items-center gap-1.5 rounded-lg border border-green-500/30 bg-green-500/10 px-3 py-1.5 text-xs font-medium text-green-400 transition-colors hover:bg-green-500/20 disabled:opacity-50"
                  >
                    <CheckCircle class="h-3.5 w-3.5" />
                    Approve
                  </button>
                {/if}
                {#if app.status !== 'rejected'}
                  <button
                    onclick={() => openRejectModal(app.id)}
                    disabled={appsActionLoading}
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
    {/if}

    <!-- ===== Users tab ===== -->
    {#if activeTab === 'users'}
      <div class="mb-6 flex items-center gap-3">
        <Users class="h-6 w-6 text-[#ffd700]" />
        <div>
          <h2 class="text-xl font-semibold text-[#e2e8f0]">Users &amp; Handles</h2>
          <p class="mt-0.5 text-xs text-[#e2e8f0]/50">Manage verified users and RSI handle blocks</p>
        </div>
      </div>

      {#if usersLoading}
        <div class="py-16 text-center text-sm text-[#e2e8f0]/40">
          <LoaderCircle class="mx-auto mb-3 h-6 w-6 animate-spin opacity-40" />
          Loading…
        </div>
      {:else}
        <!-- Verified users -->
        <section class="mb-10">
          <h3 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[#e2e8f0]/50">
            Verified users ({users.length})
          </h3>
          {#if users.length === 0}
            <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-8 text-center text-sm text-[#e2e8f0]/40">
              No verified users yet.
            </div>
          {:else}
            <div class="overflow-hidden rounded-xl border border-[#1e3a5f]">
              <table class="w-full text-sm">
                <thead class="bg-[#0d1526] text-xs text-[#e2e8f0]/50">
                  <tr>
                    <th class="px-4 py-2.5 text-left font-medium">RSI Handle</th>
                    <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Verified</th>
                    <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Status</th>
                    <th class="px-4 py-2.5 text-right font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
                  {#each users as user (user.user_id)}
                    <tr class="transition-colors hover:bg-[#0d1526]/60">
                      <td class="px-4 py-3 font-medium text-[#e2e8f0]">{user.handle}</td>
                      <td class="hidden px-4 py-3 text-xs text-[#e2e8f0]/50 md:table-cell">
                        {new Date(user.verified_at).toLocaleDateString()}
                      </td>
                      <td class="hidden px-4 py-3 md:table-cell">
                        {#if user.handle_blocked}
                          <span class="rounded border border-red-500/20 bg-red-500/10 px-1.5 py-0.5 text-[10px] font-medium text-red-400">Handle blocked</span>
                        {:else}
                          <span class="rounded border border-green-500/20 bg-green-500/10 px-1.5 py-0.5 text-[10px] font-medium text-green-400">Active</span>
                        {/if}
                      </td>
                      <td class="px-4 py-3">
                        <div class="flex justify-end gap-1.5">
                          {#if !user.handle_blocked}
                            <button
                              onclick={() => { blockTarget = user.handle; blockReason = ''; showBlockModal = true; }}
                              class="flex items-center gap-1 rounded border border-yellow-500/20 bg-yellow-500/10 px-2 py-1 text-xs text-yellow-400 transition-colors hover:bg-yellow-500/20"
                              title="Block handle only"
                            >
                              <ShieldX class="h-3 w-3" />
                              Block
                            </button>
                          {/if}
                          <button
                            onclick={() => { deleteTarget = user; deleteBlockHandle = false; deleteReason = ''; }}
                            class="flex items-center gap-1 rounded border border-red-500/20 bg-red-500/10 px-2 py-1 text-xs text-red-400 transition-colors hover:bg-red-500/20"
                            title="Remove user data"
                          >
                            <Trash2 class="h-3 w-3" />
                            Remove
                          </button>
                        </div>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </section>

        <!-- Blocked handles -->
        <section>
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-sm font-semibold uppercase tracking-wider text-[#e2e8f0]/50">
              Blocked handles ({blockedHandles.length})
            </h3>
            <button
              onclick={() => { blockTarget = ''; blockReason = ''; showBlockModal = true; }}
              class="flex items-center gap-1.5 rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-1.5 text-xs font-medium text-yellow-400 transition-colors hover:bg-yellow-500/20"
            >
              <ShieldX class="h-3.5 w-3.5" />
              Block a Handle
            </button>
          </div>
          {#if blockedHandles.length === 0}
            <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-8 text-center text-sm text-[#e2e8f0]/40">
              No handles are currently blocked.
            </div>
          {:else}
            <div class="overflow-hidden rounded-xl border border-[#1e3a5f]">
              <table class="w-full text-sm">
                <thead class="bg-[#0d1526] text-xs text-[#e2e8f0]/50">
                  <tr>
                    <th class="px-4 py-2.5 text-left font-medium">Handle</th>
                    <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Blocked</th>
                    <th class="hidden px-4 py-2.5 text-left font-medium md:table-child">Reason</th>
                    <th class="px-4 py-2.5 text-right font-medium">Action</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
                  {#each blockedHandles as bh (bh.handle)}
                    <tr class="transition-colors hover:bg-[#0d1526]/60">
                      <td class="px-4 py-3 font-medium text-[#e2e8f0]">{bh.handle}</td>
                      <td class="hidden px-4 py-3 text-xs text-[#e2e8f0]/50 md:table-cell">
                        {new Date(bh.blocked_at).toLocaleDateString()}
                        {#if bh.blocked_by}<span class="ml-1 text-[#e2e8f0]/30">by {bh.blocked_by}</span>{/if}
                      </td>
                      <td class="hidden max-w-[200px] truncate px-4 py-3 text-xs text-[#e2e8f0]/50 md:table-cell">
                        {bh.reason || '—'}
                      </td>
                      <td class="px-4 py-3 text-right">
                        <button
                          onclick={() => unblockHandle(bh.handle)}
                          disabled={usersActionLoading}
                          class="flex items-center gap-1 rounded border border-green-500/20 bg-green-500/10 px-2 py-1 text-xs text-green-400 transition-colors hover:bg-green-500/20 disabled:opacity-50"
                        >
                          <ShieldCheck class="h-3 w-3" />
                          Unblock
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </section>
      {/if}
    {/if}

    <!-- ===== Orgs tab ===== -->
    {#if activeTab === 'orgs'}
      <div class="mb-6 flex items-center gap-3">
        <Building2 class="h-6 w-6 text-[#ffd700]" />
        <div>
          <h2 class="text-xl font-semibold text-[#e2e8f0]">Org Logos</h2>
          <p class="mt-0.5 text-xs text-[#e2e8f0]/50">Block or unblock cached organisation logos</p>
        </div>
      </div>

      {#if orgsLoading}
        <div class="py-16 text-center text-sm text-[#e2e8f0]/40">
          <LoaderCircle class="mx-auto mb-3 h-6 w-6 animate-spin opacity-40" />
          Loading…
        </div>
      {:else if orgs.length === 0}
        <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-12 text-center text-sm text-[#e2e8f0]/40">
          No organisations in cache yet.
        </div>
      {:else}
        <div class="overflow-hidden rounded-xl border border-[#1e3a5f]">
          <table class="w-full text-sm">
            <thead class="bg-[#0d1526] text-xs text-[#e2e8f0]/50">
              <tr>
                <th class="px-4 py-2.5 text-left font-medium">SID</th>
                <th class="hidden px-4 py-2.5 text-left font-medium sm:table-cell">Name</th>
                <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Logo</th>
                <th class="hidden px-4 py-2.5 text-left font-medium md:table-cell">Cached</th>
                <th class="px-4 py-2.5 text-right font-medium">Action</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
              {#each orgs as org (org.sid)}
                <tr class="transition-colors hover:bg-[#0d1526]/60">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      {#if org.has_logo && !org.logo_blocked}
                        <img
                          src="/api/orgs/{org.sid}/logo"
                          alt=""
                          class="h-7 w-7 rounded border border-[#1e3a5f] object-contain bg-[#0d1526]"
                          loading="lazy"
                        />
                      {:else}
                        <div class="flex h-7 w-7 items-center justify-center rounded border border-[#1e3a5f] bg-[#0d1526]">
                          <ImageOff class="h-3.5 w-3.5 text-[#e2e8f0]/20" />
                        </div>
                      {/if}
                      <span class="font-mono font-semibold text-[#e2e8f0]">{org.sid}</span>
                    </div>
                  </td>
                  <td class="hidden px-4 py-3 text-[#e2e8f0]/70 sm:table-cell">{org.name || '—'}</td>
                  <td class="hidden px-4 py-3 md:table-cell">
                    {#if org.logo_blocked}
                      <span class="rounded border border-red-500/20 bg-red-500/10 px-1.5 py-0.5 text-[10px] font-medium text-red-400">Blocked</span>
                    {:else if org.has_logo}
                      <span class="rounded border border-green-500/20 bg-green-500/10 px-1.5 py-0.5 text-[10px] font-medium text-green-400">Has logo</span>
                    {:else}
                      <span class="rounded border border-[#1e3a5f] px-1.5 py-0.5 text-[10px] font-medium text-[#e2e8f0]/30">No logo</span>
                    {/if}
                  </td>
                  <td class="hidden px-4 py-3 text-xs text-[#e2e8f0]/40 md:table-cell">
                    {new Date(org.fetched_at).toLocaleDateString()}
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button
                      onclick={() => toggleOrgLogo(org)}
                      disabled={orgsActionLoading === org.sid}
                      class="flex items-center gap-1.5 rounded border px-2.5 py-1.5 text-xs font-medium transition-colors disabled:opacity-50
                        {org.logo_blocked
                          ? 'border-green-500/20 bg-green-500/10 text-green-400 hover:bg-green-500/20'
                          : 'border-red-500/20 bg-red-500/10 text-red-400 hover:bg-red-500/20'}"
                    >
                      {#if orgsActionLoading === org.sid}
                        <LoaderCircle class="h-3 w-3 animate-spin" />
                      {:else if org.logo_blocked}
                        <ShieldCheck class="h-3 w-3" />
                        Unblock
                      {:else}
                        <ShieldX class="h-3 w-3" />
                        Block Logo
                      {/if}
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    {/if}

    <!-- ===== Reports tab ===== -->
    {#if activeTab === 'reports'}
      <div class="mb-6 flex items-start justify-between gap-4">
        <div class="flex items-center gap-3">
          <Flag class="h-6 w-6 text-[#ffd700]" />
          <div>
            <h2 class="text-xl font-semibold text-[#e2e8f0]">Report Queue</h2>
            <p class="mt-0.5 text-xs text-[#e2e8f0]/50">Review abuse reports submitted by the community</p>
          </div>
        </div>
        <div class="flex overflow-hidden rounded-lg border border-[#1e3a5f] text-xs font-medium">
          {#each [['pending', 'Pending'], ['reviewed', 'Reviewed'], ['dismissed', 'Dismissed'], ['', 'All']] as [val, label]}
            <button
              onclick={() => { reportsFilterStatus = val; loadReports(); }}
              class="px-3 py-2 transition-colors {reportsFilterStatus === val ? 'bg-[#1e3a5f] text-[#00d4ff]' : 'text-[#e2e8f0]/50 hover:text-[#e2e8f0]'}"
            >{label}</button>
          {/each}
        </div>
      </div>

      {#if reportsLoading}
        <div class="py-16 text-center text-sm text-[#e2e8f0]/40">
          <LoaderCircle class="mx-auto mb-3 h-6 w-6 animate-spin opacity-40" />
          Loading…
        </div>
      {:else if reports.length === 0}
        <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-12 text-center">
          <CheckCircle class="mx-auto mb-4 h-12 w-12 text-green-400/40" />
          <p class="text-[#e2e8f0]/50">No reports match the current filter.</p>
        </div>
      {:else}
        <div class="space-y-3">
          {#each reports as report (report.id)}
            {@const badge = reportStatusBadge(report.status)}
            <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-4">
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <div class="mb-1.5 flex flex-wrap items-center gap-2">
                    {#if report.type === 'user'}
                      <span class="inline-flex items-center gap-1 rounded border border-[#00d4ff]/20 bg-[#00d4ff]/10 px-1.5 py-0.5 text-[10px] font-medium text-[#00d4ff]">
                        <User class="h-2.5 w-2.5" />User
                      </span>
                    {:else}
                      <span class="inline-flex items-center gap-1 rounded border border-purple-400/20 bg-purple-400/10 px-1.5 py-0.5 text-[10px] font-medium text-purple-400">
                        <Building2 class="h-2.5 w-2.5" />Org
                      </span>
                    {/if}
                    <span class="rounded border px-1.5 py-0.5 text-[10px] font-medium {badge.cls}">{badge.label}</span>
                    <span class="font-semibold text-[#e2e8f0]">{report.target}</span>
                  </div>
                  <p class="line-clamp-2 text-sm text-[#e2e8f0]/70">{report.reason}</p>
                  <div class="mt-1.5 flex flex-wrap gap-x-4 text-xs text-[#e2e8f0]/40">
                    <span>IP: {report.reporter_ip}</span>
                    <span>{new Date(report.created_at).toLocaleString()}</span>
                    {#if report.reviewed_by}<span>Actioned by {report.reviewed_by}</span>{/if}
                  </div>
                </div>
                {#if report.status === 'pending'}
                  <div class="flex shrink-0 flex-col gap-1.5 sm:flex-row">
                    <button
                      onclick={() => reviewReport(report.id)}
                      disabled={reportsActionLoading === report.id}
                      class="flex items-center gap-1 rounded border border-green-500/20 bg-green-500/10 px-2.5 py-1.5 text-xs font-medium text-green-400 transition-colors hover:bg-green-500/20 disabled:opacity-50"
                    >
                      {#if reportsActionLoading === report.id}
                        <LoaderCircle class="h-3 w-3 animate-spin" />
                      {:else}
                        <CheckCircle class="h-3 w-3" />
                      {/if}
                      Reviewed
                    </button>
                    <button
                      onclick={() => dismissReport(report.id)}
                      disabled={reportsActionLoading === report.id}
                      class="flex items-center gap-1 rounded border border-[#1e3a5f] bg-[#0a0e1a] px-2.5 py-1.5 text-xs font-medium text-[#e2e8f0]/50 transition-colors hover:text-[#e2e8f0] disabled:opacity-50"
                    >
                      {#if reportsActionLoading === report.id}
                        <LoaderCircle class="h-3 w-3 animate-spin" />
                      {:else}
                        <XCircle class="h-3 w-3" />
                      {/if}
                      Dismiss
                    </button>
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}

  {/if}
</div>

<!-- ===== Reject app modal ===== -->
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
        >Cancel</button>
        <button
          onclick={confirmReject}
          disabled={appsActionLoading}
          class="rounded-lg bg-red-600/80 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-600 disabled:opacity-50"
        >Confirm Reject</button>
      </div>
    </div>
  </div>
{/if}

<!-- ===== Delete user modal ===== -->
{#if deleteTarget}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    onclick={(e) => { if (e.target === e.currentTarget && !usersActionLoading) deleteTarget = null; }}
    onkeydown={(e) => { if (e.key === 'Escape' && !usersActionLoading) deleteTarget = null; }}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="w-full max-w-md rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6 shadow-xl">
      <h2 class="mb-1 text-base font-semibold text-[#e2e8f0]">Remove user <span class="text-red-400">{deleteTarget.handle}</span>?</h2>
      <p class="mb-4 text-xs text-[#e2e8f0]/50">
        This will delete all local verification data for this user. Their Pocket ID account remains untouched.
      </p>
      <label class="mb-4 flex cursor-pointer items-start gap-2.5">
        <input type="checkbox" bind:checked={deleteBlockHandle} class="mt-0.5 accent-red-500" />
        <span class="text-sm text-[#e2e8f0]/80">Also block this RSI handle from re-verifying</span>
      </label>
      {#if deleteBlockHandle}
        <label for="del-reason" class="mb-1.5 block text-xs text-[#e2e8f0]/60">Block reason (optional)</label>
        <input
          id="del-reason"
          type="text"
          bind:value={deleteReason}
          placeholder="Reason for blocking…"
          class="mb-4 w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 outline-none focus:border-red-500/50"
        />
      {/if}
      <div class="flex justify-end gap-3">
        <button
          onclick={() => deleteTarget = null}
          disabled={usersActionLoading}
          class="rounded-lg border border-[#1e3a5f] px-4 py-2 text-sm text-[#e2e8f0]/60 transition-colors hover:text-[#e2e8f0] disabled:opacity-50"
        >Cancel</button>
        <button
          onclick={confirmDelete}
          disabled={usersActionLoading}
          class="flex items-center gap-1.5 rounded-lg border border-red-500/30 bg-red-600/80 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-red-600 disabled:opacity-50"
        >
          {#if usersActionLoading}<LoaderCircle class="h-4 w-4 animate-spin" />{/if}
          Remove User
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- ===== Block handle modal ===== -->
{#if showBlockModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
    onclick={(e) => { if (e.target === e.currentTarget && !usersActionLoading) showBlockModal = false; }}
    onkeydown={(e) => { if (e.key === 'Escape' && !usersActionLoading) showBlockModal = false; }}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="w-full max-w-md rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-6 shadow-xl">
      <h2 class="mb-4 text-base font-semibold text-[#e2e8f0]">Block RSI Handle</h2>
      <label for="block-handle" class="mb-1.5 block text-xs text-[#e2e8f0]/60">RSI Handle</label>
      <input
        id="block-handle"
        type="text"
        bind:value={blockTarget}
        placeholder="exact RSI handle"
        class="mb-3 w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 outline-none focus:border-yellow-500/50"
      />
      <label for="block-reason" class="mb-1.5 block text-xs text-[#e2e8f0]/60">Reason (optional)</label>
      <input
        id="block-reason"
        type="text"
        bind:value={blockReason}
        placeholder="Reason…"
        class="mb-4 w-full rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] px-3 py-2 text-sm text-[#e2e8f0] placeholder-[#e2e8f0]/30 outline-none focus:border-yellow-500/50"
      />
      <div class="flex justify-end gap-3">
        <button
          onclick={() => showBlockModal = false}
          disabled={usersActionLoading}
          class="rounded-lg border border-[#1e3a5f] px-4 py-2 text-sm text-[#e2e8f0]/60 transition-colors hover:text-[#e2e8f0] disabled:opacity-50"
        >Cancel</button>
        <button
          onclick={confirmBlock}
          disabled={usersActionLoading || !blockTarget.trim()}
          class="flex items-center gap-1.5 rounded-lg border border-yellow-500/30 bg-yellow-600/80 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-yellow-600 disabled:opacity-50"
        >
          {#if usersActionLoading}<LoaderCircle class="h-4 w-4 animate-spin" />{/if}
          Block Handle
        </button>
      </div>
    </div>
  </div>
{/if}
