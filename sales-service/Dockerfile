FROM public.ecr.aws/docker/library/node:18-alpine

# Update packages and add security patches
RUN apk update && apk upgrade && apk add --no-cache dumb-init

# Create non-root user
RUN addgroup -g 1001 -S nodejs && adduser -S nodejs -u 1001

WORKDIR /app
COPY package*.json ./
RUN npm install

COPY . .
RUN chown -R nodejs:nodejs /app
USER nodejs

EXPOSE 3000
ENTRYPOINT ["dumb-init", "--"]
CMD ["npm", "start"]