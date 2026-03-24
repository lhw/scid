<svelte:head>
  <title>OIDC Claims Reference — My SCID Docs</title>
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-6 py-16">
  <nav class="mb-8 text-xs text-[#e2e8f0]/40">
    <a href="/" class="hover:text-[#00d4ff]">Home</a>
    <span class="mx-2">/</span>
    <a href="/docs" class="hover:text-[#00d4ff]">Docs</a>
    <span class="mx-2">/</span>
    <span class="text-[#e2e8f0]/70">Claims Reference</span>
  </nav>

  <header class="mb-12">
    <h1 class="mb-3 text-4xl font-bold text-[#00d4ff]">OIDC Claims Reference</h1>
    <p class="text-lg text-[#e2e8f0]/60">All claims returned in ID tokens and the userinfo endpoint</p>
  </header>

  <div class="space-y-12">

    <p class="text-sm text-[#e2e8f0]/60 leading-relaxed">
      Claims are returned in the <strong class="text-[#e2e8f0]/80">ID token</strong> and on the
      <strong class="text-[#e2e8f0]/80">/api/oidc/userinfo</strong> endpoint when the user grants
      the corresponding scopes.
    </p>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Standard claims</h2>
      <p class="mb-4 text-xs text-[#e2e8f0]/40 uppercase tracking-wider">scope: openid profile email</p>
      <div class="overflow-x-auto rounded-xl border border-[#1e3a5f]">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[#1e3a5f] bg-[#0d1526]">
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Claim</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Type</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Description</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
            {#each [
              { claim: 'sub', type: 'string', desc: 'Stable SCID user ID (UUID)' },
              { claim: 'preferred_username', type: 'string', desc: 'SCID username (chosen at registration)' },
              { claim: 'email', type: 'string', desc: "User's email address" },
              { claim: 'email_verified', type: 'bool', desc: 'Whether the email address has been verified' },
            ] as row}
              <tr>
                <td class="px-4 py-3 font-mono text-xs text-[#00d4ff]">{row.claim}</td>
                <td class="px-4 py-3 font-mono text-xs text-[#e2e8f0]/40">{row.type}</td>
                <td class="px-4 py-3 text-xs text-[#e2e8f0]/60">{row.desc}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">RSI identity claims</h2>
      <p class="mb-4 text-sm text-[#e2e8f0]/60 leading-relaxed">
        These are populated after a user completes RSI bio verification.
        They are always included when the user is in the <code class="rounded bg-[#1e3a5f] px-1 font-mono text-[#00d4ff]">verified</code> group.
      </p>
      <div class="overflow-x-auto rounded-xl border border-[#1e3a5f]">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[#1e3a5f] bg-[#0d1526]">
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Claim</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Type</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Example</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Description</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
            {#each [
              { claim: 'rsi_handle', type: 'string', example: '"CaptainKirk"', desc: 'Verified RSI citizen handle' },
              { claim: 'rsi_verified_at', type: 'string (ISO 8601)', example: '"2025-11-03T14:22:00Z"', desc: 'Timestamp of initial verification' },
              { claim: 'rsi_enlisted', type: 'string (date)', example: '"2013-04-16"', desc: 'RSI enlistment date from public profile' },
              { claim: 'rsi_citizen_record', type: 'string (optional)', example: '"40746"', desc: 'UEE Citizen Record number — omitted when RSI shows n/a' },
            ] as row}
              <tr>
                <td class="px-4 py-3 font-mono text-xs text-[#00d4ff]">{row.claim}</td>
                <td class="px-4 py-3 font-mono text-xs text-[#e2e8f0]/40">{row.type}</td>
                <td class="px-4 py-3 font-mono text-xs text-[#e2e8f0]/40">{row.example}</td>
                <td class="px-4 py-3 text-xs text-[#e2e8f0]/60">{row.desc}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Group claims</h2>
      <div class="mb-6 overflow-x-auto rounded-xl border border-[#1e3a5f]">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[#1e3a5f] bg-[#0d1526]">
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Claim</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Type</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Description</th>
            </tr>
          </thead>
          <tbody class="bg-[#0a0e1a]">
            <tr>
              <td class="px-4 py-3 font-mono text-xs text-[#00d4ff]">groups</td>
              <td class="px-4 py-3 font-mono text-xs text-[#e2e8f0]/40">string[]</td>
              <td class="px-4 py-3 text-xs text-[#e2e8f0]/60">All Pocket ID groups the user belongs to</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h3 class="mb-3 text-sm font-semibold text-[#e2e8f0]/80">Well-known groups</h3>
      <div class="overflow-x-auto rounded-xl border border-[#1e3a5f]">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-[#1e3a5f] bg-[#0d1526]">
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Group name</th>
              <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-[#e2e8f0]/40">Meaning</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[#1e3a5f] bg-[#0a0e1a]">
            {#each [
              { name: 'verified', desc: 'User has a confirmed RSI identity' },
              { name: 'admin', desc: 'SCID site administrator' },
              { name: 'rsi:<SID>', desc: 'Member of RSI org with Spectrum ID <SID> (e.g. rsi:LUG)' },
            ] as row}
              <tr>
                <td class="px-4 py-3 font-mono text-xs text-[#00d4ff]">{row.name}</td>
                <td class="px-4 py-3 text-xs text-[#e2e8f0]/60">{row.desc}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="mt-4 space-y-2 text-sm text-[#e2e8f0]/50">
        <p><code class="font-mono text-xs text-[#00d4ff]">verified</code> is the canonical signal for RSI identity verification.</p>
        <p><code class="font-mono text-xs text-[#00d4ff]">rsi:&lt;SID&gt;</code> groups are always namespaced to avoid collisions with built-in Pocket ID groups.</p>
      </div>
    </section>

    <section>
      <h2 class="mb-4 text-xl font-semibold text-[#e2e8f0]">Example userinfo response</h2>
      <div class="rounded-lg border border-[#1e3a5f] bg-[#0a0e1a] p-4 overflow-x-auto">
<pre class="font-mono text-xs text-[#e2e8f0]/70 whitespace-pre">{`{
  "sub": "3f7a1c2e-...",
  "preferred_username": "MyUsername",
  "email": "pilot@example.com",
  "email_verified": true,
  "rsi_handle": "CaptainKirk",
  "rsi_verified_at": "2025-11-03T14:22:00Z",
  "rsi_enlisted": "2013-04-16",
  "rsi_citizen_record": "40746",
  "groups": ["verified", "rsi:SPAWO", "rsi:LUG"]
}`}</pre>
      </div>
    </section>

  </div>

  <div class="mt-12 flex items-center justify-between border-t border-[#1e3a5f] pt-6">
    <a href="/docs/integration" class="text-sm text-[#e2e8f0]/40 hover:text-[#00d4ff]">← Integration Guide</a>
    <a href="/docs" class="text-sm text-[#e2e8f0]/40 hover:text-[#00d4ff]">Documentation →</a>
  </div>
</div>
