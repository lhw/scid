<script lang="ts">
  import { onMount } from 'svelte';
  import { toast } from 'svelte-sonner';
  import { Building2, ShieldX, ShieldCheck, ImageOff, AlertCircle, LoaderCircle } from '@lucide/svelte';
  import { goto } from '$app/navigation';
  import type { PageData } from './$types';
  import type { AdminOrgEntry } from '$lib/utils/api';
  import { adminListOrgs, adminBlockOrgLogo, adminUnblockOrgLogo } from '$lib/utils/api';

  let { data }: { data: PageData } = $props();

  let orgs = $state<AdminOrgEntry[]>([]);
  let loading = $state(true);
  let actionLoading = $state('');   // SID of org currently being toggled

  onMount(async () => {
    if (!data.status?.admin) { goto('/'); return; }
    await loadOrgs();
  });

  async function loadOrgs() {
    loading = true;
    try {
      orgs = await adminListOrgs();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load orgs');
    } finally {
      loading = false;
    }
  }

  async function toggleBlock(org: AdminOrgEntry) {
    actionLoading = org.sid;
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
      actionLoading = '';
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
          {label === 'Org Logos'
            ? 'border-[#ffd700]/40 bg-[#ffd700]/5 text-[#ffd700]'
            : 'border-[#1e3a5f] text-[#e2e8f0]/50 hover:text-[#e2e8f0]'}"
      >{label}</a>
    {/each}
  </nav>

  <div class="mb-8 flex items-center gap-3">
    <Building2 class="h-7 w-7 text-[#ffd700]" />
    <div>
      <h1 class="text-2xl font-bold text-[#ffd700]">Org Logos</h1>
      <p class="text-xs text-[#e2e8f0]/50">Block or unblock cached organisation logos</p>
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
                  onclick={() => toggleBlock(org)}
                  disabled={actionLoading === org.sid}
                  class="flex items-center gap-1.5 rounded border px-2.5 py-1.5 text-xs font-medium transition-colors disabled:opacity-50
                    {org.logo_blocked
                      ? 'border-green-500/20 bg-green-500/10 text-green-400 hover:bg-green-500/20'
                      : 'border-red-500/20 bg-red-500/10 text-red-400 hover:bg-red-500/20'}"
                >
                  {#if actionLoading === org.sid}
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
</div>
