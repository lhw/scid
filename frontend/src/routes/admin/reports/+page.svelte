<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { Flag, CheckCircle, XCircle, Clock, AlertCircle, LoaderCircle, User, Building2 } from '@lucide/svelte';
  import { goto } from '$app/navigation';
  import type { PageData } from './$types';
  import type { AdminReport } from '$lib/utils/api';
  import { adminListReports, adminReviewReport, adminDismissReport } from '$lib/utils/api';

  let { data }: { data: PageData } = $props();

  let reports = $state<AdminReport[]>([]);
  let loading = $state(true);
  let filterStatus = $state('pending');
  let actionLoading = $state('');   // report id currently being actioned

  onMount(async () => {
    if (!data.status?.admin) { goto('/'); return; }
    await loadReports();
  });

  async function loadReports() {
    loading = true;
    try {
      reports = await adminListReports(filterStatus || undefined);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load reports');
    } finally {
      loading = false;
    }
  }

  async function review(id: string) {
    actionLoading = id;
    try {
      await adminReviewReport(id);
      toast.success('Marked as reviewed');
      reports = reports.map(r => r.id === id ? { ...r, status: 'reviewed' } : r);
      reports = reports.filter(r => filterStatus === '' || r.status === filterStatus);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Action failed');
    } finally {
      actionLoading = '';
    }
  }

  async function dismiss(id: string) {
    actionLoading = id;
    try {
      await adminDismissReport(id);
      toast.success('Report dismissed');
      reports = reports.map(r => r.id === id ? { ...r, status: 'dismissed' } : r);
      reports = reports.filter(r => filterStatus === '' || r.status === filterStatus);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Action failed');
    } finally {
      actionLoading = '';
    }
  }

  function statusBadge(status: string) {
    switch (status) {
      case 'reviewed':  return { label: 'Reviewed',  cls: 'text-green-400 bg-green-400/10 border-green-400/20' };
      case 'dismissed': return { label: 'Dismissed', cls: 'text-[#e2e8f0]/40 bg-[#e2e8f0]/5 border-[#1e3a5f]' };
      default:          return { label: 'Pending',   cls: 'text-yellow-400 bg-yellow-400/10 border-yellow-400/20' };
    }
  }
</script>

