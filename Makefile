podman-wsl.exe: image/podman-wsl-distro.tar.gz *.go cmd/*.go config/*.go winapi/*.go wsl/*.go
	GOOS=windows go build -tags 'remote containers_image_openpgp' .

image/podman-wsl-distro.tar.gz: $(filter-out $(wildcard image/podman-wsl-distro.*), $(wildcard image/*))
	${MAKE} -C image podman-wsl-distro.tar.gz
