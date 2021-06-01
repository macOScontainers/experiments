package testutil

// Builds a container image by invoking the `docker build` command
func DockerBuild(tag string, directory string, dockerfile string) error {
	return Run(
		"docker", "build",
		"-t", tag,
		"-f", dockerfile,
		directory,
	)
}

// Runs a container by invoking the `docker run` command
func DockerRun(image string, command []string, options []string) error {
	dockerCmd := []string{"docker", "run", "--rm"}
	dockerCmd = append(dockerCmd, options...)
	dockerCmd = append(dockerCmd, image)
	dockerCmd = append(dockerCmd, command...)
	return Run(dockerCmd...)
}

// Stops a running container by invoking the `docker kill` command
func DockerStop(container string) error {
	return Run("docker", "kill", container)
}

// Copies files from a running container to the host filesystem by invoking the `docker cp` command
func DockerCopy(source string, dest string) error {
	return Run("docker", "cp", source, dest)
}
