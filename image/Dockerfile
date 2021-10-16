# Using the image from https://github.com/mgoltzsche/podman-static
FROM docker.io/mgoltzsche/podman:latest as build
RUN ln -s /usr/local/bin/podman /usr/bin/podman

FROM scratch
COPY --from=build / /
CMD ["/bin/sh"]

