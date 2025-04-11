// This file is kept as a placeholder for backward compatibility
// The actual OIDC handling is now done on the backend with the BFF pattern

import { getConfig } from './config';

// Return minimal config for backward compatibility
export default function getOidcConfig() {
    const config = getConfig();

    // Return minimal config with just what might be needed for references
    return {
        authority: config.oidc?.issuer || '',
        clientId: config.oidc?.clientId || '',
        redirectUri: '/api/auth/callback',
    };
}
