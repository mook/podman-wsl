podman-wsl.exe:
	GOOS=windows go build -tags 'remote containers_image_openpgp' .
