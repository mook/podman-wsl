podman-wsl.exe: image/podman-wsl-distro.tar
	GOOS=windows go build -tags 'remote containers_image_openpgp' .

image/podman-wsl-distro.tar:
	${MAKE} -C image podman-wsl-distro.tar
