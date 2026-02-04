# Build Stage
FROM node:20-alpine as builder
WORKDIR /app
# Only copy package.json to avoid using the potentially tainted package-lock.json from local env
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

# Production Stage
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
