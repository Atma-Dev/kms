# syntax=docker/dockerfile:1.7

FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
# --ignore-scripts: `postinstall` runs `nuxt prepare`, which needs the full
# source (nuxt.config.ts, app/) to generate correct types/aliases. Running it
# here — before COPY . . — bakes a broken .nuxt into this cached layer that
# never gets fixed, since `nuxt build` below does not regenerate it.
RUN npm ci --ignore-scripts

COPY . .
RUN npm run build

FROM node:22-alpine

RUN apk add --no-cache curl

WORKDIR /app

COPY --from=builder /app/.output ./.output

EXPOSE 3000

ENV HOST=0.0.0.0
ENV PORT=3000

CMD ["node", ".output/server/index.mjs"]
