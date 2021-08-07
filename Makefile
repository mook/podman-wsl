podman-wsl.exe: image/podman-wsl-distro.tar.gz
	GOOS=windows go build -tags 'remote containers_image_openpgp' .

image/podman-wsl-distro.tar.gz:
	${MAKE} -C image podman-wsl-distro.tar.gz
