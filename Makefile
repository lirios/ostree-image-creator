IMAGE=liriorg/ostree-image-creator

image:
	@podman build -t $(IMAGE) .

push:
	@podman push $(shell podman images $(IMAGE) --format '{{.ID}}') docker://docker.io/$(IMAGE)
