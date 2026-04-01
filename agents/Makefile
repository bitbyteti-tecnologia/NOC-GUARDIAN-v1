APP=nocguardian-agent
VERSION=0.1.0
ARCH=amd64
DIST=dist

.PHONY: all build clean deb rpm publish

all: build

build:
	mkdir -p $(DIST)/linux_$(ARCH)
	GOOS=linux GOARCH=$(ARCH) CGO_ENABLED=0 go build -o $(DIST)/linux_$(ARCH)/$(APP) ./cmd/$(APP)

clean:
	rm -rf $(DIST)

deb: build
	rm -f $(DIST)/*.deb
	fpm -s dir -t deb -n $(APP) -v $(VERSION) -a $(ARCH) --description "NOC Guardian Agent (Linux)" --after-install deploy/deb/postinst --before-remove deploy/deb/prerm $(DIST)/linux_$(ARCH)/$(APP)=/usr/bin/$(APP) deploy/systemd/$(APP).service=/lib/systemd/system/$(APP).service
	mv *.deb $(DIST)/

rpm: build
	rm -f $(DIST)/*.rpm
	fpm -s dir -t rpm -n $(APP) -v $(VERSION) -a x86_64 --description "NOC Guardian Agent (Linux)" --after-install deploy/rpm/post --before-remove deploy/rpm/preun $(DIST)/linux_$(ARCH)/$(APP)=/usr/bin/$(APP) deploy/systemd/$(APP).service=/usr/lib/systemd/system/$(APP).service
	mv *.rpm $(DIST)/

publish: deb rpm
	cp -f $(DIST)/*.deb /opt/noc-guardian/downloads/nocguardian-agent_amd64.deb
	cp -f $(DIST)/*.rpm /opt/noc-guardian/downloads/nocguardian-agent_x86_64.rpm
