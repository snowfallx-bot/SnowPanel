# Frontend

React + TypeScript + Vite admin UI with:
- Tailwind
- shadcn-style UI components
- Zustand auth store
- Axios API wrapper
- React Router guards
- Dashboard, Files, Services, Docker, Cron, and Audit pages

## Run

1. `npm install`
2. `npm run dev`

Optional env:
- `VITE_API_BASE_URL` (default empty, meaning same-origin requests such as `/api/v1/...`)
- `VITE_API_PROXY_TARGET` (default `http://127.0.0.1:8080` for local Vite proxy)
