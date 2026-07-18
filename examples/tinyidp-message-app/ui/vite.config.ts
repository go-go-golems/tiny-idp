import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({base:"/static/app/",plugins:[react()],build:{outDir:"../static/app",emptyOutDir:true,rollupOptions:{output:{entryFileNames:"assets/main.js",assetFileNames:"assets/[name][extname]"}}},server:{proxy:{"/api":"http://127.0.0.1:8080","/auth":"http://127.0.0.1:8080"}}});
