# Node menjalankan TypeScript langsung (type stripping, Node >= 22.18), jadi tidak
# ada tahap compile. Tidak ada dependency runtime -> tidak perlu npm install di image.
FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY package.json ./
COPY src ./src
# image resmi menyediakan user non-root "node"
USER node
EXPOSE 8080
CMD ["node", "src/main.ts"]
