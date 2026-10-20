import { resolve } from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// `just dev` sets this, and `nisaba serve` proxies :8080 to it (hot-reload websocket included),
// so Vite must listen exactly there. 127.0.0.1, not localhost: that can bind IPv6 only.
const devServer = new URL(process.env.APP_FRONTEND_DEV_SERVER ?? "http://127.0.0.1:5173")

// https://vite.dev/config/
export default defineConfig({
    plugins: [react(), tailwindcss()],
    resolve: {
        alias: {
            "@": resolve(import.meta.dirname, "./src"),
        },
    },
    build: {
        outDir: resolve(import.meta.dirname, "../internal/web/assets/app"),
        emptyOutDir: true,
    },
    server: {
        host: devServer.hostname,
        port: Number(devServer.port),
        strictPort: true,
    },
})
