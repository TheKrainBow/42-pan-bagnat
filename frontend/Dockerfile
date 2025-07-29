FROM node:24-alpine

WORKDIR /app

# Install PNPM globally
RUN npm install -g pnpm

# Copy package metadata first for better cache usage
COPY package.json pnpm-lock.yaml ./
RUN pnpm install

# Copy only necessary source and config files
COPY src/ ./src/
COPY public/ ./public/
COPY index.html .
COPY vite.config.js .

# Build the app
RUN pnpm run build

# Setup PNPM global binary path (used for serve)
ENV PNPM_HOME="/root/.local/share/pnpm"
ENV PATH="$PNPM_HOME:$PATH"

# Install serve to host the built app
RUN pnpm add -g serve

EXPOSE 3000

CMD ["serve", "-s", "dist", "-l", "3000"]
