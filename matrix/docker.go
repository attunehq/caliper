package matrix

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// DockerClient wraps the Docker SDK client
type DockerClient struct {
	cli *client.Client
}

// NewDockerClient creates a new Docker client
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &DockerClient{cli: cli}, nil
}

// Close closes the Docker client
func (d *DockerClient) Close() error {
	return d.cli.Close()
}

// ContainerConfig holds configuration for creating a container
type ContainerConfig struct {
	Image      string
	CPUs       int
	Memory     int // GB
	WorkingDir string
	MountPath  string // Host path to mount at /workspace
}

// Container represents a running Docker container
type Container struct {
	ID     string
	client *DockerClient
}

// EnsureImage checks if the image exists locally, pulls if not
func (d *DockerClient) EnsureImage(ctx context.Context, imageName string) error {
	// Check if image exists locally
	_, _, err := d.cli.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image exists
	}

	// Try to pull the image
	fmt.Printf("  Pulling image %s...\n", imageName)
	reader, err := d.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Consume the pull output
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}

	return nil
}

// CreateContainer creates and starts a new container with resource limits
func (d *DockerClient) CreateContainer(ctx context.Context, cfg ContainerConfig) (*Container, error) {
	// Calculate resource limits
	memoryBytes := int64(cfg.Memory) * 1024 * 1024 * 1024 // Convert GB to bytes
	nanoCPUs := int64(cfg.CPUs) * 1e9                     // Docker uses nano CPUs

	// Create cpuset string (0 to CPUs-1)
	cpusetCPUs := fmt.Sprintf("0-%d", cfg.CPUs-1)
	if cfg.CPUs == 1 {
		cpusetCPUs = "0"
	}

	// Container configuration
	containerCfg := &container.Config{
		Image:      cfg.Image,
		Cmd:        []string{"sleep", "infinity"},
		WorkingDir: "/workspace",
		Tty:        false,
	}

	// Host configuration with resource limits
	hostCfg := &container.HostConfig{
		Resources: container.Resources{
			Memory:     memoryBytes,
			MemorySwap: memoryBytes, // Same as memory to disable swap
			NanoCPUs:   nanoCPUs,
			CpusetCpus: cpusetCPUs,
		},
		Binds: []string{
			fmt.Sprintf("%s:/workspace", cfg.MountPath),
		},
	}

	// Create the container
	resp, err := d.cli.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up the created container
		_ = d.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &Container{
		ID:     resp.ID,
		client: d,
	}, nil
}

// ExecResult holds the result of executing a command in a container
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Exec executes a command in the container and returns the result
func (c *Container) Exec(ctx context.Context, cmd []string, workDir string) (*ExecResult, error) {
	execCfg := container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   workDir,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := c.client.cli.ContainerExecCreate(ctx, c.ID, execCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := c.client.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()

	// Read stdout and stderr
	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read exec output: %w", err)
	}

	// Get exit code
	inspectResp, err := c.client.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return &ExecResult{
		ExitCode: inspectResp.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// ExecShell executes a shell command in the container
func (c *Container) ExecShell(ctx context.Context, command string, workDir string) (*ExecResult, error) {
	return c.Exec(ctx, []string{"bash", "-c", command}, workDir)
}

// CopyFileToContainer copies a file from the host to the container
func (c *Container) CopyFileToContainer(ctx context.Context, srcPath, dstPath string) error {
	// Read the source file
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get file info for permissions
	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create a tar archive containing the file
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	header := &tar.Header{
		Name:    filepath.Base(dstPath),
		Size:    int64(len(content)),
		Mode:    int64(fileInfo.Mode()),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("failed to write tar content: %w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Copy the tar archive to the container
	dstDir := filepath.Dir(dstPath)
	err = c.client.cli.CopyToContainer(ctx, c.ID, dstDir, &buf, container.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("failed to copy to container: %w", err)
	}

	return nil
}

// CopyFileFromContainer copies a file from the container to the host
func (c *Container) CopyFileFromContainer(ctx context.Context, srcPath, dstPath string) error {
	reader, _, err := c.client.cli.CopyFromContainer(ctx, c.ID, srcPath)
	if err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}
	defer reader.Close()

	// Extract from tar
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		if header.Typeflag == tar.TypeReg {
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				return fmt.Errorf("failed to create destination directory: %w", err)
			}

			// Write the file
			outFile, err := os.Create(dstPath)
			if err != nil {
				return fmt.Errorf("failed to create destination file: %w", err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write destination file: %w", err)
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("file not found in container: %s", srcPath)
}

// CopyDirFromContainer copies a directory from the container to the host
func (c *Container) CopyDirFromContainer(ctx context.Context, srcPath, dstPath string) error {
	reader, _, err := c.client.cli.CopyFromContainer(ctx, c.ID, srcPath)
	if err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}
	defer reader.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract from tar
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Remove the first path component (the source directory name)
		name := header.Name
		parts := strings.SplitN(name, "/", 2)
		if len(parts) > 1 {
			name = parts[1]
		} else {
			continue // Skip the root directory entry
		}

		target := filepath.Join(dstPath, name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		}
	}

	return nil
}

// Stop stops and removes the container
func (c *Container) Stop(ctx context.Context) error {
	timeout := 10 // seconds
	stopOptions := container.StopOptions{Timeout: &timeout}

	if err := c.client.cli.ContainerStop(ctx, c.ID, stopOptions); err != nil {
		// Container might already be stopped, try to remove anyway
	}

	if err := c.client.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}
