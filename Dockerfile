FROM node:20-alpine

WORKDIR /app

RUN apk add --no-cache git curl

COPY package.json ./
RUN npm install

COPY server.js ./

EXPOSE 3000

CMD ["node", "server.js"]
