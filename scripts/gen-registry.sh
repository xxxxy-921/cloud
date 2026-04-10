#!/bin/bash
# gen-registry.sh — Generate web/src/apps/registry.ts for production builds.
# Usage: APPS=system,ai ./scripts/gen-registry.sh
#
# If APPS is not set, restores the git version (full-featured).

set -euo pipefail

REGISTRY="web/src/apps/registry.ts"

# No APPS specified → restore git version
if [ -z "${APPS:-}" ]; then
  git checkout -- "$REGISTRY" 2>/dev/null || true
  echo "[gen-registry] restored full registry from git"
  exit 0
fi

# Generate filtered registry
cat > "$REGISTRY" << 'HEADER'
import type { RouteObject } from "react-router"

export interface AppModule {
  name: string
  routes: RouteObject[]
}

const modules: AppModule[] = []

export function registerApp(m: AppModule) {
  modules.push(m)
}

export function getAppRoutes(): RouteObject[] {
  return modules.flatMap((m) => m.routes)
}

// Auto-generated — do not edit. Run without APPS to restore full version.
HEADER

for app in $(echo "$APPS" | tr ',' '\n'); do
  [ "$app" = "system" ] && continue  # system is kernel, no app module
  echo "import './${app}/module'" >> "$REGISTRY"
done

echo "[gen-registry] generated registry with APPS=$APPS"
