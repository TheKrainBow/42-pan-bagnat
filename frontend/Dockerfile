# Use an official Node.js runtime as a base image
FROM node:18-alpine

# Set the working directory in the container
WORKDIR /app

# Install pnpm
RUN npm install -g pnpm

# Copy package.json and pnpm-lock.yaml first to leverage Docker cache
COPY package.json pnpm-lock.yaml ./

# Install dependencies using pnpm
RUN pnpm install

# Copy the rest of the frontend code into the container
COPY . .

# Build the frontend app
RUN pnpm run build

# Expose the port the app runs on
EXPOSE 3000

# Start the app (adjust if you are using a custom start script)
CMD ["pnpm", "start"]
