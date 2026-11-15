package collectors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

type dockerActor struct {
	cfg *config.DockerSystemConfig
}

// client opens a docker client without a fixed timeout: image pulls and
// stop grace periods take longer than the collector's 10 s. The caller's
// ctx bounds every call.
func (d *dockerActor) client() (*client.Client, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	switch {
	case d.cfg.Host != "":
		opts = append(opts, client.WithHost(d.cfg.Host))
	case d.cfg.Socket != "":
		opts = append(opts, client.WithHost(d.cfg.Socket))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("docker connect: %w", err)
	}
	return cli, nil
}

func (d *dockerActor) Run(ctx context.Context, a Action) (string, error) {
	cli, err := d.client()
	if err != nil {
		return "", err
	}
	defer cli.Close()

	name := a.Target
	switch a.Name {
	case "start":
		return "", cli.ContainerStart(ctx, name, container.StartOptions{})
	case "stop":
		return "", cli.ContainerStop(ctx, name, container.StopOptions{})
	case "restart":
		return "", cli.ContainerRestart(ctx, name, container.StopOptions{})
	case "pause":
		return "", cli.ContainerPause(ctx, name)
	case "unpause":
		return "", cli.ContainerUnpause(ctx, name)
	case "kill":
		return "", cli.ContainerKill(ctx, name, "KILL")
	case "recreate":
		return recreateContainer(ctx, cli, name)
	}
	return "", fmt.Errorf("unsupported docker action %q", a.Name)
}

// recreateContainer pulls the container's image reference again and
// replaces the container with one built from the same settings, in the way
// Watchtower does. If any step after the rename fails, the old container is
// restored.
func recreateContainer(ctx context.Context, cli *client.Client, name string) (string, error) {
	old, err := cli.ContainerInspect(ctx, name)
	if err != nil {
		return "", fmt.Errorf("inspect: %w", err)
	}
	// The stop would delete an --rm container, and then nothing is left to restore.
	if old.HostConfig.AutoRemove {
		return "", fmt.Errorf("container %s was started with --rm (auto-remove) and cannot be recreated", name)
	}
	ref := old.Config.Image
	if strings.HasPrefix(ref, "sha256:") {
		return "", fmt.Errorf("container %s uses an image ID, not a pullable reference", name)
	}

	pull, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("pull %s: %w", ref, err)
	}
	err = jsonmessage.DisplayJSONMessagesStream(pull, io.Discard, 0, false, nil)
	pull.Close()
	if err != nil {
		return "", fmt.Errorf("pull %s: %w", ref, err)
	}

	oldImage, _, err := cli.ImageInspectWithRaw(ctx, old.Image)
	if err != nil {
		return "", fmt.Errorf("inspect image: %w", err)
	}

	wasRunning := old.State != nil && (old.State.Running || old.State.Paused)
	wasPaused := old.State != nil && old.State.Paused
	if wasRunning {
		if err := cli.ContainerStop(ctx, old.ID, container.StopOptions{}); err != nil {
			return "", fmt.Errorf("stop: %w", err)
		}
	}

	shortID := old.ID[:12]
	backup := fmt.Sprintf("%s-old-%s", name, shortID)
	if err := cli.ContainerRename(ctx, old.ID, backup); err != nil {
		if wasRunning {
			_ = cli.ContainerStart(context.WithoutCancel(ctx), old.ID, container.StartOptions{})
		}
		return "", fmt.Errorf("rename: %w", err)
	}

	newID := ""
	rollback := func(cause error) (string, error) {
		// Use a fresh context: the request context may be what failed.
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		defer cancel()
		errs := []error{cause}
		if newID != "" {
			if err := cli.ContainerRemove(rctx, newID, container.RemoveOptions{Force: true}); err != nil {
				errs = append(errs, fmt.Errorf("rollback remove new: %w", err))
			}
		}
		if err := cli.ContainerRename(rctx, old.ID, name); err != nil {
			errs = append(errs, fmt.Errorf("rollback rename: %w", err))
		}
		if wasRunning {
			if err := cli.ContainerStart(rctx, old.ID, container.StartOptions{}); err != nil {
				errs = append(errs, fmt.Errorf("rollback start: %w", err))
			} else if wasPaused {
				if err := cli.ContainerPause(rctx, old.ID); err != nil {
					errs = append(errs, fmt.Errorf("rollback pause: %w", err))
				}
			}
		}
		return "", errors.Join(errs...)
	}

	cfg := recreateConfig(old.Config, oldImage.Config, old.ID)
	first, rest := recreateNetworks(old.HostConfig.NetworkMode, old.NetworkSettings.Networks, shortID)
	created, err := cli.ContainerCreate(ctx, cfg, recreateHostConfig(old.HostConfig, old.Mounts), first, nil, name)
	if err != nil {
		return rollback(fmt.Errorf("create: %w", err))
	}
	newID = created.ID
	for _, netName := range slices.Sorted(maps.Keys(rest)) {
		if err := cli.NetworkConnect(ctx, netName, newID, rest[netName]); err != nil {
			return rollback(fmt.Errorf("connect network %s: %w", netName, err))
		}
	}
	if wasRunning {
		if err := cli.ContainerStart(ctx, newID, container.StartOptions{}); err != nil {
			return rollback(fmt.Errorf("start: %w", err))
		}
	}
	if wasPaused {
		if err := cli.ContainerPause(ctx, newID); err != nil {
			return rollback(fmt.Errorf("pause: %w", err))
		}
	}
	if err := cli.ContainerRemove(ctx, old.ID, container.RemoveOptions{}); err != nil {
		return rollback(fmt.Errorf("remove old: %w", err))
	}
	return fmt.Sprintf("recreated %s from %s (%s)", name, ref, newID[:12]), nil
}

