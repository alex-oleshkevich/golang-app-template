package assets

import "embed"

// app/ is the frontend build output (just ui-build); its .gitkeep keeps the embed valid before the first build.
// Vite empties app/ on every build and copies the .gitkeep back from ui/public.
//
//go:embed all:docs all:app
var FS embed.FS
