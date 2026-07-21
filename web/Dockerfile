# syntax=docker/dockerfile:1

FROM node:22-alpine AS builder

WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# Static-file-only runtime. No nginx, no reverse proxy, no TLS handling
# here on purpose, the operator's own reverse proxy in front of this
# container is responsible for all of that (see the root README's
# production checklist).
FROM node:22-alpine AS runtime

WORKDIR /app
RUN npm install -g serve@14

COPY --from=builder /src/dist ./dist

EXPOSE 8091

USER node

CMD ["serve", "-s", "dist", "-l", "8091"]