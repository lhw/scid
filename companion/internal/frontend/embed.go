package frontend

import "embed"

// FS holds the compiled SvelteKit static build, embedded at compile time.
// In Docker the dist/ directory is populated by the frontend build stage before
// go build runs. For local development a placeholder index.html is used; run
// pnpm dev in the frontend/ directory for a real HMR dev server instead.

//go:embed all:dist
var FS embed.FS
