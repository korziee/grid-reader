FROM golang:1.23

RUN apt-get update \
    && apt-get install -y \
        wget build-essential \
        pkg-config \
        --no-install-recommends \
    && apt-get -q -y install \
        libjpeg-dev \
        libpng-dev \
        libtiff-dev \
        libgif-dev \
        libx11-dev \
        fontconfig fontconfig-config libfontconfig1-dev \
        ghostscript gsfonts gsfonts-x11 \
        libfreetype6-dev \
        --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

ARG IMAGEMAGICK_PROJECT=ImageMagick
ARG IMAGEMAGICK_VERSION=7.1.1-11
ENV IMAGEMAGICK_VERSION=$IMAGEMAGICK_VERSION

RUN cd && \
	wget https://github.com/ImageMagick/${IMAGEMAGICK_PROJECT}/archive/${IMAGEMAGICK_VERSION}.tar.gz && \
	tar xvzf ${IMAGEMAGICK_VERSION}.tar.gz && \
	cd ImageMagick* && \
	./configure \
	    --without-magick-plus-plus \
	    --without-perl \
	    --disable-openmp \
	    --with-gvc=no \
	    --with-fontconfig=yes \
	    --with-freetype=yes \
	    --with-gslib \
	    --disable-docs && \
	make -j$(nproc) && make install && \
	ldconfig /usr/local/lib


WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go .
COPY internal/*.go ./internal/

RUN GOOS=linux go build -o grid-reader
COPY . .

EXPOSE 8080

CMD [ "./grid-reader" ]
