podman-wsl-distro.tar: podman-wsl-distro.container-id
	podman export --output "$@" "$(shell cat "$<")"
	podman rm --force --ignore "$(shell cat "$<")"

podman-wsl-distro.image-id: Dockerfile containers.conf
	podman build --iidfile "$@" --os linux .

podman-wsl-distro.container-id: podman-wsl-distro.image-id
	podman create "$(shell cat "$<")" > "$@"

.INTERMEDIATE: podman-wsl-distro.container-id
