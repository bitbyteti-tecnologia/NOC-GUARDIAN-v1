# Dockerfile da UI
# - Dev server (porta 5173) exposto via Nginx
FROM node:20-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
EXPOSE 5173
CMD ["npm","run","dev"]