// recreateConfig copies the container config and drops every value that
// only came from the old image, so the new image's defaults apply. Values
// set for the container (compose labels, env, command) are kept.
func recreateConfig(cur, img *container.Config, id string) *container.Config {
	cfg := *cur
	if img == nil {
		img = &container.Config{}
	}
	if len(id) >= 12 && cfg.Hostname == id[:12] {
		cfg.Hostname = ""
	}
	cfg.Env = slices.DeleteFunc(slices.Clone(cur.Env), func(e string) bool { return slices.Contains(img.Env, e) })
	if cur.Labels != nil {
		cfg.Labels = maps.Clone(cur.Labels)
		maps.DeleteFunc(cfg.Labels, func(k, v string) bool { iv, ok := img.Labels[k]; return ok && iv == v })
	}
	if slices.Equal(cur.Cmd, img.Cmd) {
		cfg.Cmd = nil
	}
	if slices.Equal(cur.Entrypoint, img.Entrypoint) {
		cfg.Entrypoint = nil
	}
	if cur.WorkingDir == img.WorkingDir {
		cfg.WorkingDir = ""
	}
	if cur.User == img.User {
		cfg.User = ""
	}
	if cur.StopSignal == img.StopSignal {
		cfg.StopSignal = ""
	}
	if reflect.DeepEqual(cur.Healthcheck, img.Healthcheck) {
		cfg.Healthcheck = nil
	}
	if cur.ExposedPorts != nil {
		cfg.ExposedPorts = maps.Clone(cur.ExposedPorts)
		for p := range img.ExposedPorts {
			delete(cfg.ExposedPorts, p)
		}
	}
	if cur.Volumes != nil {
		cfg.Volumes = maps.Clone(cur.Volumes)
		for v := range img.Volumes {
			delete(cfg.Volumes, v)
		}
	}
	return &cfg
}

// recreateHostConfig copies the host config and adds a bind for every
// volume mount that HostConfig does not name: the anonymous volumes from an
// image VOLUME or a bare `-v /path`. The new container then keeps their data,
// as docker compose does.
func recreateHostConfig(cur *container.HostConfig, mounts []types.MountPoint) *container.HostConfig {
	hc := *cur
	hc.Binds = slices.Clone(cur.Binds)
	named := make(map[string]bool, len(cur.Binds)+len(cur.Mounts))
	for _, b := range cur.Binds {
		if parts := strings.Split(b, ":"); len(parts) >= 2 {
			named[parts[1]] = true
		}
	}
	for _, m := range cur.Mounts {
		named[m.Target] = true
	}
	for _, m := range mounts {
		if m.Type != mount.TypeVolume || m.Name == "" || named[m.Destination] {
			continue
		}
		bind := m.Name + ":" + m.Destination
		if !m.RW {
			bind += ":ro"
		}
		hc.Binds = append(hc.Binds, bind)
	}
	return &hc
}

// recreateNetworks splits the old container's networks into the one passed
// at create (the network of its NetworkMode if attached, else the first by
// name) and the rest, connected afterwards. Only user settings are copied;
// runtime data (IP, endpoint ID) and the old ID alias are dropped.
func recreateNetworks(mode container.NetworkMode, nets map[string]*network.EndpointSettings, shortID string) (*network.NetworkingConfig, map[string]*network.EndpointSettings) {
	rest := make(map[string]*network.EndpointSettings, len(nets))
	for netName, ep := range nets {
		rest[netName] = &network.EndpointSettings{
			IPAMConfig: ep.IPAMConfig,
			Links:      ep.Links,
			Aliases:    slices.DeleteFunc(slices.Clone(ep.Aliases), func(a string) bool { return a == shortID }),
			DriverOpts: ep.DriverOpts,
		}
	}
	first := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}
	if len(rest) == 0 {
		return first, rest
	}
	primary := string(mode)
	if _, ok := rest[primary]; !ok {
		primary = slices.Sorted(maps.Keys(rest))[0]
	}
	first.EndpointsConfig[primary] = rest[primary]
	delete(rest, primary)
	return first, rest
}

// Logs returns the last tail lines of a container's stdout and stderr in
// order. Without a TTY the stream is multiplexed and demuxed with stdcopy.
func (d *dockerActor) Logs(ctx context.Context, name string, tail int) ([]models.LogLine, error) {
	cli, err := d.client()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	info, err := cli.ContainerInspect(ctx, name)
	if err != nil {
		return nil, err
	}
	rc, err := cli.ContainerLogs(ctx, name, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	})
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var lines []models.LogLine
	stdout := &lineWriter{stream: "stdout", out: &lines}
	stderr := &lineWriter{stream: "stderr", out: &lines}
	if info.Config != nil && info.Config.Tty {
		_, err = io.Copy(stdout, rc)
	} else {
		_, err = stdcopy.StdCopy(stdout, stderr, rc)
	}
	if err != nil {
		return nil, err
	}
	stdout.flush()
	stderr.flush()
	if lines == nil {
		lines = []models.LogLine{}
	}
	return lines, nil
}

// lineWriter splits one stream into lines and appends them to a list
// shared with the other stream, so the output keeps arrival order.
type lineWriter struct {
	stream  string
	out     *[]models.LogLine
	pending []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
	for {
		i := bytes.IndexByte(w.pending, '\n')
		if i < 0 {
			return len(p), nil
		}
		w.emit(w.pending[:i])
		w.pending = w.pending[i+1:]
	}
}

func (w *lineWriter) flush() {
	if len(w.pending) > 0 {
		w.emit(w.pending)
		w.pending = nil
	}
}

func (w *lineWriter) emit(line []byte) {
	*w.out = append(*w.out, models.LogLine{Stream: w.stream, Text: strings.TrimRight(string(line), "\r")})
}
