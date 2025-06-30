FROM node:18-alpine

# 2) Create app directory
WORKDIR /usr/src/app

# 3) Install only production deps
COPY package.json package-lock.json ./
RUN npm ci --only=production

# 4) Copy the rest of the application
COPY . .

# 5) Expose the port your server listens on
EXPOSE 3000

# 6) Default command
CMD ["node", "server.js"]
