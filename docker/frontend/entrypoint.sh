#!/bin/sh
# Generate runtime config.json from environment variables
# Falls back to built-in defaults if env vars are not set

CONFIG_PATH="/usr/share/nginx/html/config.json"

# Only regenerate if at least one branding env var is set
if [ -n "$BRANDING_HERO_SUBTITLE" ] || [ -n "$BRANDING_LOGIN_BUTTON_TEXT" ] || [ -n "$BRANDING_LOGIN_DESCRIPTION" ]; then
    cat > "$CONFIG_PATH" <<EOF
{
  "apiBaseUrl": "/api",
  "features": {
    "enableDocumentation": true
  },
  "branding": {
    "heroSubtitle": "${BRANDING_HERO_SUBTITLE:-Browsing GSI Lustre made easy}",
    "loginButtonText": "${BRANDING_LOGIN_BUTTON_TEXT:-Sign in with GSI Account}",
    "loginDescription": "${BRANDING_LOGIN_DESCRIPTION:-Please sign in with your GSI account to access the file browser and other protected resources.}"
  }
}
EOF
    echo "Runtime config.json updated with branding overrides"
fi

# Execute nginx
exec "$@"
