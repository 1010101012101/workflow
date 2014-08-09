COMPONENTS=builder cache controller database logger registry router

build:
	go build .

package:
	rm -f package
	docker build -t deis/deisctl .
	mkdir -p package
	-docker cp $(shell docker run -d deis/deisctl):/tmp/deisctl.tar.gz package/
