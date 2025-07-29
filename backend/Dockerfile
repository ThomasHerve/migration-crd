# Base image
FROM node:22

RUN apt-get update && \
    apt-get install -y python3 python3-pip ffmpeg pipx && \
    pipx install yt-dlp && \
    ln -s /root/.local/bin/yt-dlp /usr/local/bin/yt-dlp

# Create app directory
WORKDIR /usr/src/app

# A wildcard is used to ensure both package.json AND package-lock.json are copied
COPY package*.json ./

# Install app dependencies
RUN npm install

# Bundle app source
COPY . .

# Creates a "dist" folder with the production build
RUN npm run build

# Start the server using the production build
CMD [ "node", "dist/main.js" ]