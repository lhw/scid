type PublicEnv = {
	PUBLIC_POCKET_ID_URL?: string;
	PUBLIC_OIDC_CLIENT_ID?: string;
	PUBLIC_TURNSTILE_SITE_KEY?: string;
};

const defaults: Required<PublicEnv> = {
	PUBLIC_POCKET_ID_URL: 'https://auth.scid.my',
	PUBLIC_OIDC_CLIENT_ID: 'scid-frontend',
	PUBLIC_TURNSTILE_SITE_KEY: '',
};

const runtimeEnv = (globalThis as { __SCID_PUBLIC_ENV__?: PublicEnv }).__SCID_PUBLIC_ENV__;

export const PUBLIC_POCKET_ID_URL = runtimeEnv?.PUBLIC_POCKET_ID_URL ?? defaults.PUBLIC_POCKET_ID_URL;
export const PUBLIC_OIDC_CLIENT_ID = runtimeEnv?.PUBLIC_OIDC_CLIENT_ID ?? defaults.PUBLIC_OIDC_CLIENT_ID;
export const PUBLIC_TURNSTILE_SITE_KEY = runtimeEnv?.PUBLIC_TURNSTILE_SITE_KEY ?? defaults.PUBLIC_TURNSTILE_SITE_KEY;