<div class="mx-auto w-full max-w-4xl px-6 py-16">
  <!-- Admin sub-nav -->
  <nav class="mb-8 flex gap-2 overflow-x-auto">
    {#each [
      ['/admin/apps', 'App Registrations'],
      ['/admin/users', 'Users & Handles'],
      ['/admin/orgs', 'Org Logos'],
      ['/admin/reports', 'Reports'],
    ] as [href, label]}
      <a
        {href}
        class="shrink-0 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors
          {label === 'Reports'
            ? 'border-[#ffd700]/40 bg-[#ffd700]/5 text-[#ffd700]'
            : 'border-[#1e3a5f] text-[#e2e8f0]/50 hover:text-[#e2e8f0]'}"
      >{label}</a>
    {/each}
  </nav>

  <div class="mb-8 flex items-start justify-between gap-4">
    <div class="flex items-center gap-3">
      <Flag class="h-7 w-7 text-[#ffd700]" />
      <div>
        <h1 class="text-2xl font-bold text-[#ffd700]">Report Queue</h1>
        <p class="text-xs text-[#e2e8f0]/50">Review abuse reports submitted by the community</p>
      </div>
    </div>
    <!-- Filter tabs -->
    <div class="flex overflow-hidden rounded-lg border border-[#1e3a5f] text-xs font-medium">
      {#each [['pending', 'Pending'], ['reviewed', 'Reviewed'], ['dismissed', 'Dismissed'], ['', 'All']] as [val, label]}
        <button
          onclick={() => { filterStatus = val; loadReports(); }}
          class="px-3 py-2 transition-colors {filterStatus === val ? 'bg-[#1e3a5f] text-[#00d4ff]' : 'text-[#e2e8f0]/50 hover:text-[#e2e8f0]'}"
        >{label}</button>
      {/each}
    </div>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-[#e2e8f0]/40">
      <LoaderCircle class="mx-auto mb-3 h-6 w-6 animate-spin opacity-40" />
      Loading…
    </div>
  {:else if !data.status?.admin}
    <div class="rounded-2xl border border-red-900/40 bg-red-950/20 p-8 text-center">
      <AlertCircle class="mx-auto mb-3 h-10 w-10 text-red-400/60" />
      <p class="text-[#e2e8f0]/60">You do not have admin access.</p>
    </div>
  {:else if reports.length === 0}
    <div class="rounded-2xl border border-[#1e3a5f] bg-[#0d1526] p-12 text-center">
      <CheckCircle class="mx-auto mb-4 h-12 w-12 text-green-400/40" />
      <p class="text-[#e2e8f0]/50">No reports match the current filter.</p>
    </div>
  {:else}
    <div class="space-y-3">
      {#each reports as report (report.id)}
        {@const badge = statusBadge(report.status)}
        <div class="rounded-xl border border-[#1e3a5f] bg-[#0d1526] p-4">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <!-- Header row -->
              <div class="flex flex-wrap items-center gap-2 mb-1.5">
                <!-- Type badge -->
                {#if report.type === 'user'}
                  <span class="inline-flex items-center gap-1 rounded border border-[#00d4ff]/20 bg-[#00d4ff]/10 px-1.5 py-0.5 text-[10px] font-medium text-[#00d4ff]">
                    <User class="h-2.5 w-2.5" />User
                  </span>
                {:else}
                  <span class="inline-flex items-center gap-1 rounded border border-purple-400/20 bg-purple-400/10 px-1.5 py-0.5 text-[10px] font-medium text-purple-400">
                    <Building2 class="h-2.5 w-2.5" />Org
                  </span>
                {/if}
                <!-- Status badge -->
                <span class="rounded border px-1.5 py-0.5 text-[10px] font-medium {badge.cls}">{badge.label}</span>
                <!-- Target -->
                <span class="font-semibold text-[#e2e8f0]">{report.target}</span>
              </div>
              <!-- Reason -->
              <p class="line-clamp-2 text-sm text-[#e2e8f0]/70">{report.reason}</p>
              <!-- Meta -->
              <div class="mt-1.5 flex flex-wrap gap-x-4 text-xs text-[#e2e8f0]/40">
                <span>IP: {report.reporter_ip}</span>
                <span>{new Date(report.created_at).toLocaleString()}</span>
                {#if report.reviewed_by}
                  <span>Actioned by {report.reviewed_by}</span>
                {/if}
              </div>
            </div>
            <!-- Actions -->
            {#if report.status === 'pending'}
              <div class="flex shrink-0 flex-col gap-1.5 sm:flex-row">
                <button
                  onclick={() => review(report.id)}
                  disabled={actionLoading === report.id}
                  class="flex items-center gap-1 rounded border border-green-500/20 bg-green-500/10 px-2.5 py-1.5 text-xs font-medium text-green-400 transition-colors hover:bg-green-500/20 disabled:opacity-50"
                >
                  {#if actionLoading === report.id}
                    <LoaderCircle class="h-3 w-3 animate-spin" />
                  {:else}
                    <CheckCircle class="h-3 w-3" />
                  {/if}
                  Reviewed
                </button>
                <button
                  onclick={() => dismiss(report.id)}
                  disabled={actionLoading === report.id}
                  class="flex items-center gap-1 rounded border border-[#1e3a5f] bg-[#0a0e1a] px-2.5 py-1.5 text-xs font-medium text-[#e2e8f0]/50 transition-colors hover:text-[#e2e8f0] disabled:opacity-50"
                >
                  {#if actionLoading === report.id}
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
</div>
