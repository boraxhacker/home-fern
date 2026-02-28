.PHONY: clean run build deb

BINARY_NAME=home-fern
VERSION ?= 0.0.3

build: clean
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build -trimpath -o release/${BINARY_NAME}-amd64 ./cmd/${BINARY_NAME}
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o release/${BINARY_NAME}-alpine ./cmd/${BINARY_NAME}

run:
	go run ./cmd/home-fern

clean:
	go clean
	rm -rf release

deb: build
	mkdir -p release/deb/DEBIAN
	mkdir -p release/deb/usr/bin
	mkdir -p release/deb/etc/systemd/system
	mkdir -p release/deb/usr/share/doc/${BINARY_NAME}
	mkdir -p release/deb/etc/home-fern

	cp release/${BINARY_NAME}-amd64 release/deb/usr/bin/${BINARY_NAME}
	chmod 755 release/deb/usr/bin/${BINARY_NAME}

	cp debian/control release/deb/DEBIAN/
	sed -i 's/Version: .*/Version: ${VERSION}/' release/deb/DEBIAN/control
	cp debian/postinst release/deb/DEBIAN/
	cp debian/prerm release/deb/DEBIAN/
	cp debian/postrm release/deb/DEBIAN/
	chmod 755 release/deb/DEBIAN/postinst
	chmod 755 release/deb/DEBIAN/prerm
	chmod 755 release/deb/DEBIAN/postrm

	cp debian/home-fern.service release/deb/etc/systemd/system/
	# Copy the example config to a non-conffile location. The postinst script will handle creating the real config.
	cp debian/home-fern-config.yaml release/deb/usr/share/doc/${BINARY_NAME}/config.yaml.example

	dpkg-deb --build release/deb release/${BINARY_NAME}_${VERSION}_amd64.deb
	rm -rf release/deb
