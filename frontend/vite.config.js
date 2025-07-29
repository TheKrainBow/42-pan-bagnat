import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      Global: path.resolve(__dirname, 'src/Global'),
      Pages: path.resolve(__dirname, 'src/Pages'),
      Modules: path.resolve(__dirname, 'src/Pages/Modules'),
    },
  },
});
