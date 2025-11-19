# Use Red Hat UBI Node.js 20 minimal image for dependencies
FROM registry.access.redhat.com/ubi9/nodejs-20-minimal AS deps

WORKDIR /app

USER 0

# Install dependencies based on the preferred package manager
COPY package.json package-lock.json* .npmrc ./
RUN npm ci --legacy-peer-deps

# Rebuild the source code only when needed
FROM registry.access.redhat.com/ubi9/nodejs-20-minimal AS builder

USER 0

WORKDIR /app

# Copy node_modules from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Next.js collects completely anonymous telemetry data about general usage.
# Learn more here: https://nextjs.org/telemetry
# Uncomment the following line in case you want to disable telemetry during the build.
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

# Production image, copy all the files and run next
FROM registry.access.redhat.com/ubi9/nodejs-20-minimal AS runner

WORKDIR /app

ENV NODE_ENV=production
# Uncomment the following line in case you want to disable telemetry during runtime.
ENV NEXT_TELEMETRY_DISABLED=1

# Copy public assets
COPY --from=builder /app/public ./public

USER 0

# Automatically leverage output traces to reduce image size
# https://nextjs.org/docs/advanced-features/output-file-tracing
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# Create directories and set permissions for OpenShift arbitrary UIDs
# OpenShift runs containers with random UIDs in the root group (GID 0)
# chmod g=u gives the root group the same permissions as the owner
RUN mkdir -p .next && \
    chmod -R g=u /app && \
    chgrp -R 0 /app

USER 1001

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# server.js is created by next build from the standalone output
# https://nextjs.org/docs/pages/api-reference/next-config-js/output
CMD ["node", "server.js"]
