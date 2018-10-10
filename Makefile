.PHONY: lint

lint:
	gometalinter ./... --vendor --skip=vendor --exclude=\.*_mock\.*\.go --exclude=vendor\.* --cyclo-over=15 --deadline=10m --disable-all \
        --enable=errcheck \
        --enable=vet \
        --enable=deadcode \
        --enable=gocyclo \
        --enable=golint \
        --enable=varcheck \
        --enable=structcheck \
        --enable=maligned \
        --enable=vetshadow \
        --enable=ineffassign \
        --enable=interfacer \
        --enable=unconvert \
        --enable=goconst \
        --enable=gosimple \
        --enable=staticcheck \
        --enable=gosec