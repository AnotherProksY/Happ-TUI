FROM archlinux:base

# Install Go and build tools
RUN pacman -Sy --noconfirm go git base-devel && pacman -Scc --noconfirm

# Set workdir
WORKDIR /src

# Copy your source code
COPY . .

# Build binaries
RUN go build -o happ-tui && cd core && go build -o ../happ-tui-core .

# Default command: show binary info
CMD ["file", "./happ-tui", "./happ-tui-core"]
